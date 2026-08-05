package analytics

import (
	"context"
	"time"
)

// EventStats 单个事件类型的统计指标
type EventStats struct {
	EventType string  `json:"eventType"`
	Total     int64   `json:"total"`
	AvgMs     float64 `json:"avgMs"`
	MaxMs     int64   `json:"maxMs"`
	ErrorRate float64 `json:"errorRate"`
}

// TrendPoint 单日趋势数据点
type TrendPoint struct {
	Date  string  `json:"date"`
	Total int64   `json:"total"`
	AvgMs float64 `json:"avgMs"`
	P95Ms float64 `json:"p95Ms"`
}

// SlowItem 慢查询项
type SlowItem struct {
	EventTime  string `json:"eventTime"`
	EventType  string `json:"eventType"`
	DurationMs int64  `json:"durationMs"`
	Success    bool   `json:"success"`
	TraceID    string `json:"traceId"`
	Detail     string `json:"detail"`
}

// Overview 返回指定时间段内各 event_type 的统计概览
func (a *Analytics) Overview(since time.Time) ([]EventStats, error) {
	if a.db == nil {
		return nil, nil
	}
	query := `
		SELECT
			event_type,
			count()                    AS total,
			round(avg(duration_ms), 2) AS avg_ms,
			max(duration_ms)           AS max_ms,
			round(countIf(success = 0) * 100.0 / count(), 2) AS error_rate
		FROM operation_logs
		WHERE event_time >= ?
		GROUP BY event_type
		ORDER BY event_type
	`
	rows, err := a.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EventStats
	for rows.Next() {
		var s EventStats
		if err := rows.Scan(&s.EventType, &s.Total, &s.AvgMs, &s.MaxMs, &s.ErrorRate); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// Trends 返回最近 N 天按天的请求数与耗时趋势
func (a *Analytics) Trends(days int) ([]TrendPoint, error) {
	if a.db == nil {
		return nil, nil
	}
	if days <= 0 {
		days = 7
	}
	query := `
		SELECT
			toString(toDate(event_time))      AS date,
			count()                            AS total,
			round(avg(duration_ms), 2)         AS avg_ms,
			round(quantile(0.95)(duration_ms), 2) AS p95_ms
		FROM operation_logs
		WHERE event_time >= now() - INTERVAL ? DAY
		GROUP BY date
		ORDER BY date
	`
	rows, err := a.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TrendPoint
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Date, &p.Total, &p.AvgMs, &p.P95Ms); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// SlowQueries 返回耗时最长的 TOP N 操作记录
func (a *Analytics) SlowQueries(limit int) ([]SlowItem, error) {
	if a.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT
			toString(event_time)  AS event_time,
			event_type,
			duration_ms,
			success,
			trace_id,
			detail
		FROM operation_logs
		ORDER BY duration_ms DESC
		LIMIT ?
	`
	rows, err := a.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SlowItem
	for rows.Next() {
		var item SlowItem
		var successU8 uint8
		if err := rows.Scan(&item.EventTime, &item.EventType, &item.DurationMs, &successU8, &item.TraceID, &item.Detail); err != nil {
			return nil, err
		}
		item.Success = successU8 != 0
		result = append(result, item)
	}
	return result, rows.Err()
}

// ==================== Dashboard 大盘聚合（Reporter 服务） ====================

// HourlyPoint 24h 按小时趋势点
type HourlyPoint struct {
	Hour      string  `json:"hour"`
	ChatCnt   int64   `json:"chatCnt"`
	SearchCnt int64   `json:"searchCnt"`
	UploadCnt int64   `json:"uploadCnt"`
	AvgMs     float64 `json:"avgMs"`
}

// DistItem 事件类型分布
type DistItem struct {
	EventType string `json:"eventType"`
	Cnt       int64  `json:"cnt"`
}

// SuccessRatePoint 7天成功率数据点
type SuccessRatePoint struct {
	Date       string  `json:"date"`
	Total      int64   `json:"total"`
	SuccessCnt int64   `json:"successCnt"`
	Rate       float64 `json:"rate"`
}

// DashboardData 大盘聚合数据
type DashboardData struct {
	ChatTotal     int64              `json:"chatTotal"`
	SearchTotal   int64              `json:"searchTotal"`
	UploadTotal   int64              `json:"uploadTotal"`
	ChatAvgMs     float64            `json:"chatAvgMs"`
	SearchAvgMs   float64            `json:"searchAvgMs"`
	UploadAvgMs   float64            `json:"uploadAvgMs"`
	ErrorRate     float64            `json:"errorRate"`
	P50Ms         float64            `json:"p50Ms"`
	P95Ms         float64            `json:"p95Ms"`
	P99Ms         float64            `json:"p99Ms"`
	Hourly        []HourlyPoint      `json:"hourly"`
	Distribution  []DistItem         `json:"distribution"`
	SuccessRate7d []SuccessRatePoint `json:"successRate7d"`
}

// Dashboard 返回前端大盘所需的所有图表数据
func (a *Analytics) Dashboard() (*DashboardData, error) {
	if a.db == nil {
		return &DashboardData{}, nil
	}
	ctx := context.Background()
	d := &DashboardData{}

	// 1. 24h 各类别总量 + 平均延迟
	rows, err := a.db.QueryContext(ctx,
		`SELECT event_type, count() AS total, round(avg(duration_ms),2) AS avg_ms
		 FROM operation_logs WHERE event_time >= now() - INTERVAL 24 HOUR
		 GROUP BY event_type`)
	if err == nil {
		for rows.Next() {
			var et string
			var total int64
			var avgMs float64
			rows.Scan(&et, &total, &avgMs)
			switch et {
			case "chat":
				d.ChatTotal, d.ChatAvgMs = total, avgMs
			case "search":
				d.SearchTotal, d.SearchAvgMs = total, avgMs
			case "upload":
				d.UploadTotal, d.UploadAvgMs = total, avgMs
			}
		}
		rows.Close()
	}

	// 2. 错误率 + 延迟分位数
	_ = a.db.QueryRowContext(ctx,
		`SELECT round(countIf(success=0)*100.0/nullIf(count(),0),2),
		        round(quantile(0.50)(duration_ms),2),
		        round(quantile(0.95)(duration_ms),2),
		        round(quantile(0.99)(duration_ms),2)
		 FROM operation_logs WHERE event_time >= now() - INTERVAL 24 HOUR`,
	).Scan(&d.ErrorRate, &d.P50Ms, &d.P95Ms, &d.P99Ms)

	// 3. 24h 按小时趋势
	hr, err := a.db.QueryContext(ctx,
		`SELECT toString(toStartOfHour(event_time)) AS h,
		        countIf(event_type='chat') AS chat_cnt,
		        countIf(event_type='search') AS search_cnt,
		        countIf(event_type='upload') AS upload_cnt,
		        round(avg(duration_ms),1) AS avg_ms
		 FROM operation_logs WHERE event_time >= now() - INTERVAL 24 HOUR
		 GROUP BY h ORDER BY h`)
	if err == nil {
		for hr.Next() {
			var hp HourlyPoint
			hr.Scan(&hp.Hour, &hp.ChatCnt, &hp.SearchCnt, &hp.UploadCnt, &hp.AvgMs)
			d.Hourly = append(d.Hourly, hp)
		}
		hr.Close()
	}

	// 4. 事件类型分布
	dr, err := a.db.QueryContext(ctx,
		`SELECT event_type, count() AS cnt
		 FROM operation_logs WHERE event_time >= now() - INTERVAL 24 HOUR
		 GROUP BY event_type ORDER BY cnt DESC`)
	if err == nil {
		for dr.Next() {
			var di DistItem
			dr.Scan(&di.EventType, &di.Cnt)
			d.Distribution = append(d.Distribution, di)
		}
		dr.Close()
	}

	// 5. 7天成功率趋势
	sr, err := a.db.QueryContext(ctx,
		`SELECT toString(toDate(event_time)) AS d, count() AS total,
		        countIf(success=1) AS ok_cnt,
		        round(countIf(success=1)*100.0/nullIf(count(),0),2) AS rate
		 FROM operation_logs WHERE event_time >= now() - INTERVAL 7 DAY
		 GROUP BY d ORDER BY d`)
	if err == nil {
		for sr.Next() {
			var sp SuccessRatePoint
			sr.Scan(&sp.Date, &sp.Total, &sp.SuccessCnt, &sp.Rate)
			d.SuccessRate7d = append(d.SuccessRate7d, sp)
		}
		sr.Close()
	}

	return d, nil
}
