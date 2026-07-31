// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
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
	answer, err := l.svcCtx.ReActAgent.RunWithMessages(l.ctx, l.svcCtx.ChatModel, messages, onToken)
	if err != nil {
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
