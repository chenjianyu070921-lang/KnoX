# KnoX — 企业级智能知识库后端服务

基于 **go-zero** 微服务架构的企业知识库后端，融合文档管理、向量检索与 Agent 智能问答。

> 项目定位：将 Go 后端工程深度与 LLM 应用落地拧成一条主线。

---

## 功能特性

| 阶段 | 模块 | 技术选型 | 说明 |
|:----:|------|---------|------|
| P1 | API 网关 | go-zero | 限流、熔断、统一错误处理 |
| P1 | 文档管理 | GORM + MySQL | 文件上传、版本控制（乐观锁） |
| P1 | 统一异常码 | 自定义 errcode | 业务码 + 统一 JSON 响应 |
| P1 | 缓存层 | Redis | 会话存储、分布式锁 |
| P1 | 基础设施 | Docker Compose | 一键启动全部依赖 |
| P2 | 文件上传 | 七牛云 | 文档存储到 CDN |
| P2 | 异步处理 | Kafka + Sarama | 文档上传后异步向量化 |
| P2 | 文档分块 | Eino Markdown Splitter | 按标题结构语义化分块 |
| P2 | 向量检索 | Milvus + Eino | 语义检索，COSINE 距离 |
| P2 | 本地 Embedding | Ollama bge-m3 | 1024 维本地向量化 |
| P3 | Agent 问答 | Eino ReAct Agent | ReAct 循环 + Tool Calling |
| P3 | 流式响应 | SSE | 逐 token 推送 |
| P3 | 对话历史 | Redis Session | 跨轮次上下文记忆 |
| P3 | 聊天模型 | 火山方舟 ARK | OpenAI 兼容 API |

---

## 架构

```
用户
  │
  ▼
API Gateway (go-zero :8080)
  │
  ├── POST /api/v1/doc/upload ──→ 七牛云 CDN
  │                              └── MySQL 存储元信息
  │                              └── Kafka 异步通知
  │
  ├── POST /api/v1/search  ──→ Embedding(bge-m3)
  │                           └── Milvus 向量检索
  │
  ├── POST /api/v1/chat (SSE) ──→ ReAct Agent
  │                              ├── knowledge_search 工具 → Milvus
  │                              └── ARK 大模型生成回答
  │                              └── Redis 会话历史
  │
  └── GET /api/v1/ping

Consumer (独立进程)
  Kafka ──→ Eino Indexer ──→ Milvus 向量入库
```

---

## 快速开始

### 前置要求

| 依赖 | 用途 |
|------|------|
| Go ≥ 1.21 | 编译运行 |
| Docker & Docker Compose | 启动依赖服务 |
| Ollama | 本地 Embedding（bge-m3） |
| 火山方舟 ARK API Key | 大模型对话 |

### 启动步骤

```bash
# 1. 启动全部依赖（MySQL + Redis + Kafka + Milvus）
docker compose -f docker-compose/infra.yml up -d

# 2. 启动 Ollama 本地模型
ollama pull bge-m3
ollama serve

# 3. 创建 ARK API Key 并设置到配置
# 编辑 cmd/gateway/etc/doc.yaml 中的 ARK 配置

# 4. 启动消费者（文档异步向量化）
go run ./cmd/consumer

# 5. 启动 API 服务（新开一个终端）
make run

# 6.（可选）上传示例文档测试
#   用 Apipost 上传 md/txt/pdf 文件到 /api/v1/doc/upload
```

---

## API 文档

### 路由一览

| 方法 | 路径 | 说明 |
|:----:|------|------|
| GET | `/api/v1/ping` | 健康检查 |
| POST | `/api/v1/doc/upload` | 上传文档（multipart/form-data，字段名 `file`） |
| POST | `/api/v1/search` | 向量搜索知识库 |
| POST | `/api/v1/chat` | Agent 问答（支持 SSE 流式和 JSON 两种响应） |

### 上传文档

```
POST /api/v1/doc/upload
Content-Type: multipart/form-data
Body: file = (选择 .md / .txt / .pdf 文件)
```

响应：
```json
{"docId":"doc_20260730xxxxxx","url":"http://...","version":1}
```

### 知识库搜索

```
POST /api/v1/search
Content-Type: application/json
{"query":"林微喜欢什么花","topK":5}
```

响应：
```json
{"results":[{"docId":"doc_xxx","content":"她喜欢白色的花，尤其是栀子花……","score":0.85}]}
```

### Agent 聊天（SSE 流式）

```
POST /api/v1/chat
Content-Type: application/json
{"question":"你最喜欢看什么书"}
```

响应（SSE 逐 token）：
```
data: {"content":"我"}
data: {"content":"最近在读"}
data: {"content":"村上春树……"}
data: [DONE]
```

第一次请求会返回 `sessionId`，后续带上可保持对话历史：
```
{"question":"为什么喜欢","sessionId":"abc-123"}
```

### 统一响应格式（非 SSE 接口）

```json
{"code":0,"message":"success","data":{...}}
```

### 错误码

| 错误码 | 说明 |
|:------:|------|
| 0 | 成功 |
| 1000 | 请求参数错误 |
| 1001 | 参数校验失败 |
| 2001 | 文档不存在 |
| 2002 | 不支持的文档类型 |
| 2003 | 文档上传失败 |
| -1 | 系统错误 |

---

## 项目结构

```
KnoX/
├── api/                          # API 定义 (.api)
│   └── doc.api
├── cmd/
│   ├── gateway/                  # API 网关服务
│   │   ├── doc.go
│   │   ├── etc/doc.yaml          # 配置
│   │   └── internal/
│   │       ├── config/           # 配置结构体
│   │       ├── handler/          # HTTP Handler
│   │       ├── logic/            # 业务逻辑
│   │       ├── svc/              # 依赖注入
│   │       └── types/            # 请求响应结构
│   └── consumer/                 # 异步消费者服务
│       ├── main.go
│       ├── handler.go
│       └── etc/config.yaml
├── docker-compose/
│   └── infra.yml                 # 全部依赖编排
├── internal/
│   ├── agent/                    # Eino ReAct Agent
│   ├── errcode/                  # 异常码体系
│   ├── model/                    # 数据模型
│   ├── repository/               # 数据访问层
│   ├── session/                  # 会话历史（Redis）
│   └── vector/                   # 向量检索组件
│       ├── chunker_markdown.go   # Markdown 分块
│       ├── embedding.go          # Ollama Embedding
│       ├── indexer.go            # Eino Indexer
│       ├── milvus.go             # Milvus 客户端
│       └── retriever.go          # Eino Retriever
├── pkg/
│   ├── QiniuYun/                 # 七牛云上传
│   ├── database/                 # MySQL 连接
│   └── redisx/                   # Redis 连接
├── go.mod
├── Makefile
└── README.md
```

---

## 开发

```bash
# 修改 .api 文件后重新生成代码
make goctl

# 启动 gateway
make run

# 启动消费者（需新开终端）
go run ./cmd/consumer
```

---

## 技术栈

| 分类 | 技术 |
|------|------|
| 框架 | go-zero, Eino |
| 数据库 | MySQL 8.0, Redis 7.x, Milvus 2.5 |
| 消息队列 | Kafka 3.7 |
| 对象存储 | 七牛云 |
| 搜索 | Milvus 向量检索 |
| AI 模型 | Ollama bge-m3（Embedding）, 火山方舟 ARK（对话） |
| LLM 框架 | Eino（React Agent, Indexer, Retriever, Splitter） |

---

## License

仅供学习交流。
