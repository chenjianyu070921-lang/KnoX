package handler

import (
	"net/http"
	"strings"

	"github.com/yourname/know/cmd/gateway/internal/logic"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/errcode"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// DocListHandler GET /api/v1/docs
func DocListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DocListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, errcode.New(errcode.BadRequest, "参数错误: "+err.Error()))
			return
		}
		l := logic.NewDocListLogic(r.Context(), svcCtx)
		resp, err := l.DocList(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// DocDetailHandler GET /api/v1/docs/:docId
func DocDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 URL path 中提取 docId: /api/v1/docs/DOC_ID
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/docs/")
		docID := strings.TrimSpace(path)
		if docID == "" {
			httpx.ErrorCtx(r.Context(), w, errcode.New(errcode.BadRequest, "docId 缺失"))
			return
		}
		l := logic.NewDocDetailLogic(r.Context(), svcCtx)
		resp, err := l.DocDetail(docID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
