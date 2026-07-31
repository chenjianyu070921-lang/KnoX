package analytics

import (
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
