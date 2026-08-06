package logic

import (
	"strings"
	"testing"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
)

func TestValidateChatRequest(t *testing.T) {
	if err := validateChatRequest(&types.ChatReq{Question: ""}); err == nil {
		t.Fatal("expected empty question to fail")
	}
	if err := validateChatRequest(&types.ChatReq{Question: "   "}); err == nil {
		t.Fatal("expected whitespace question to fail")
	}
	if err := validateChatRequest(&types.ChatReq{Question: "你好"}); err != nil {
		t.Fatalf("expected valid question to pass: %v", err)
	}
	if err := validateChatRequest(&types.ChatReq{Question: strings.Repeat("长", 4001)}); err == nil {
		t.Fatal("expected oversized question to fail")
	}
	if err := validateChatRequest(&types.ChatReq{Question: "你好", SessionId: strings.Repeat("s", 65)}); err == nil {
		t.Fatal("expected oversized sessionId to fail")
	}
	if err := validateChatRequest(&types.ChatReq{Question: "你好", SessionId: strings.Repeat("s", 64)}); err != nil {
		t.Fatalf("expected 64-rune sessionId to pass: %v", err)
	}
}
