# 配置死项、参数校验、会话预算修复方案

日期：2026-08-06

本文记录三个问题的修复方案与验证结果，不改动现有分层架构。

## 1. 配置死项与硬编码

- Gateway 配置补上 `Ollama`（`URL`/`Model`/`Dimension`）、`Milvus`（`Addr`/`DBName`/`Collection`/`VectorField`）和 `Retrieval`（`DefaultTopK`/`MaxTopK`）字段，`doc.yaml` 里的配置从“写了不生效”变为实际读取。
- Consumer 配置补上 `Ollama.Dimension`、`Milvus.DBName/Collection/VectorField`，消费者初始化时真正使用配置值。
- `internal/vector` 包改为配置注入：`GetEmbeddingClient(ctx, baseURL, model)`、`GetMilvusClient(ctx, addr, dbName)`、`RetrieverClient(ctx, ..., collection, vectorField, topK)`、`IndexerClient(ctx, ..., collection, vectorField, dimension)`，删除硬编码的 `localhost:11434`、`127.0.0.1:19530`、`knox_docs`、`vector`、`1024`、`TopK: 5`。
- 客户端统一用 `sync.Once` 单例并返回 error，避免初始化失败后返回 nil 客户端。
- `Config.SetDefaults()` 提供缺省值（地址、模型、维度、集合、TopK），Gateway 与 Consumer 启动时自动归一化。
- 示例配置（`doc.yaml.example`、`config.yaml.example`）与 README 配置表同步更新。

## 2. 参数校验

- Chat：`question` 去空格后不能为空、不超过 4000 字符；`sessionId` 不超过 64 字符。
- Search：`query` 去空格后不能为空、不超过 500 字符；`topK` 使用配置的默认值与上限归一化。
- Analytics：`days` 缺省 7、上限 90；`limit` 缺省 20、上限 100，防止超大参数放大聚合查询。
- 校验失败返回统一业务错误码 `1001`（参数校验失败）。

## 3. 会话历史与 token 预算

- `internal/session` 增加历史裁剪：最多保留 20 条消息、总内容不超过 12000 字符（按字符数近似 token 预算）。
- 保存会话前与读取历史时都会裁剪，始终保留最新消息；旧会话中已超限的历史也会被收敛。
- 对话上下文 = System Prompt + 有界历史 + 当前问题，长对话不再无限累积。

## 测试与验证

- 新增 Gateway/Consumer 配置默认值测试（`config_test.go`）。
- 新增 Chat/Search/Analytics 校验与归一化助手测试。
- 新增 Session 裁剪测试（条数上限、token 预算、超大最新消息保留）。
- 验证命令：`go test ./...`、`go build ./...`、`go vet ./...`，全部通过。
