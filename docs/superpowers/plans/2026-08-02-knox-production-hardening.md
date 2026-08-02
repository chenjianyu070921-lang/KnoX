# KnoX Production Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 KnoX 从“能跑的个人项目”提升到“生产级可面试项目”，完成 P0 安全与可靠性、P1 工程化与可观测、P2 架构与性能、P3 部署上线。

**Architecture:** 现有 go-zero Gateway + Kafka Consumer 分层保留；P0 先补安全边界与可靠性闭环（鉴权、上传校验、超时降级、Kafka 一致性、文档生命周期、优雅退出），P1 补 CI、迁移、日志、指标、链路追踪，P2 做配置注入、批量写入与性能修正，P3 做容器化与压测。每阶段都产出可编译、可测试的独立交付物。

**Tech Stack:** Go 1.26.5、go-zero 1.10.2、GORM 1.31.2、Sarama、Eino、Milvus 2.5、ClickHouse 24.3、Docker Compose、gormigrate、otel。

## Global Constraints

- Go 版本以 `go.mod` 的 `go 1.26.5` 为准，不使用更高版本特性。
- 保持 goctl 目录结构：`cmd/gateway/internal/{handler,logic,svc,types,middleware,config}`、`cmd/consumer`、`internal/`、`pkg/`。
- GORM 连接已开启 `TranslateError: true`（`pkg/database/mysql.go`），1062 会翻译为 `gorm.ErrDuplicatedKey`，新代码继续依赖这一点。
- Consumer 配置约束：`LockTTL > HandleTimeout`，当前 `90s > 60s`，后续任务不得改坏。
- 错误响应统一走 `internal/errcode` 的 BizError；非 BizError 的原始错误只写日志，不直接返回客户端。
- 新增依赖需写进 go.mod/go.sum；测试用 `go test ./...`，CI 里跑 `-race`。
- 新文件默认 UTF-8，代码注释沿用现有中文风格。

---

## 学生推荐路径（先做这 9 个，其余以后再说）

如果一次性做完 30 个任务压力太大，只做下面的最小集合。这 9 个任务每个都是独立的小改动，按顺序做完，面试时 P0 的安全与可靠性就都有证据了。

| 顺序 | 任务 | 难度 | 为什么先做 |
| --- | --- | --- | --- |
| 1 | Task 4 密钥环境变量化 | 低 | 十几行代码，面试必问，完成感强 |
| 2 | Task 5 统一错误响应 | 低 | 改动集中，立刻消除“泄露内部错误”硬伤 |
| 3 | Task 8 锁 TTL + GenDocID | 低 | 两处小改，附带一个能跑的单测 |
| 4 | Task 10 AutoMigrate/InitSchema 错误处理 | 低 | 每个错误检查都是一两行，练手刚好 |
| 5 | Task 1 鉴权中间件 | 中 | 核心安全项，有现成测试代码 |
| 6 | Task 2 限流 key 改 IP+身份 | 中 | 逻辑小，但要求你读懂现有测试 |
| 7 | Task 6 Chat 超时降级 | 中 | 最能在面试讲出“生产思维”的一条 |
| 8 | Task 9 Consumer 优雅退出 | 中 | WaitGroup 是 Go 面试高频考点 |
| 9 | Task 3 上传白名单校验 | 中 | 收尾安全项，做完 P0 安全四条齐了 |

跳过的任务先别焦虑：

- Task 7（Kafka 补偿）需要想清楚“失败后怎么办”，先理解，不急着写。
- Task 11（文档删除生命周期）最难，建议后面和 Codex 结对做，你先读代码再一起写。
- Task 12-30 属于加分项，P0 没做完之前不用碰。

建议节奏：每天 1-2 个任务，做完一个提交一个；卡住超过 30 分钟就停下来问，不要硬啃。

---

## 阶段一：P0 安全与可靠性（先做，约 1-2 周）

### Task 1: 接口鉴权中间件（JWT/API Key）

**Files:**
- Create: `cmd/gateway/internal/middleware/auth.go`
- Create: `cmd/gateway/internal/middleware/auth_test.go`
- Modify: `cmd/gateway/internal/config/config.go`
- Modify: `cmd/gateway/internal/handler/routes.go`
- Modify: `internal/errcode/code.go`
- Modify: `cmd/gateway/main.go`
- Modify: `cmd/gateway/etc/doc.yaml`、`cmd/gateway/etc/doc.yaml.example`（Task 4 建示例，此处先改 doc.yaml）

**Interfaces:**
- Consumes: `config.Config.Auth.APIKey string`
- Produces: `middleware.WithAPIKey(apiKey string) func(http.HandlerFunc) http.HandlerFunc`；`errcode.Unauthorized = 1002`

- [ ] **Step 1: 写失败测试**

