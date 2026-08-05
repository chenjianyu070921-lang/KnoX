package types

// DashboardResp 大盘聚合响应（直接透传 analytics.DashboardData 的 JSON）
type DashboardResp struct {
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

type HourlyPoint struct {
	Hour      string  `json:"hour"`
	ChatCnt   int64   `json:"chatCnt"`
	SearchCnt int64   `json:"searchCnt"`
	UploadCnt int64   `json:"uploadCnt"`
	AvgMs     float64 `json:"avgMs"`
}

type DistItem struct {
	EventType string `json:"eventType"`
	Cnt       int64  `json:"cnt"`
}

type SuccessRatePoint struct {
	Date       string  `json:"date"`
	Total      int64   `json:"total"`
	SuccessCnt int64   `json:"successCnt"`
	Rate       float64 `json:"rate"`
}
