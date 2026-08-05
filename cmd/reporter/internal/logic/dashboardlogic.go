package logic

import (
	"context"

	"github.com/yourname/know/cmd/reporter/internal/svc"
	"github.com/yourname/know/cmd/reporter/internal/types"
	"github.com/yourname/know/internal/analytics"

	"github.com/zeromicro/go-zero/core/logx"
)

type DashboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DashboardLogic {
	return &DashboardLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DashboardLogic) Dashboard() (*types.DashboardResp, error) {
	if l.svcCtx.Analytics == nil {
		return &types.DashboardResp{}, nil
	}

	d, err := l.svcCtx.Analytics.Dashboard()
	if err != nil {
		return nil, err
	}

	return convertDashboard(d), nil
}

func convertDashboard(d *analytics.DashboardData) *types.DashboardResp {
	resp := &types.DashboardResp{
		ChatTotal:   d.ChatTotal,
		SearchTotal: d.SearchTotal,
		UploadTotal: d.UploadTotal,
		ChatAvgMs:   d.ChatAvgMs,
		SearchAvgMs: d.SearchAvgMs,
		UploadAvgMs: d.UploadAvgMs,
		ErrorRate:   d.ErrorRate,
		P50Ms:       d.P50Ms,
		P95Ms:       d.P95Ms,
		P99Ms:       d.P99Ms,
	}

	for _, hp := range d.Hourly {
		resp.Hourly = append(resp.Hourly, types.HourlyPoint{
			Hour:      hp.Hour,
			ChatCnt:   hp.ChatCnt,
			SearchCnt: hp.SearchCnt,
			UploadCnt: hp.UploadCnt,
			AvgMs:     hp.AvgMs,
		})
	}
	for _, di := range d.Distribution {
		resp.Distribution = append(resp.Distribution, types.DistItem{
			EventType: di.EventType,
			Cnt:       di.Cnt,
		})
	}
	for _, sp := range d.SuccessRate7d {
		resp.SuccessRate7d = append(resp.SuccessRate7d, types.SuccessRatePoint{
			Date:       sp.Date,
			Total:      sp.Total,
			SuccessCnt: sp.SuccessCnt,
			Rate:       sp.Rate,
		})
	}
	return resp
}