`cmd/gateway/internal/middleware/auth_test.go`：

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAPIKey(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
	handler := WithAPIKey("secret")(ok)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("missing key should not pass")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "wrong")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("wrong key should not pass")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "secret")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("correct key should pass, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./cmd/gateway/internal/middleware/...`
Expected: FAIL（`WithAPIKey` 未定义）

- [ ] **Step 3: 实现中间件**

`cmd/gateway/internal/middleware/auth.go`：

```go
package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/yourname/know/internal/errcode"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func WithAPIKey(apiKey string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-API-Key")
			if apiKey == "" || subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) != 1 {
				httpx.ErrorCtx(r.Context(), w, errcode.New(errcode.Unauthorized, "invalid api key"))
				return
			}
			next(w, r)
		}
	}
}
```

- [ ] **Step 4: 加错误码与状态映射**

`internal/errcode/code.go`：常量区加 `Unauthorized = 1002`，`codeMsg` 加 `Unauthorized: "未授权"`。

`cmd/gateway/main.go` 的 `httpx.SetErrorHandler` 内加：

```go
case errcode.Unauthorized:
	status = http.StatusUnauthorized
```

- [ ] **Step 5: 配置与路由挂载**

`cmd/gateway/internal/config/config.go` 的 `Config` 加：

```go
Auth struct {
	APIKey string `json:"apiKey"`
} `json:"auth"`
```

`cmd/gateway/etc/doc.yaml` 加：

```yaml
auth:
  apiKey: "change-me"
```

`cmd/gateway/internal/handler/routes.go`：在 `RegisterHandlers` 开头定义 `auth := middleware.WithAPIKey(serverCtx.Config.Auth.APIKey)`，然后两组路由的每个 `Handler:` 都包一层 `auth(...)`，例如 `Handler: auth(middleware.WithRateLimit(...))`。`/ping` 保持公开，不包鉴权。

- [ ] **Step 6: 跑测试确认通过**

Run: `go test ./cmd/gateway/internal/middleware/... ./internal/errcode/... && go build ./...`
Expected: PASS，编译通过

- [ ] **Step 7: 提交**

```bash
git add cmd/gateway/internal/middleware/auth.go cmd/gateway/internal/middleware/auth_test.go cmd/gateway/internal/config/config.go cmd/gateway/internal/handler/routes.go internal/errcode/code.go cmd/gateway/main.go cmd/gateway/etc/doc.yaml
git commit -m "feat: add api key auth for all /api/v1 routes"
```

### Task 2: 限流 key 改为 IP + 用户身份

**Files:**
- Modify: `cmd/gateway/internal/middleware/ratelimit.go`
- Modify: `cmd/gateway/internal/middleware/ratelimit_test.go`

**Interfaces:**
- Consumes: `X-API-Key` 请求头（Task 1 已鉴权）
- Produces: `clientKey(r *http.Request) string`

- [ ] **Step 1: 改实现**

`cmd/gateway/internal/middleware/ratelimit.go`：删除 `X-Request-Id` 优先逻辑，改用 `net.SplitHostPort` 解析 IP，并叠加 API Key 身份前缀。

```go
import (
	"net"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/yourname/know/internal/errcode"
)

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	host = strings.Trim(host, "[]")
	key := "ip:" + host
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		key += ":key:" + apiKey
	}
	return key
}
```

`WithRateLimit` 内把 `key := r.Header.Get("X-Request-Id")...` 那一段整段替换为 `key := clientKey(r)`。

- [ ] **Step 2: 更新测试**

`ratelimit_test.go`：

- `TestWithRateLimit_RejectExceedQuota`：把 `if rec.Code == http.StatusBadRequest` 改为 `if rec.Code != http.StatusOK`，并把日志文案改成 “rejected %d/10 requests (non-200)”。
- `TestWithRateLimit_RequestIDPriority` 整体删除，替换为 `TestWithRateLimit_RequestIDDoesNotBypass`：同一 IP、不同 X-Request-Id 仍应共享配额（发 3 次，配额 2，第 3 次应被拒绝）。
- `TestWithRateLimit_DifferentIPs` 保留，补充断言：IPv6 `[::1]:8080` 与 `[::2]:8080` 相互独立。

示例新增测试：

```go
func TestWithRateLimit_RequestIDDoesNotBypass(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 2, 1)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Request-Id", "req-000")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if i < 2 && rec.Code != http.StatusOK {
			t.Fatalf("request %d within quota should pass, got %d", i+1, rec.Code)
		}
		if i == 2 && rec.Code == http.StatusOK {
			t.Fatal("request 3 should be rejected: request id must not rotate the key")
		}
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `go test ./cmd/gateway/internal/middleware/...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/gateway/internal/middleware/ratelimit.go cmd/gateway/internal/middleware/ratelimit_test.go
git commit -m "fix: rate limit by ip + identity, request id cannot bypass"
```

### Task 3: 上传文件白名单 + 文件头校验 + 大小限制

**Files:**
- Modify: `cmd/gateway/internal/logic/uploaddoclogic.go`
- Modify: `cmd/gateway/internal/config/config.go`
- Modify: `cmd/gateway/etc/doc.yaml`

**Interfaces:**
- Consumes: `config.Config.Upload.MaxSizeBytes int64`
- Produces: 使用 `errcode.DocTypeNotSupported`；非 md/txt/pdf 在上传入口即拒绝

- [ ] **Step 1: 配置加上传限制**

`config.go` 的 `Config` 加：

```go
Upload struct {
	MaxSizeBytes int64 `json:"maxSizeBytes"`
} `json:"upload"`
```

`doc.yaml` 加：

```yaml
upload:
  maxSizeBytes: 10485760
```

- [ ] **Step 2: 实现校验**

`uploaddoclogic.go` 中 `filename = header.Filename` 之后、`ext := path.Ext(...)` 之前插入：

```go
ext = strings.ToLower(path.Ext(header.Filename))
switch ext {
case ".md", ".txt", ".pdf":
default:
	return nil, errcode.New(errcode.DocTypeNotSupported, "仅支持 md/txt/pdf 文件")
}
if header.Size > l.svcCtx.Config.Upload.MaxSizeBytes {
	return nil, errcode.New(errcode.DocUploadFailed, "文件超过大小限制")
}
if _, err := file.Seek(0, io.SeekStart); err != nil {
	return nil, errcode.New(errcode.DocUploadFailed, "读取文件失败: "+err.Error())
}
head := make([]byte, 512)
n, _ := io.ReadFull(file, head)
head = head[:n]
if ext == ".pdf" && !bytes.HasPrefix(head, []byte("%PDF-")) {
	return nil, errcode.New(errcode.DocTypeNotSupported, "pdf 文件头校验失败")
}
if (ext == ".md" || ext == ".txt") && !isTextBytes(head) {
	return nil, errcode.New(errcode.DocTypeNotSupported, "文本文件内容校验失败")
}
if _, err := file.Seek(0, io.SeekStart); err != nil {
	return nil, errcode.New(errcode.DocUploadFailed, "读取文件失败: "+err.Error())
}
```

文件末尾新增辅助函数：

```go
func isTextBytes(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}
```

导入补 `bytes`、`io`、`strings`；把原有的 `ext := path.Ext(header.Filename)` 改为 `ext := ""`，由上面的 `ext = strings.ToLower(path.Ext(...))` 赋值。

- [ ] **Step 3: 编译验证**

Run: `go build ./... && go vet ./cmd/gateway/...`
Expected: 无输出，退出码 0

- [ ] **Step 4: 提交**

```bash
git add cmd/gateway/internal/logic/uploaddoclogic.go cmd/gateway/internal/config/config.go cmd/gateway/etc/doc.yaml
git commit -m "feat: validate upload file type, magic bytes and size"
```

### Task 4: 密钥环境变量化 + 示例配置

**Files:**
- Modify: `cmd/gateway/main.go`
- Modify: `cmd/consumer/main.go`
- Create: `.env.example`
- Create: `cmd/gateway/etc/doc.yaml.example`
- Create: `cmd/consumer/etc/config.yaml.example`
- Modify: `README.md`

**Interfaces:**
- Consumes: 环境变量 `KNOX_QINIU_ACCESS_KEY`、`KNOX_QINIU_SECRET_KEY`、`KNOX_ARK_API_KEY`、`KNOX_MYSQL_DSN`、`KNOX_CLICKHOUSE_PASSWORD`

- [ ] **Step 1: 网关配置覆盖**

`cmd/gateway/main.go` 在 `conf.MustLoad(*configFile, &c)` 后加：

```go
if v := os.Getenv("KNOX_QINIU_ACCESS_KEY"); v != "" {
	c.Qiniu.AccessKey = v
}
if v := os.Getenv("KNOX_QINIU_SECRET_KEY"); v != "" {
	c.Qiniu.SecretKey = v
}
if v := os.Getenv("KNOX_ARK_API_KEY"); v != "" {
	c.ARK.APIKey = v
}
if v := os.Getenv("KNOX_MYSQL_DSN"); v != "" {
	c.Mysql.DSN = v
}
if v := os.Getenv("KNOX_CLICKHOUSE_PASSWORD"); v != "" {
	c.ClickHouse.Password = v
}
```

补 import `"os"`。

- [ ] **Step 2: Consumer 配置覆盖**

`cmd/consumer/main.go` 在 `conf.MustLoad(*configFile, &c)` 后加：

```go
if v := os.Getenv("KNOX_MYSQL_DSN"); v != "" {
	c.Mysql.DSN = v
}
```

补 import `"os"`。

- [ ] **Step 3: 写示例文件**

`.env.example`：

```bash
export KNOX_QINIU_ACCESS_KEY=""
export KNOX_QINIU_SECRET_KEY=""
export KNOX_ARK_API_KEY=""
export KNOX_MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/knox?charset=utf8mb4&parseTime=True&loc=Local"
export KNOX_CLICKHOUSE_PASSWORD=""
```

`cmd/gateway/etc/doc.yaml.example`：复制 doc.yaml 全部结构，密钥字段留空字符串，其余保留。

`cmd/consumer/etc/config.yaml.example`：复制 config.yaml，DSN 密码留空。

- [ ] **Step 4: 更新 README**

在 README “配置参数说明” 前新增“环境变量”小节，列出上表五个变量，并说明：生产环境必须通过环境变量注入，`doc.yaml` 已被 `.gitignore` 排除。

- [ ] **Step 5: 验证构建**

Run: `go build ./...`
Expected: 退出码 0

- [ ] **Step 6: 提交**

```bash
git add .env.example cmd/gateway/etc/doc.yaml.example cmd/consumer/etc/config.yaml.example cmd/gateway/main.go cmd/consumer/main.go README.md
git commit -m "feat: inject secrets via env vars and add example configs"
```

### Task 5: 统一错误响应（非 BizError 不透出原始错误）

**Files:**
- Modify: `cmd/gateway/main.go`
- Modify: `cmd/gateway/internal/handler/chathandler.go`

- [ ] **Step 1: 改错误处理器**

`main.go` 的 `httpx.SetErrorHandler` 非 BizError 分支改为：

```go
logx.Errorf("internal error: %v", err)
return http.StatusInternalServerError, map[string]interface{}{
	"code":    errcode.InternalError,
	"message": errcode.Msg(errcode.InternalError),
}
```

补 import `"github.com/zeromicro/go-zero/core/logx"`。

- [ ] **Step 2: SSE 错误事件不再透出原始文本**

`chathandler.go` 的 error 分支改为：

```go
if err != nil {
	code := errcode.InternalError
	msg := errcode.Msg(errcode.InternalError)
	if bizErr, ok := err.(*errcode.BizError); ok {
		code = bizErr.Code
		msg = bizErr.Message
	}
	data, _ := json.Marshal(map[string]interface{}{"code": code, "error": msg})
	fmt.Fprintf(w, "data: %s\n\n", data)
}
```

- [ ] **Step 3: 编译并跑测试**

Run: `go build ./... && go test ./...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/gateway/main.go cmd/gateway/internal/handler/chathandler.go
git commit -m "fix: never leak raw error text to clients"
```

### Task 6: Chat AgentTimeout + 降级回答不写会话

**Files:**
- Modify: `cmd/gateway/internal/logic/chatlogic.go`
- Modify: `cmd/gateway/internal/config/config.go`
- Modify: `cmd/gateway/etc/doc.yaml`
- Create: `cmd/gateway/internal/logic/chatlogic_test.go`

**Interfaces:**
- Consumes: `config.Config.Chat.AgentTimeout time.Duration`
- Produces: 超时/熔断时返回降级 `ChatResp`，且不 append 历史、不调 `SessionStore.Save`

- [ ] **Step 1: 配置**

`config.go` 的 `Config` 加：

```go
Chat struct {
	AgentTimeout time.Duration `json:"agentTimeout"`
} `json:"chat"`
```

`doc.yaml` 加：

```yaml
chat:
  agentTimeout: 30s
```

- [ ] **Step 2: 实现超时与降级**

`chatlogic.go` 的 `Chat` 中 `err = breaker.Do(...)` 前加：

```go
timeout := l.svcCtx.Config.Chat.AgentTimeout
if timeout <= 0 {
	timeout = 30 * time.Second
}
ctx, cancel := context.WithTimeout(l.ctx, timeout)
defer cancel()
```

把 `RunWithMessages(l.ctx, ...)` 改为 `RunWithMessages(ctx, ...)`。

`err` 判断分支改为：

```go
if err != nil {
	if ctx.Err() == context.DeadlineExceeded {
		logx.WithContext(l.ctx).Infof("chat degraded: agent timeout after %s", timeout)
		return &types.ChatResp{
			Answer:    "服务暂时繁忙，请稍后再试",
			SessionId: session.ID,
		}, nil
	}
	if bizErr, ok := err.(*errcode.BizError); ok && bizErr.Code == errcode.CircuitBreakerOpen {
		logx.WithContext(l.ctx).Info("chat degraded: circuit breaker open")
		return &types.ChatResp{
			Answer:    "服务暂时不可用，请稍后再试",
			SessionId: session.ID,
		}, nil
	}
	return nil, err
}
```

降级分支直接 return，不走到下面的 `session.Messages = append(...)` 和 `Save`。

补 import `"github.com/yourname/know/internal/errcode"`。

- [ ] **Step 3: 写测试**

`chatlogic_test.go`：构造带 1ns 超时的 ServiceContext 太重，这里用纯函数抽出来测。把降级判断抽成：

```go
func isDegradable(err error, ctx context.Context, timeout time.Duration) bool {
	if ctx.Err() == context.DeadlineExceeded {
		return true
	}
	bizErr, ok := err.(*errcode.BizError)
	return ok && bizErr.Code == errcode.CircuitBreakerOpen
}
```

测试：

```go
func TestIsDegradable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if !isDegradable(context.DeadlineExceeded, ctx, 30*time.Second) {
		t.Fatal("deadline exceeded should be degradable")
	}
	if isDegradable(errors.New("boom"), context.Background(), 30*time.Second) {
		t.Fatal("generic error should not be degradable")
	}
	if !isDegradable(errcode.New(errcode.CircuitBreakerOpen), context.Background(), 30*time.Second) {
		t.Fatal("breaker open should be degradable")
	}
}
```

- [ ] **Step 4: 跑测试**

Run: `go test ./cmd/gateway/internal/logic/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/gateway/internal/logic/chatlogic.go cmd/gateway/internal/logic/chatlogic_test.go cmd/gateway/internal/config/config.go cmd/gateway/etc/doc.yaml
git commit -m "feat: chat agent timeout with degraded answer, no session write"
```

### Task 7: Kafka 发送失败补偿，杜绝“入库但永不建索引”

**Files:**
- Modify: `cmd/gateway/internal/logic/uploaddoclogic.go`
- Modify: `internal/repository/document.go`

**Interfaces:**
- Produces: `DocumentRepository.DeleteByDocID(ctx, docID string) error`

- [ ] **Step 1: 仓库加删除方法**

`document.go` 加：

```go
func (r *DocumentRepository) DeleteByDocID(ctx context.Context, docID string) error {
	return r.db.WithContext(ctx).Where("doc_id = ?", docID).Delete(&model.Document{}).Error
}
```

- [ ] **Step 2: 发送失败补偿**

`uploaddoclogic.go` 中 `_, _, err = l.svcCtx.KafkaProducer.SendMessage(msg)` 分支改为：

```go
if _, _, sendErr := l.svcCtx.KafkaProducer.SendMessage(msg); sendErr != nil {
	logx.Errorf("send kafka message failed: %v", sendErr)
	if delErr := l.svcCtx.DocRepo.DeleteByDocID(l.ctx, document.DocID); delErr != nil {
		logx.Errorf("compensate delete doc %s failed: %v", document.DocID, delErr)
	}
	return nil, errcode.New(errcode.DocUploadFailed, "索引任务发送失败，请重试")
}
```

这样 Kafka 失败时 DB 行被补偿删除，客户端用同一 RequestId 重试会走干净的上传路径。

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 退出码 0

- [ ] **Step 4: 提交**

```bash
git add cmd/gateway/internal/logic/uploaddoclogic.go internal/repository/document.go
git commit -m "fix: compensate db row when kafka send fails"
```

### Task 8: 上传锁 TTL 与 GenDocID 防碰撞

**Files:**
- Modify: `cmd/gateway/internal/logic/uploaddoclogic.go`
- Modify: `internal/repository/document.go`
- Modify: `internal/repository/document_test.go`（新建）

- [ ] **Step 1: 锁 TTL 提到 5 分钟**

`uploaddoclogic.go`：`time.Second*30` 改为 `5 * time.Minute`。

- [ ] **Step 2: GenDocID 加随机后缀**

`document.go`：

```go
func GenDocID() string {
	return "doc_" + time.Now().Format("20060102150405") + "_" + uuid.NewString()[:8]
}
```

补 import `"github.com/google/uuid"`。

- [ ] **Step 3: 写测试**

`document_test.go`：

```go
package repository

