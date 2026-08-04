package handler

import (
	"net/http"

	"github.com/chenjianyu070921-lang/KnoX/cmd/reporter/internal/logic"
	"github.com/chenjianyu070921-lang/KnoX/cmd/reporter/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// DashboardHandler GET /api/v1/analytics/dashboard
func DashboardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewDashboardLogic(r.Context(), svcCtx)
		resp, err := l.Dashboard()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
