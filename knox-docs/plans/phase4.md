# Phase 4 可靠性与可观测性设计

## 背景

KnoX 目前完成了 P1-P3：文档上传、七牛云存储、Kafka 异步向量化、Milvus 检索、Agent 对话与 SSE 流式输出。Phase 4 的目标是补齐生产化短板：重复请求、重复消费、下游故障、超时无兜底、无统计数据、无链路追踪。

## 目标

1. 上传接口对同一 `RequestId` 只处理一次。
2. Kafka 消息重复投递、消费者重启后不会重复建索引。
3. 高并发下网关限流、熔断、超时降级，下游不被打爆。
4. 搜索、对话、上传、消费的关键指标写入 ClickHouse，可查询性能大盘。
5. Gateway 与 Consumer 的请求链路可被 Jaeger 追踪。

## 范围

Phase 4 分为三组：

- A. 幂等三件套：上传幂等、Redis 分布式锁、Kafka 消费幂等。
- B. 保护能力：限流/熔断、Agent 总超时与降级回答。
- C. 可观测性：ClickHouse 统计、Jaeger 链路追踪。

本设计文档覆盖 A/B/C 的总体方案；A 组已有独立实现计划，B/C 组在后续计划中展开。

## 总体架构

```text
客户端
  │ X-Request-Id
  ▼
Gateway (go-zero)
  ├─ 幂等：Redis SET NX EX + Document.RequestID 唯一索引
  ├─ 保护：MaxConns/Breaker/Shedding + PeriodLimit + AgentTimeout
  ├─ 可观测：Telemetry(OTLP) + 异步写 ClickHouse
  ├─ POST /doc/upload ──► Qiniu ──► MySQL ──► Kafka
  ├─ POST /search ──► Milvus
  ├─ POST /chat(SSE) ──► Eino ReAct Agent ──► ARK
  └─ GET /analytics/* ──► ClickHouse

Consumer
  Kafka ──► consume_record 查重 ──► Redis 锁 ──► Eino Indexer ──► Milvus
             失败记录 failed，成功记录 done，只有成功才 MarkMessage
```

## A. 幂等三件套设计

### A1. 上传幂等（RequestId）

接口约定：`POST /api/v1/doc/upload` 必须携带 `X-Request-Id`，缺失返回业务码 `1000`。

流程：

1. 生成随机 `token`（UUID）。
2. Redis `SET idem:upload:{requestId} {token} NX EX {ttl}`：
   - 成功：继续上传。
   - 失败：说明已有请求在处理，查 `Document.RequestID`，命中则返回已有 `docId/url/version`，未命中返回“请求处理中，请稍后重试”。
3. 上传七牛云，写入 `Document`（含 `RequestID`、`URL`），发送 Kafka。
4. 任一步失败时用 Lua 原子释放占位，允许客户端重试。
5. 数据库唯一索引兜底：若 Redis 丢失但 DB 已有同 `RequestID`，捕获 `gorm.ErrDuplicatedKey` 后返回已有记录。

数据变更：

- `Document` 增加 `RequestID *string`（唯一索引，允许 NULL，兼容存量数据）。
- `Document` 增加 `URL string`，重复请求直接返回原 URL。

### A2. Redis 分布式锁

新建 `pkg/redisx/distlock`：

- `TryLock(ctx, key, token, ttl)`：`SET key token NX PX ttl`。
- `Unlock(ctx, key, token)`：Lua `if GET(key)==token then DEL(key) end`，防止误删他人锁。

锁用途：Consumer 按 `lock:embed:{docId}` 加锁，保证同一文档同一时间只建一次索引；并发重复投递时只有一个实例真正执行，天然防缓存击穿。

### A3. Kafka 消费幂等

新增 `consume_record` 表，唯一键 `(topic, partition, offset)`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint PK | 主键 |
| topic | string | Kafka topic |
| partition | int32 | 分区 |
| offset | int64 | 偏移 |
| doc_id | string | 文档 ID |
| status | string | processing / done / failed |
| message | text | 失败原因 |
| retry_count | int | 重试次数 |
| created_at / updated_at | time | 时间 |

消费流程：