import "testing"

func TestGenDocIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenDocID()
		if seen[id] {
			t.Fatalf("duplicate doc id: %s", id)
		}
		seen[id] = true
	}
}
```

- [ ] **Step 4: 跑测试**

Run: `go test ./internal/repository/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/gateway/internal/logic/uploaddoclogic.go internal/repository/document.go internal/repository/document_test.go
git commit -m "fix: longer upload lock ttl and collision-free doc id"
```

### Task 9: Consumer 优雅退出（WaitGroup 等待在途消息）

**Files:**
- Modify: `cmd/consumer/main.go`

- [ ] **Step 1: 改实现**

`main.go`：

- 删掉 `time.Sleep(1 * time.Second)`。
- 加 `var wg sync.WaitGroup`。
- 启动消费 goroutine 前 `wg.Add(1)`，goroutine 末尾 `defer wg.Done()`。
- 收到信号后 `cancel()`，然后 `wg.Wait()`，再 `consumer.Close()`（现有 defer 保留也可）。

补 import `"sync"`，删除不再使用的 `"time"` import。

- [ ] **Step 2: 编译**

Run: `go build ./cmd/consumer/...`
Expected: 退出码 0

- [ ] **Step 3: 提交**

```bash
git add cmd/consumer/main.go
git commit -m "fix: graceful consumer shutdown waits for in-flight messages"
```

### Task 10: AutoMigrate / InitSchema 错误显式处理

**Files:**
- Modify: `cmd/gateway/internal/svc/servicecontext.go`
- Modify: `cmd/consumer/main.go`

- [ ] **Step 1: Gateway AutoMigrate**

`servicecontext.go`：

```go
if err := db.AutoMigrate(&model.Document{}); err != nil {
	panic("failed to auto migrate: " + err.Error())
}
```

- [ ] **Step 2: ClickHouse InitSchema 失败降级为 nil**

`servicecontext.go` 的 schema 失败分支改为：

```go
if err := analyticsClient.InitSchema(); err != nil {
	logx.Errorf("clickhouse init schema failed: %v", err)
	analyticsClient = analytics.New(nil) // 建表失败则整体降级，避免每条写入都失败
}
```

- [ ] **Step 3: Consumer AutoMigrate**

`cmd/consumer/main.go`：

```go
if err := db.AutoMigrate(&model.Document{}, &model.ConsumeRecord{}); err != nil {
	panic("failed to auto migrate: " + err.Error())
}
```

- [ ] **Step 4: 编译并提交**

Run: `go build ./...`
Expected: 退出码 0

```bash
git add cmd/gateway/internal/svc/servicecontext.go cmd/consumer/main.go
git commit -m "fix: handle automigrate and analytics init errors explicitly"
```

### Task 11: 文档更新/删除流程（重新索引 + Milvus 清理）

**Files:**
- Create: `internal/model/doc_chunk.go`
- Modify: `internal/repository/document.go`
- Create: `internal/repository/doc_chunk.go`
- Modify: `cmd/consumer/handler.go`
- Modify: `cmd/gateway/internal/logic/deletedoclogic.go`（新建）
- Modify: `cmd/gateway/internal/handler/deletedochandler.go`（新建）
- Modify: `cmd/gateway/internal/handler/routes.go`
- Modify: `api/doc.api`

**Interfaces:**
- Produces: `model.DocChunk{DocID, ChunkID}`；`DocChunkRepository.SaveChunks(ctx, docID, chunkIDs)` / `ListChunkIDs(ctx, docID)` / `DeleteByDocID(ctx, docID)`；`DocumentRepository.GetByDocID`（已有）

- [ ] **Step 1: 新增 chunk 映射表**

`internal/model/doc_chunk.go`：

```go
package model

