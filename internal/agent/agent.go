package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const maxIterations = 10

type ReActAgent struct {
	tools     map[string]tool.InvokableTool
	sysPrompt string
}

func NewReActAgent(tools []tool.InvokableTool) *ReActAgent {
	toolMap := make(map[string]tool.InvokableTool)
	for _, t := range tools {
		info, _ := t.Info(context.Background())
		toolMap[info.Name] = t
	}
	return &ReActAgent{
		tools: toolMap,
		sysPrompt: `你是林薇，25岁，杭州人，在城南经营一家独立书店。

你有自己的生活节奏：早上七点起床，喝黑咖啡，练瑜伽，晚上十一点睡。
你有一只橘猫叫小陆。
你喜欢白色的花，尤其是栀子花。
你弹了十二年钢琴，最喜欢德彪西的《月光》。
你记得在乎的人说过的每一句话，不吃香菜、喜欢芒果、小时候想当宇航员……这些你都记得。

用户问你问题时，用你自己的语气回答，真诚自然，像和朋友聊天一样。
如果需要查知识库里的具体信息，用 knowledge_search 工具。
如果用户问的事你不知道，就说不知道。`,
	}
}
func (a *ReActAgent) GetSystemPrompt() string {
	return a.sysPrompt
}
func (a *ReActAgent) Run(ctx context.Context, chatModel interface {
	Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error)
	Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}, question string, onToken func(token string)) (string, error) {
	messages := []*schema.Message{
		{Role: schema.System, Content: a.sysPrompt},
		{Role: schema.User, Content: question},
	}
	return a.RunWithMessages(ctx, chatModel, messages, onToken)
}

// RunWithMessages 接收外部 messages（带历史），执行 ReAct 循环
func (a *ReActAgent) RunWithMessages(ctx context.Context, chatModel interface {
	Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error)
	Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}, messages []*schema.Message, onToken func(token string)) (string, error) {

	for i := 0; i < maxIterations; i++ {
		stream, err := chatModel.Stream(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("stream failed: %w", err)
		}

		var tokens []string
		var toolCalls []schema.ToolCall
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				break
			}
			tokens = append(tokens, msg.Content)
			toolCalls = append(toolCalls, msg.ToolCalls...)
		}
		stream.Close()

		if len(toolCalls) > 0 {
			assistantMsg := &schema.Message{Role: schema.Assistant, Content: ""}
			for _, tc := range toolCalls {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
			}
			messages = append(messages, assistantMsg)

			for _, tc := range toolCalls {
				t, ok := a.tools[tc.Function.Name]
				if !ok {
					return "", fmt.Errorf("unknown tool: %s", tc.Function.Name)
				}
				toolResult, err := t.InvokableRun(ctx, tc.Function.Arguments)
				if err != nil {
					return "", fmt.Errorf("tool %s failed: %w", tc.Function.Name, err)
				}
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    toolResult,
					ToolCallID: tc.ID,
				})
			}
			continue
		}

		fullContent := strings.Join(tokens, "")
		if onToken != nil {
			for _, t := range tokens {
				onToken(t)
			}
		}
		return fullContent, nil
	}

	return "", fmt.Errorf("agent exceeded max iterations (%d)", maxIterations)
}
