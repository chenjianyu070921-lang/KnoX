// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/chenjianyu070921-lang/KnoX/internal/breaker"
	"github.com/chenjianyu070921-lang/KnoX/internal/errcode"
	"github.com/chenjianyu070921-lang/KnoX/internal/requestid"

	"github.com/cloudwego/eino/schema"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(req *types.ChatReq, onToken func(string)) (resp *types.ChatResp, err error) {
	if err := validateChatRequest(req); err != nil {
		return nil, err
	}

	start := time.Now()
	var answer string
	// defer recover：把 panic 转成 error 而不是 500，并记详细 stack 到日志便于定位
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("[CHAT_PANIC] question=%s panic=%v\n%s", req.Question, r, debug.Stack())
			err = fmt.Errorf("chat panic: %v", r)
		}
		// 埋点（无论正常/异常都记）
		l.svcCtx.Analytics.LogChat(
			time.Since(start).Milliseconds(),
			err == nil,
			requestid.FromContext(l.ctx),
			len(req.Question),
			len(answer),
			0,
		)
	}()

	session := l.svcCtx.SessionStore.GetOrCreate(req.SessionId)

	// 搭完整 messages：System 提示 + 历史对话 + 当前问题
	messages := []*schema.Message{
		{Role: schema.System, Content: l.svcCtx.ReActAgent.GetSystemPrompt()},
	}
	messages = append(messages, session.Messages...)
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: req.Question,
	})
	err = breaker.Do(breaker.ARK, func() error {
		var innerErr error
		answer, innerErr = l.svcCtx.ReActAgent.RunWithMessages(l.ctx, l.svcCtx.ChatModel, messages, onToken)
		return innerErr
	})
	if err != nil {
		logx.Errorf("[CHAT_ERR] question=%s err=%v", req.Question, err)
		return nil, err
	}
	// 把回答也存进历史
	session.Messages = append(session.Messages,
		&schema.Message{Role: schema.User, Content: req.Question},
		&schema.Message{Role: schema.Assistant, Content: answer},
	)
	l.svcCtx.SessionStore.Save(session)

	return &types.ChatResp{
		Answer:    answer,
		SessionId: session.ID,
	}, nil
}

const (
	maxQuestionRunes  = 4000
	maxSessionIDRunes = 64
)

func validateChatRequest(req *types.ChatReq) error {
	if strings.TrimSpace(req.Question) == "" {
		return errcode.New(errcode.InvalidParam, "question 不能为空")
	}
	if utf8.RuneCountInString(req.Question) > maxQuestionRunes {
		return errcode.New(errcode.InvalidParam, "question 过长")
	}
	if utf8.RuneCountInString(req.SessionId) > maxSessionIDRunes {
		return errcode.New(errcode.InvalidParam, "sessionId 过长")
	}
	return nil
}