import "gorm.io/gorm"

type DocChunk struct {
	gorm.Model
	DocID   string `gorm:"index;size:64;not null"`
	ChunkID string `gorm:"uniqueIndex;size:128;not null"`
}
```

`internal/repository/doc_chunk.go`：

```go
package repository

import (
	"context"

	"github.com/yourname/know/internal/model"
	"gorm.io/gorm"
)

type DocChunkRepository struct {
	db *gorm.DB
}

func NewDocChunkRepository(db *gorm.DB) *DocChunkRepository {
	return &DocChunkRepository{db: db}
}

func (r *DocChunkRepository) SaveChunks(ctx context.Context, docID string, chunkIDs []string) error {
	if len(chunkIDs) == 0 {
		return nil
	}
	rows := make([]model.DocChunk, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		rows = append(rows, model.DocChunk{DocID: docID, ChunkID: id})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *DocChunkRepository) ListChunkIDs(ctx context.Context, docID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&model.DocChunk{}).
		Where("doc_id = ?", docID).
		Pluck("chunk_id", &ids).Error
	return ids, err
}

func (r *DocChunkRepository) DeleteByDocID(ctx context.Context, docID string) error {
	return r.db.WithContext(ctx).Where("doc_id = ?", docID).Delete(&model.DocChunk{}).Error
}
```

把 `DocChunk` 加进 gateway 和 consumer 的 AutoMigrate。

- [ ] **Step 2: Consumer 建索引成功后记录 chunk**

`cmd/consumer/handler.go` 的 `messageHandler` 加字段 `chunkRepo *repository.DocChunkRepository`，`newMessageHandler` 里初始化；`docIndexer.Store` 成功后：

```go
chunkIDs := make([]string, 0, len(chunks))
for _, c := range chunks {
	chunkIDs = append(chunkIDs, c.ID)
}
if err := h.chunkRepo.SaveChunks(ctx, task.DocID, chunkIDs); err != nil {
	log.Printf("[%s] save chunks failed: %v", task.DocID, err)
	h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, "save chunks failed: "+err.Error())
	return false
}
```

- [ ] **Step 3: Consumer 处理 delete 消息**

`embedTask` 加 `Action string \`json:"action"\``。`handleMessage` 开头：

