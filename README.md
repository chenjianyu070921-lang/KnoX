# KnoX 企业级智能知识库后端服务

## 1. 项目介绍

KnoX 是一个基于 **go-zero** 微服务架构的企业级智能知识库后端，提供文档管理、向量检索、RAG 问答、Agent 对话和运营统计能力。

核心业务链路：

- 文档上传：七牛云对象存储 + MySQL 元数据管理，携带 `X-Request-Id` 实现幂等上传。
- 异步向量化：Kafka 异步消息驱动，Consumer 分块、Embedding 后写入 Milvus，并通过消费记录与分布式锁保证只处理一次。
- 知识检索：Ollama bge-m3 生成向量，Milvus COSINE 相似度检索。
- Agent 问答：Eino ReAct Agent 调用知识检索工具，使用火山方舟 ARK 大模型生成回答，SSE 流式输出。
- 会话记忆：Redis 保存多轮对话历史，首轮返回 `sessionId` 用于续接。
- 运营观测：ClickHouse 记录搜索、对话、上传日志，提供性能大盘与慢查询接口。
- 高可用保护：接口级限流、下游熔断、Agent 总超时降级、统一错误码。

## 2. 技术栈清单

| 分类 | 技术 |
| --- | --- |
| 语言 | Go 1.26.5 |
| Web 框架 | go-zero 1.10.2（REST + 内置中间件） |
| 数据库 | MySQL 8.0（GORM） |
| 缓存 | Redis 7.x（go-redis + go-zero Redis） |
| 消息队列 | Kafka 4.3.1（IBM Sarama） |
| 向量数据库 | Milvus 2.6.8（HNSW + COSINE） |
| 对象存储 | 七牛云（Qiniu SDK） |
| Embedding | Ollama bge-m3（1024 维） |
| LLM | 火山方舟 ARK（OpenAI 兼容 API） |
| LLM 框架 | CloudWeGo Eino（Splitter / Indexer / Retriever / ReAct Agent） |
| 统计分析 | ClickHouse 24.3（MergeTree） |
| 部署 | Docker Compose |

## 3. 环境依赖

| 依赖 | 说明 |
| --- | --- |
| Go | `go.mod` 要求 Go 1.26.5 或更高 |
| Docker & Docker Compose | 启动 MySQL / Redis / Kafka / Milvus / etcd / MinIO / ClickHouse |
| Ollama | 本地运行 `bge-m3` 模型，监听 `127.0.0.1:11434` |
| 火山方舟 ARK | 对话大模型 API Key 与 Endpoint ID |
| 七牛云 | AccessKey、SecretKey、Bucket、Region、外链 Domain |

### 3.1 环境变量（密钥）

密钥不写入任何配置文件，启动前通过环境变量注入：

| 环境变量 | 说明 |
| --- | --- |
| `KNOX_QINIU_ACCESS_KEY` | 七牛云 AccessKey |
| `KNOX_QINIU_SECRET_KEY` | 七牛云 SecretKey |
| `KNOX_ARK_API_KEY` | 火山方舟 API Key |
| `KNOX_MYSQL_DSN` | MySQL 连接串（可选覆盖） |
| `KNOX_CLICKHOUSE_PASSWORD` | ClickHouse 密码（可选覆盖） |
| `KNOX_CLICKHOUSE_DSN` | Reporter 的 ClickHouse 连接串（可选覆盖） |

变量模板见 `.env.example`。PowerShell 启动前示例：

```powershell
$env:KNOX_QINIU_ACCESS_KEY="..."
$env:KNOX_QINIU_SECRET_KEY="..."
$env:KNOX_ARK_API_KEY="..."
make run
```

## 4. 一键部署命令

启动全部基础设施：

```shell
docker compose -f docker-compose/infra.yml up -d
```

准备 Ollama Embedding 模型：

```shell
ollama pull bge-m3
ollama serve
```

启动异步消费者（文档向量化）：

```shell
go run ./cmd/consumer -f cmd/consumer/etc/config.yaml
```

启动 API 网关：

```shell
go run ./cmd/gateway -f cmd/gateway/etc/doc.yaml
```

启动统计大盘服务（前端大盘页依赖）：

```shell
go run ./cmd/reporter -f cmd/reporter/etc/reporter.yaml
```

或使用 Makefile：

```shell
make run
```

## 5. 目录结构说明

```text
KnoX/
├── api/                          # goctl API 定义
│   └── doc.api
├── cmd/
│   ├── gateway/                  # API 网关
│   │   ├── etc/doc.yaml          # 网关配置
│   │   └── internal/
│   │       ├── config/           # 配置结构体
│   │       ├── handler/          # HTTP Handler
│   │       ├── logic/            # 业务逻辑（上传/搜索/对话/统计）
│   │       ├── middleware/       # 限流中间件
│   │       ├── svc/              # 依赖注入
│   │       └── types/            # 请求响应结构
│   ├── consumer/                 # Kafka 异步消费者
│   │   ├── etc/config.yaml       # 消费者配置
│   │   └── internal/config/
│   └── reporter/                 # 统计大盘服务
│       ├── etc/reporter.yaml     # 大盘配置
│       └── internal/
├── docker-compose/
│   └── infra.yml                 # 基础设施编排
├── internal/
│   ├── agent/                    # Eino ReAct Agent + 知识检索工具
│   ├── analytics/                # ClickHouse 统计组件与查询
│   ├── breaker/                  # 下游熔断封装
│   ├── errcode/                  # 统一错误码
│   ├── model/                    # GORM 模型（Document / ConsumeRecord）
│   ├── repository/               # 数据访问层
│   ├── session/                  # Redis 会话历史
│   └── vector/                   # 分块/Embedding/索引/检索
├── pkg/
│   ├── qiniuyun/                 # 七牛云上传
│   ├── clickhouse/               # ClickHouse 连接
│   ├── database/                 # MySQL 连接
│   └── redisx/                   # Redis 连接与分布式锁
├── knox-docs/                    # 人物设定与知识库语料
├── go.mod
├── Makefile
└── README.md
```

