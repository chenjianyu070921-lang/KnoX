package handler

import (
	"net/http"

	"github.com/chenjianyu070921-lang/KnoX/cmd/reporter/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterRoutes(server *rest.Server, svcCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/dashboard",
				Handler: DashboardHandler(svcCtx),
			},
		},
		rest.WithPrefix("/api/v1/analytics"),
	)
}