```go
if task.Action == "delete" {
	chunkIDs, err := h.chunkRepo.ListChunkIDs(ctx, task.DocID)
	if err != nil {
		log.Printf("[%s] list chunks failed: %v", task.DocID, err)
		return false
	}
	if len(chunkIDs) > 0 {
		if err := docIndexer.Delete(ctx, chunkIDs); err != nil {
			log.Printf("[%s] delete vectors failed: %v", task.DocID, err)
			return false
		}
	}
	if err := h.chunkRepo.DeleteByDocID(ctx, task.DocID); err != nil {
		log.Printf("[%s] delete chunk rows failed: %v", task.DocID, err)
		return false
	}
	log.Printf("[%s] deleted vectors (%d chunks)", task.DocID, len(chunkIDs))
	return true
}
```

> 说明：`docIndexer.Delete` 以 eino milvus2 Indexer 的实际 API 为准，若签名是 `Delete(ctx, ids ...string)` 则传 `chunkIDs...`；实现时先 `go doc` 确认。

- [ ] **Step 4: Gateway 删除接口**

`deletedoclogic.go`：

```go
package logic

import (
	"context"

	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/internal/errcode"
	"github.com/yourname/know/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocLogic {
	return &DeleteDocLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteDocLogic) DeleteDoc(docID string) error {
	doc, err := l.svcCtx.DocRepo.GetByDocID(l.ctx, docID)
	if err != nil {
		return errcode.New(errcode.DocNotFound, "文档不存在")
	}
	if err := l.svcCtx.DocRepo.DeleteByDocID(l.ctx, doc.DocID); err != nil {
		return errcode.New(errcode.DocUploadFailed, "删除文档失败")
	}
	msg := map[string]string{"action": "delete", "docId": doc.DocID}
	data, _ := json.Marshal(msg)
	if _, _, err := l.svcCtx.KafkaProducer.SendMessage(&sarama.ProducerMessage{
		Topic: l.svcCtx.Config.Kafka.Topic,
		Value: sarama.ByteEncoder(data),
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("send delete kafka message failed: %v", err)
	}
	return nil
}
```

