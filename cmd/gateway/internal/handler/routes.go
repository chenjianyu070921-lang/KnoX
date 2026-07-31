package handler

import (
	"net/http"

	"github.com/yourname/know/cmd/gateway/internal/middleware"
	"github.com/yourname/know/cmd/gateway/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	rds := serverCtx.GoZeroRedis
	cfg := serverCtx.Config.RateLimit

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/chat",
				Handler: middleware.WithRateLimit(ChatHandler(serverCtx), rds, cfg.Chat.Quota, cfg.Chat.Period),
			},
			{
				Method:  http.MethodPost,
				Path:    "/doc/upload",
				Handler: middleware.WithRateLimit(UploadDocHandler(serverCtx), rds, cfg.Upload.Quota, cfg.Upload.Period),
			},
			{
				Method:  http.MethodGet,
				Path:    "/ping",
				Handler: PingHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/search",
				Handler: middleware.WithRateLimit(SearchHandler(serverCtx), rds, cfg.Search.Quota, cfg.Search.Period),
			},
		},
		rest.WithPrefix("/api/v1"),
	)

	// 统计大盘（只读，不限流）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/analytics/overview",
				Handler: AnalyticsOverviewHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/analytics/trends",
				Handler: AnalyticsTrendsHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/analytics/slow-queries",
				Handler: AnalyticsSlowHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1"),
	)
}
