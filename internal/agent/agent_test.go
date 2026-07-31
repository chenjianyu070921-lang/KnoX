package agent

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var errStreamBoom = errors.New("stream boom")

type fakeChatModel struct {
	msgs []*schema.Message
	chunk *schema.Message
	err   error
}

func (f *fakeChatModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, nil
}

func (f *fakeChatModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	f.msgs = msgs
	sr, sw := schema.Pipe[*schema.Message](2)
	if f.chunk != nil {
		sw.Send(f.chunk, nil)
	}
	sw.Send(nil, f.err)
	sw.Close()
	return sr, nil
}

func TestRunWithMessagesReturnsStreamError(t *testing.T) {
	agent := NewReActAgent(nil)
	model := &fakeChatModel{err: errStreamBoom}

	_, err := agent.RunWithMessages(context.Background(), model, []*schema.Message{
		{Role: schema.User, Content: "hi"},
	}, nil)

	if !errors.Is(err, errStreamBoom) {
		t.Fatalf("expected stream error to propagate, got %v", err)
	}
}

func TestRunWithMessagesStreamsTokensBeforeError(t *testing.T) {
	agent := NewReActAgent(nil)
	var tokens []string
	model := &fakeChatModel{
		chunk: &schema.Message{Role: schema.Assistant, Content: "hello"},
		err:   io.EOF,
	}

	_, err := agent.RunWithMessages(context.Background(), model, []*schema.Message{
		{Role: schema.User, Content: "hi"},
	}, func(token string) {
		tokens = append(tokens, token)
	})

	if err != nil {
		t.Fatalf("expected EOF to end stream, got %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "hello" {
		t.Fatalf("expected streamed token hello, got %v", tokens)
	}
}