补 import `encoding/json`、`github.com/IBM/sarama`。

`deletedochandler.go`：`DELETE /doc/:docId`，用 `httpx.ParsePathVars` 取 docId，调用 `DeleteDocLogic`。

`routes.go` 加路由：

```go
{
	Method:  http.MethodDelete,
	Path:    "/doc/:docId",
	Handler: auth(DeleteDocHandler(serverCtx)),
},
```

`api/doc.api` 加：

```go
@handler DeleteDoc
delete /doc/:docId
```

- [ ] **Step 5: 编译验证**

Run: `go build ./...`
Expected: 退出码 0

- [ ] **Step 6: 提交**

```bash
git add internal/model/doc_chunk.go internal/repository/doc_chunk.go cmd/consumer/handler.go cmd/gateway/internal/logic/deletedoclogic.go cmd/gateway/internal/handler/deletedochandler.go cmd/gateway/internal/handler/routes.go api/doc.api
git commit -m "feat: document delete flow removes milvus vectors and db rows"
```

---

## 阶段二：P1 工程化与可观测

### Task 12: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

```yaml
name: ci
on:
  push:
  pull_request:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - run: go build ./...
      - run: go vet ./...
      - run: go test ./...
      - run: go test -race ./...
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

验证：推分支后 Actions 全绿；本机跑 `go vet ./... && go test ./...` 通过再提交。

提交：`git add .github/workflows/ci.yml && git commit -m "ci: add build/test/vet/race/lint pipeline"`

### Task 13: 引入迁移工具（gormigrate）

**Files:**
- Create: `internal/migration/migrate.go`
- Modify: `cmd/gateway/internal/svc/servicecontext.go`
- Modify: `cmd/consumer/main.go`

实现：`gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{...})`，首版迁移建 `documents`、`consume_records`、`doc_chunks` 三张表；两处 `AutoMigrate` 全部替换为 `migration.Migrate(db)`。迁移脚本用 `gorm.AutoMigrate` 建表，后续 schema 变更以新增 migration 为准。

验证：`go build ./... && go test ./...`；本地起 MySQL 后跑一遍 gateway，确认表结构存在。

### Task 14: 死代码清理

**Files:**
- Delete: `internal/vector/cozeloop.go`
- Modify: `go.mod` / `go.sum`（`go mod tidy` 去掉 cozeloop 依赖）
- Modify: `internal/repository/document.go`（删 `UpdateVersion`）
- Delete: `cmd/analytics`、`cmd/search`、`internal/config`、`pkg/httpx`、`pkg/response` 空目录

统一 `GetByRequestID` / `GetByDocID` 风格：`GetByRequestID` 找不到时返回 `gorm.ErrRecordNotFound`，所有调用处改为 `errors.Is(err, gorm.ErrRecordNotFound)` 判断。

验证：`go build ./... && go test ./...`；`go mod tidy` 后确认 go.mod 不再含 cozeloop。

### Task 15: 补关键单测

**Files:**
- Create: `cmd/gateway/internal/logic/uploaddoclogic_test.go`（用 sqlmock/glebarez-sqlite 断言同 RequestId 第二次不上传、返回相同 DocId）
- Modify: `cmd/gateway/main_test.go`（错误码映射：BizError 1003→429、1004→503、非 BizError→500+1005）
- Create: `internal/session/store_test.go`（会话裁剪：最近 10 轮）
- Create: `cmd/gateway/internal/logic/searchlogic_test.go`（TopK 透传、score 从 metadata 取，依赖 Task 20）

上传幂等测试用现有 `internal/repository/consume_record_test.go` 的 glebarez-sqlite 模式，仓库方法已足够；logic 层测试用假 `DocRepo` 接口抽出来写，避免依赖七牛。

验证：`go test ./...` 全绿。

### Task 16: 统一 JSON 结构化日志

**Files:**
- Modify: `cmd/consumer/main.go`、`cmd/consumer/handler.go`（`log.Printf` → `logx.Infof/Errorf`）
- Modify: `cmd/gateway/main.go`、`cmd/consumer/main.go`（启动时 `logx.MustSetup(logx.LogConf{Mode: "file", Path: "logs", KeepDays: 7})`）

Consumer 里所有 `log.Printf` 替换为 `logx`，去掉 `"log"` import。验证：`go build ./...`，运行 consumer 后 `logs/` 下产生 JSON 日志。

### Task 17: Prometheus /metrics + Jaeger 链路

**Files:**
- Modify: `cmd/gateway/etc/doc.yaml`（`Prometheus: {Host: 0.0.0.0, Port: 9091, Path: /metrics}`，go-zero RestConf 自带）
- Modify: `docker-compose/infra.yml`（加 `prometheus` 服务，端口 9090，配置抓 9091）
- Create: `pkg/trace/trace.go`（`otlptracegrpc` 连 `127.0.0.1:4317`，`otel.SetTracerProvider`）
- Modify: `cmd/gateway/main.go`、`cmd/consumer/main.go`（启动时调 `trace.Init(ctx, "gateway"/"consumer")`）
- Modify: Kafka 生产者（`uploaddoclogic.go` / `deletedoclogic.go`）在消息头写 `traceparent`；Consumer `handleMessage` 提取 `traceparent` 续接 span，并把 `trace.SpanFromContext(ctx).SpanContext().TraceID()` 写入 analytics `trace_id` 和日志。

验证：起 jaeger + gateway，curl `/api/v1/ping` 后访问 `http://127.0.0.1:16686` 能看到 gateway span；Consumer 处理一条消息后 trace_id 非空。

