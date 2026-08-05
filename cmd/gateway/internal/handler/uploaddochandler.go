// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/yourname/know/cmd/gateway/internal/logic"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/errcode"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadDocHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get("X-Request-Id")
		if requestId == "" {
			httpx.ErrorCtx(r.Context(), w, errcode.New(errcode.BadRequest, "请求头 X-Request-Id 缺失"))
			return
		}

		var req types.UploadDocRequest
		req.RequestId = requestId
		l := logic.NewUploadDocLogic(r.Context(), svcCtx)

		resp, err := l.UploadDoc(&req, r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
