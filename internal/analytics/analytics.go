package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Analytics ClickHouse 统计组件：负责建表、异步埋点写入、统计查询
type Analytics struct {
	db *sql.DB
}

// New 创建 Analytics 实例，db 为 nil 时所有写操作静默丢弃
func New(db *sql.DB) *Analytics {
	return &Analytics{db: db}
}

// InitSchema 创建 ClickHouse 操作日志表，幂等
func (a *Analytics) InitSchema() error {
	if a.db == nil {
		return nil
	}
	ddl := `
	CREATE TABLE IF NOT EXISTS operation_logs (
		event_time  DateTime,
		event_type  String,
		duration_ms Int64,
		success     UInt8,
		trace_id    String,
		detail      String,
		created_at  DateTime DEFAULT now()
	) ENGINE = MergeTree()
	PARTITION BY toYYYYMMDD(event_time)
	ORDER BY (event_type, event_time)
	`
	_, err := a.db.Exec(ddl)
	return err
}

// logEvent 异步写入一条操作日志（fire-and-forget，失败只记日志不返回 error）
func (a *Analytics) logEvent(eventType string, durationMs int64, success bool, traceID string, detail map[string]interface{}) {
	if a.db == nil {
		return
	}
	go func() {
		detailJSON, _ := json.Marshal(detail)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := a.db.ExecContext(ctx,
			"INSERT INTO operation_logs (event_time, event_type, duration_ms, success, trace_id, detail) VALUES (?, ?, ?, ?, ?, ?)",
			time.Now(), eventType, durationMs, success, traceID, string(detailJSON))
		if err != nil {
			logx.Errorf("analytics: insert %s log failed: %v", eventType, err)
		}
	}()
}

// LogChat 记录一次聊天请求
func (a *Analytics) LogChat(durationMs int64, success bool, traceID string, questionLen, answerLen, toolCalls int) {
	a.logEvent("chat", durationMs, success, traceID, map[string]interface{}{
		"question_len": questionLen,
		"answer_len":   answerLen,
		"tool_calls":   toolCalls,
	})
}

// LogSearch 记录一次搜索请求
func (a *Analytics) LogSearch(durationMs int64, success bool, traceID string, query string, resultCount int) {
	a.logEvent("search", durationMs, success, traceID, map[string]interface{}{
		"query":        query,
		"result_count": resultCount,
	})
}

// LogUpload 记录一次文档上传
func (a *Analytics) LogUpload(durationMs int64, success bool, traceID string, filename string, fileSize int64) {
	a.logEvent("upload", durationMs, success, traceID, map[string]interface{}{
		"filename":  filename,
		"file_size": fileSize,
	})
}