### Task 18: /ping 依赖健康检查

**Files:**
- Modify: `cmd/gateway/internal/logic/pinglogic.go`
- Modify: `cmd/gateway/internal/types/types.go`

实现：`Ping` 返回 `map[string]string`，逐项检查 MySQL（`db.Exec("SELECT 1")`）、Redis（Ping）、Kafka（`producer` 连接可用性）、Milvus（`CheckHealth`）、ClickHouse（Ping）；全绿返回 `"ok"`，否则返回 503 和失败项。handler 按结果写 `httpx.OkJson` / `httpx.ErrorCtx`。

---

## 阶段三：P2 架构与性能

### Task 19: vector 配置注入

`vector.EmbeddingClient(ctx, url, model)`、`vector.MilvusClient(ctx, addr)`、`vector.RetrieverClient(ctx, emb, milvus, collection, topK)` 全部收配置参数；`servicecontext.go` 与 `searchlogic.go` 从 `Config` 传入。删除 `localhost:11434`、`127.0.0.1:19530`、`TopK: 5`、`knox_docs` 硬编码。Gateway/Consumer 配置里补 `Ollama.URL/Model`、`Milvus.Addr/Collection/TopK`。

### Task 20: 搜索真正使用 TopK + metadata docId/score

`searchlogic.go`：`req.TopK` 非法时默认 5，透传 retriever；`Retrieve` 返回的 `doc.MetaData` 中取 `doc_id` 与 `score`（consumer 建索引时已写入 `doc.MetaData["doc_id"]`，检索结果里 score 从返回结构取），映射到 `SearchResult.DocId/Score`。为 retriever 增加按请求参数重建实例的方法 `RetrieverClientWithTopK(...)`，避免并发改全局 TopK。

