package logic

import (
	"context"

	"github.com/chenjianyu070921-lang/KnoX/cmd/reporter/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/internal/analytics"

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

func (l *DashboardLogic) Dashboard() (*analytics.DashboardData, error) {
	if l.svcCtx.Analytics == nil {
		return &analytics.DashboardData{}, nil
	}
	return l.svcCtx.Analytics.Dashboard()
}