1. 解析消息，查 `consume_record`：`done` 直接跳过。
2. 按 `docId` 加 Redis 锁，失败重试 3 次后返回 `ErrLockBusy`（不 MarkMessage，等待重投）。
3. 锁内复查记录；不存在则 `INSERT ... ON DUPLICATE KEY DO NOTHING` 插入 `processing`。
4. 建索引成功 → 更新 `done`；失败 → 更新 `failed` 并返回错误。
5. 只有处理成功才 `session.MarkMessage(msg, "")`。

Consumer 改造：

- `main.go` 用 `conf.MustLoad` 读取 `-f` 指定的 `etc/config.yaml`，不再硬编码。
- 连接 Redis，`AutoMigrate(&model.ConsumeRecord{})`。
- `OffsetOldest`，保证重启后从最早消费，重复靠记录去重。

## B. 保护能力设计

### B1. 限流/熔断

- Gateway `doc.yaml` 调优：`MaxConns`（如 1000）、`CpuThreshold`、`MaxBytes`。
- 启用 rest 内置中间件：`Breaker`、`Shedding`、`Timeout`、`Recover`（默认已开启，显式确认）。
- 自定义 `RateLimitMiddleware`：使用 go-zero `core/limit.PeriodLimit`（Redis Lua）按 upload / search / chat 分接口配额，超限返回 `429`。
- 下游 ARK、Milvus 调用用 `core/breaker` 包裹，故障快速失败。

### B2. 超时兜底

- 配置 `AgentTimeout`（默认 30s）。
- Chat 用 `context.WithTimeout` 包裹 Agent；超时/错误时返回可配置降级文案，SSE 正常发 `[DONE]`，响应带 `degraded: true`。
- 降级回答不写入会话历史。
- Consumer 单条消息处理超时 `HandleTimeout` 配置化。

## C. 可观测性设计

### C1. ClickHouse 统计

- docker-compose 增加 ClickHouse。
- 新建 `internal/analytics`：客户端、建表、异步批量写入（满 100 条或 5s flush）。
- 表：`search_logs`、`chat_logs`、`upload_logs`、`consumer_logs`，均含 `duration_ms`、`success`、`trace_id`。
- Gateway 只读接口：
  - `GET /api/v1/analytics/overview`：总数、均值、p95、错误率。
  - `GET /api/v1/analytics/trends`：按天趋势（7/30 天）。
  - `GET /api/v1/analytics/slow-queries`：耗时最长的查询日志。
- ClickHouse 不可用时只记日志，不阻塞业务。

### C2. 链路追踪

- docker-compose 增加 Jaeger all-in-one（OTLP 4317、UI 16686）。
- Gateway `Telemetry` 配置 `otlpgrpc` → `127.0.0.1:4317`。
- Consumer 也启动 trace agent。
- Gateway 发 Kafka 时把 `traceparent/tracestate` 写入消息头，Consumer 续接 span。
- 关键步骤加 span：embedding、检索、LLM、文档索引。

## 错误处理

- 幂等占位失败：业务码 `1000`（参数）或 `2003`（上传失败），重复且无结果时提示“请求处理中”。
- 限流：HTTP 429 + 统一 JSON。
- 熔断：503 + 统一 JSON。
- Agent 超时：SSE 降级回答 + `degraded: true`，不报错中断。
- Consumer 失败：`consume_record.status = failed`，不 MarkMessage。

## 测试策略

- 单测（miniredis）：distlock 加锁/解锁/过期/误删防护；上传幂等存储 acquire/release。
- 单测（fake store/lock）：Processor 去重、失败重试、锁忙、失败标记。
- 集成测试（`//go:build integration`，需 MySQL）：Document RequestID 唯一约束、ConsumeRecord 唯一键与状态流转。
- 端到端：docker-compose 起 MySQL/Redis/Kafka/Milvus/ClickHouse/Jaeger，上传同一 RequestId 两次验证只入库一次；重复投递 Kafka 消息验证只建一次索引。

## 验收标准

1. 同一 `X-Request-Id` 重复上传返回同一 `docId/url/version`，数据库仅一条记录。
2. 消费者重启或重复投递同一条 Kafka 消息，`consume_record` 不重复、Milvus 不重复建索引。
3. 超限/熔断返回统一错误；Chat 超时返回降级回答且 SSE 正常结束。
4. ClickHouse 可查到搜索/对话/上传/消费日志；analytics 接口有数据。
5. Jaeger 可看到 Gateway 与 Consumer 的链路。

## 非目标

- 不做用户认证/权限。
- 不做消息补偿队列。
- 不做 ClickHouse 数据清洗与聚合任务。