### Task 21: Analytics channel + 批量写入

`Analytics` 增加 `ch chan analyticsEvent`（容量 1000）、`done chan struct{}`、后台 flush goroutine：满 100 条或每 5s 批量 INSERT，`Close()` 收尾 flush。`logEvent` 改为非阻塞入队。Gateway 退出时调用 `Close`。

### Task 22: 会话历史裁剪最近 N 轮

`session.Store.Save` / `chatlogic.go` 在 append 后裁剪：保留最近 10 轮（20 条 Message），超出的丢弃。补 `store_test.go` 断言。

### Task 23: consume_record 清理策略

Consumer 加定时 goroutine：`done` 超过 30 天删除；`failed` 超过 7 天删除并记录归档日志。`consume_records` 表加 `status` + `updated_at` 索引（迁移任务里补）。

### Task 24: ClickHouse TTL 与物化聚合

`InitSchema` 改为 `ENGINE = MergeTree() ... TTL event_time + INTERVAL 90 DAY`，并新建 `operation_logs_daily` 物化聚合表（按天、event_type 聚合 total/avg/max/error_rate）；analytics 查询优先读聚合表。

### Task 25: 抽出 internal/consumer.Processor

把 `cmd/consumer/handler.go` 的 `messageHandler` 迁到 `internal/consumer/processor.go`，依赖接口化（`DB`、`Lock`、`RecordRepo`、`ChunkRepo`、`Indexer`），`cmd/consumer` 只负责组装。处理器可脱离 Kafka 单测：喂 `sarama.ConsumerMessage`，断言状态流转。

---

## 阶段四：P3 部署上线

### Task 26: Dockerfile + docker-compose.prod.yml

为 gateway/consumer 写 `Dockerfile`（多阶段：`golang:1.26-alpine` build → `alpine` 运行），`docker-compose.prod.yml` 含两个服务 + infra 依赖，配置 `restart: unless-stopped`、`mem_limit`、`cpus`、`healthcheck`、日志 volume。

### Task 27: Nginx/Caddy TLS + SSE 关缓冲

`nginx.conf`：`/api/v1/chat` 的 `proxy_buffering off;`、`X-Accel-Buffering: no`，其余接口正常；配置证书路径。或提供 Caddyfile 一行式 TLS。

### Task 28: 数据卷独立盘 + 备份脚本

`scripts/backup.sh`：`mysqldump` + ClickHouse `BACKUP TABLE`（或 tab-separated 导出）+ 七牛/Milvus 数据卷快照说明；README 记录“挂载独立盘”要求。

### Task 29: 压测基线

`scripts/loadtest/` 放 k6 脚本：`upload.js`、`search.js`、`chat.js`；跑完后把 QPS/P95/错误率写入 README“性能基线”章节。压测前先 `docker compose up -d` 并限流配额调大。

### Task 30: README 架构图、鉴权、环境变量、部署

README 补：Mermaid 架构图、`X-API-Key` 鉴权说明、环境变量表（Task 4）、生产部署步骤（Task 26-28）、压测基线（Task 29）。

---

## Self-Review

1. **Spec 覆盖**：P0 安全（Task 1-5）、P0 可靠性（Task 6-11）、P1 工程化（Task 12-15）、P1 可观测（Task 16-18）、P2（Task 19-25）、P3（Task 26-30）全部有对应任务。
2. **占位符扫描**：全计划无 TBD/TODO；`docIndexer.Delete` 的签名以 `go doc` 确认为前置步骤，已明确写出。
3. **类型一致性**：`middleware.WithAPIKey`、`errcode.Unauthorized`、`isDegradable`、`DeleteByDocID`、`DocChunkRepository.SaveChunks/ListChunkIDs/DeleteByDocID` 在后续任务中均按本计划签名引用。