## 6. 配置参数说明

### 网关配置 `cmd/gateway/etc/doc.yaml`

| 配置项 | 说明 |
| --- | --- |
| `Timeout` | 请求级超时（毫秒），默认 120000 |
| `MaxConns` | 最大并发连接数，默认 1000 |
| `MaxBytes` | 请求体上限，默认 10MB |
| `CpuThreshold` | CPU 过载保护阈值（千分比），默认 900 |
| `MySQL.DSN` | MySQL 连接串 |
| `Redis.Addr` | Redis 地址 |
| `Kafka.Brokers / Topic` | Kafka 地址与上传消息 Topic |
| `Ollama.URL / Model / Dimension` | Embedding 服务地址、模型与向量维度 |
| `Milvus.Addr / DBName / Collection / VectorField` | Milvus 地址、库名、集合名与向量字段 |
| `Qiniu.*` | 七牛云 AccessKey / SecretKey / Bucket / Region / Domain |
| `ARK.APIKey / ModelID / BaseURL` | 火山方舟对话模型配置 |
| `RateLimit.Chat / Upload / Search` | 各接口每秒配额（`Quota` / `Period`） |
| `Retrieval.DefaultTopK / MaxTopK` | 检索默认返回条数与单次上限 |
| `ClickHouse.*` | ClickHouse 地址、库名、账号密码 |

### 消费者配置 `cmd/consumer/etc/config.yaml`

| 配置项 | 说明 |
| --- | --- |
| `MySQL.DSN` | MySQL 连接串 |
| `Redis.Addr` | Redis 地址 |
| `Kafka.Brokers / Topic / Group` | Kafka 消费配置 |
| `Ollama.URL / Model / Dimension` | Embedding 服务地址、模型与向量维度 |
| `Milvus.Addr / DBName / Collection / VectorField` | Milvus 地址、库名、集合名与向量字段 |
| `Consumer.HandleTimeout` | 单条消息处理超时，默认 60s |
| `Consumer.LockTTL` | 文档索引锁 TTL，默认 90s，需大于 HandleTimeout |
| `Consumer.LockMaxRetries` | 失败最大重试次数，默认 3，达到后停止重投 |

> 敏感信息（七牛云密钥、ARK API Key）建议改为环境变量注入，不要写入版本库。

## 7. AI 模块说明

### RAG 检索链路

```text
上传文档
  → Markdown 标题分块（Eino HeaderSplitter）
  → Ollama bge-m3 Embedding
  → Milvus knox_docs 集合（HNSW + COSINE）
  → Eino Retriever 语义检索
```

### Agent 问答链路

```text
用户问题
  → Redis 会话历史 + System Prompt
  → Eino ReAct Agent
  → knowledge_search 工具（Milvus 检索）
  → ARK 大模型生成回答
  → SSE 流式返回 content / sessionId / [DONE]
```

### 对话接口

```shell
curl -N -X POST http://127.0.0.1:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"question":"林微喜欢什么花"}'
```

首次请求返回 `sessionId`，后续请求携带即可保持上下文：

```json
{"question":"为什么喜欢","sessionId":"abc-123"}
```

### 统计接口

统计接口由两个服务提供：Gateway（`8080`）提供明细/趋势/慢查询，Reporter（`8081`）提供前端大盘聚合数据。

| 服务 | 接口 | 说明 |
| --- | --- | --- |
| Gateway | `GET http://127.0.0.1:8080/api/v1/analytics/overview` | 最近 24h 总数、平均耗时、最大耗时、错误率 |
| Gateway | `GET http://127.0.0.1:8080/api/v1/analytics/trends?days=7` | 按天趋势与 p95 |
| Gateway | `GET http://127.0.0.1:8080/api/v1/analytics/slow-queries?limit=20` | 耗时最长的操作日志 |
| Reporter | `GET http://127.0.0.1:8081/api/v1/analytics/dashboard` | 大盘聚合：24h 趋势、事件分布、P50/P95/P99、7 天成功率 |

### 统一错误码

| 错误码 | 说明 |
| --- | --- |
| 0 | 成功 |
| 1000 | 请求参数错误 |
| 1001 | 参数校验失败 |
| 1003 | 触发限流 |
| 1004 | 下游熔断 |
| 1005 | 内部错误 |
| 2001 | 文档不存在 |
| 2002 | 不支持的文档类型 |
| 2003 | 文档上传失败 |
