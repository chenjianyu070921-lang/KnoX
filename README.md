# KnoX — 企业级智能知识库后端服务

基于 **go-zero** 微服务架构的企业知识库后端，融合文档管理、全文检索与 Agent 智能问答能力。

> 项目定位：将 Go 后端工程深度与 LLM 应用落地拧成一条主线。

---

## 功能特性

### 当前（Phase 1）

| 模块 | 技术选型 | 说明 |
|------|---------|------|
| **API 网关** | go-zero | 限流、熔断、统一错误处理 |
| **文档管理** | GORM + MySQL | 文档上传、版本控制（乐观锁） |
| **统一异常码** | 自定义 errcode | 业务码 + 统一 JSON 响应 |
| **缓存层** | Redis | 全局 Redis 客户端已就绪 |
| **基础设施** | Docker Compose | 一键启动 MySQL + Redis |

### 规划中

- **全文检索**：Elasticsearch + Milvus 向量检索
- **Agent 问答**：Eino 多 Agent 编排、SSE 流式响应
- **异步处理**：Kafka 消息队列
- **监控统计**：ClickHouse + Jaeger 链路追踪

---

## 快速开始

### 前置要求

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | ≥ 1.21 | 编译运行 |
| Docker & Docker Compose | 任意 | 启动 MySQL / Redis |
| MySQL 8.0 | — | 文档数据存储 |
| Redis 7.x | — | 缓存与分布式锁 |

### 启动步骤

```bash
# 1. 克隆项目
git clone <your-repo-url> && cd KnoX

# 2. 启动基础设施（MySQL + Redis）
docker compose -f docker-compose/infra.yml up -d

# 3. 启动 API 服务
make run
# 或: go run ./cmd/gateway -f cmd/gateway/etc/doc.yaml
```

### 验证

```bash
# 健康检查
curl http://127.0.0.1:8080/api/v1/ping
# → {"message":"pong"}

# 上传文档
curl -X POST http://127.0.0.1:8080/api/v1/doc/upload \
  -H "Content-Type: application/json" \
  -d '{"content":"这是文档内容","fileName":"test.md","docType":"md"}'
# → {"docId":"doc_20260728xxxxxx","version":1,"code":0,"message":"success"}
```

---

## 项目结构

```
KnoX/
├── api/                          # API 定义文件 (.api)
│   └── doc.api                   #   文档服务接口定义
├── cmd/                          # 各服务入口
│   └── gateway/                  #   API 网关
│       ├── doc.go                #   主入口
│       ├── etc/doc.yaml          #   配置文件
│       └── internal/
│           ├── config/           #   配置结构体
│           ├── handler/          #   HTTP Handler（goctl 生成）
│           ├── logic/            #   业务逻辑
│           ├── svc/              #   服务上下文（依赖注入）
│           └── types/            #   请求/响应结构体
├── docker-compose/
│   └── infra.yml                 #  MySQL + Redis 编排
├── internal/
│   ├── errcode/                  #  统一异常码体系
│   ├── model/                    #  数据模型（GORM）
│   └── repository/               #  数据访问层
├── pkg/
│   ├── database/                 #  MySQL 连接工具
│   └── redisx/                   #  Redis 连接工具
├── go.mod
├── Makefile
└── README.md
```

---

## API 文档

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ping` | 健康检查 |
| POST | `/api/v1/doc/upload` | 上传文档 |

### 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1000 | 请求参数错误 |
| 1001 | 参数校验失败 |
| 2001 | 文档不存在 |
| 2002 | 不支持的文档类型 |
| 2003 | 文档上传失败 |

---

## 开发

```bash
# API 定义修改后重新生成代码
make goctl

# 运行服务
make run
```

---

## 技术栈

**后端框架**：go-zero  
**数据库**：MySQL 8.0  
**缓存**：Redis 7.x  
**ORM**：GORM  
**开发工具**：Goctl 1.10.1

---

## License

仅供学习交流。
