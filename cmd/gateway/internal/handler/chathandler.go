// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/logic"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/chenjianyu070921-lang/KnoX/internal/errcode"
	"github.com/chenjianyu070921-lang/KnoX/internal/requestid"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("streaming not supported"))
			return
		}

		ctx := requestid.WithID(r.Context(), r.Header.Get("X-Request-Id"))
		l := logic.NewChatLogic(ctx, svcCtx)
		resp, err := l.Chat(&req,
			func(token string) {
				data, _ := json.Marshal(map[string]string{"content": token})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			})
		if err != nil {
			code := errcode.InternalError
			msg := errcode.Msg(errcode.InternalError)
			if bizErr, ok := err.(*errcode.BizError); ok {
				code = bizErr.Code
				msg = bizErr.Message
			}
			data, _ := json.Marshal(map[string]interface{}{"code": code, "error": msg})
			fmt.Fprintf(w, "data: %s\n\n", data)
		} else if resp != nil && resp.SessionId != "" {
			data, _ := json.Marshal(map[string]string{"sessionId": resp.SessionId})
			fmt.Fprintf(w, "data: %s\n\n", data)
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}
