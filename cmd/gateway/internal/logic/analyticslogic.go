package logic

import (
	"context"
	"time"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyticsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAnalyticsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyticsLogic {
	return &AnalyticsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Overview 大盘概览（最近 24h）
func (l *AnalyticsLogic) Overview() (*types.AnalyticsOverviewResp, error) {
	if l.svcCtx.Analytics == nil {
		return &types.AnalyticsOverviewResp{}, nil
	}
	stats, err := l.svcCtx.Analytics.Overview(time.Now().Add(-24 * time.Hour))
	if err != nil {
		logx.Errorf("analytics overview query failed: %v", err)
		return &types.AnalyticsOverviewResp{}, nil // 降级返回空
	}
	resp := &types.AnalyticsOverviewResp{}
	for _, s := range stats {
		stat := &types.AnalyticsEventStat{
			Total:     s.Total,
			AvgMs:     s.AvgMs,
			MaxMs:     s.MaxMs,
			ErrorRate: s.ErrorRate,
		}
		switch s.EventType {
		case "chat":
			resp.Chat = stat
		case "search":
			resp.Search = stat
		case "upload":
			resp.Upload = stat
		}
	}
	return resp, nil
}

// Trends 按天趋势
func (l *AnalyticsLogic) Trends(req *types.AnalyticsTrendsReq) (*types.AnalyticsTrendsResp, error) {
	if l.svcCtx.Analytics == nil {
		return &types.AnalyticsTrendsResp{Points: []types.AnalyticsTrendPoint{}}, nil
	}
	days := normalizeAnalyticsDays(req.Days)
	points, err := l.svcCtx.Analytics.Trends(days)
	if err != nil {
		logx.Errorf("analytics trends query failed: %v", err)
		return &types.AnalyticsTrendsResp{Points: []types.AnalyticsTrendPoint{}}, nil
	}
	resp := &types.AnalyticsTrendsResp{Points: make([]types.AnalyticsTrendPoint, 0, len(points))}
	for _, p := range points {
		resp.Points = append(resp.Points, types.AnalyticsTrendPoint{
			Date:  p.Date,
			Total: p.Total,
			AvgMs: p.AvgMs,
			P95Ms: p.P95Ms,
		})
	}
	return resp, nil
}

// SlowQueries 慢查询 TOP N
func (l *AnalyticsLogic) SlowQueries(req *types.AnalyticsSlowReq) (*types.AnalyticsSlowResp, error) {
	if l.svcCtx.Analytics == nil {
		return &types.AnalyticsSlowResp{Items: []types.AnalyticsSlowItem{}}, nil
	}
	limit := normalizeSlowQueryLimit(req.Limit)
	items, err := l.svcCtx.Analytics.SlowQueries(limit)
	if err != nil {
		logx.Errorf("analytics slow queries failed: %v", err)
		return &types.AnalyticsSlowResp{Items: []types.AnalyticsSlowItem{}}, nil
	}
	resp := &types.AnalyticsSlowResp{Items: make([]types.AnalyticsSlowItem, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, types.AnalyticsSlowItem{
			EventTime:  it.EventTime,
			EventType:  it.EventType,
			DurationMs: it.DurationMs,
			Success:    it.Success,
			TraceID:    it.TraceID,
			Detail:     it.Detail,
		})
	}
	return resp, nil
}

const (
	maxAnalyticsDays  = 90
	maxSlowQueryLimit = 100
)

func normalizeAnalyticsDays(days int) int {
	if days <= 0 {
		days = 7
	}
	if days > maxAnalyticsDays {
		days = maxAnalyticsDays
	}
	return days
}

func normalizeSlowQueryLimit(limit int) int {
	if limit <= 0 {
		limit = 20
	}
	if limit > maxSlowQueryLimit {
		limit = maxSlowQueryLimit
	}
	return limit
}
