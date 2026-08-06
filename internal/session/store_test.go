package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewStore(rdb), mr
}

func TestGetOrCreate_CreatesNewSession(t *testing.T) {
	store, _ := newTestStore(t)
	s := store.GetOrCreate("")
	if s.ID == "" {
		t.Fatal("expected generated session id")
	}
	if len(s.Messages) != 0 {
		t.Fatalf("expected empty messages, got %d", len(s.Messages))
	}
}

func TestGetOrCreate_ReturnsSavedSession(t *testing.T) {
	store, _ := newTestStore(t)
	s := store.GetOrCreate("")
	s.Messages = append(s.Messages, &schema.Message{Role: schema.User, Content: "hi"})
	store.Save(s)

	got := store.GetOrCreate(s.ID)
	if got.ID != s.ID {
		t.Fatalf("expected session %s, got %s", s.ID, got.ID)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "hi" {
		t.Fatalf("unexpected messages: %+v", got.Messages)
	}
}

func TestGetOrCreate_UnknownIDCreatesNew(t *testing.T) {
	store, _ := newTestStore(t)
	s := store.GetOrCreate("not-exist")
	if s.ID == "not-exist" {
		t.Fatal("expected a new generated id")
	}
}

func TestSave_ExpiresAfterTTL(t *testing.T) {
	store, mr := newTestStore(t)
	s := store.GetOrCreate("")
	store.Save(s)

	mr.FastForward(25 * time.Hour)
	exists := store.rdb.Exists(context.Background(), "session:"+s.ID).Val()
	if exists != 0 {
		t.Fatalf("expected session key to expire, exists=%d", exists)
	}
}

func TestSave_TrimsHistoryToMessageCap(t *testing.T) {
	store, _ := newTestStore(t)
	s := &Session{ID: "s1", Messages: makeShortMessages(25)}

	store.Save(s)
	got := store.GetOrCreate("s1")

	if len(got.Messages) != 20 {
		t.Fatalf("expected 20 messages, got %d", len(got.Messages))
	}
	if got.Messages[0].Content != "msg-5" || got.Messages[19].Content != "msg-24" {
		t.Fatalf("expected latest messages to be kept, first=%s last=%s", got.Messages[0].Content, got.Messages[19].Content)
	}
}

func TestSave_TrimsHistoryToTokenBudget(t *testing.T) {
	store, _ := newTestStore(t)
	s := &Session{ID: "s2", Messages: makeLongMessages(15, 1000)}

	store.Save(s)
	got := store.GetOrCreate("s2")

	if len(got.Messages) != 11 {
		t.Fatalf("expected 11 messages within token budget, got %d", len(got.Messages))
	}
	if !strings.HasPrefix(got.Messages[0].Content, "msg-4") {
		t.Fatalf("expected oldest dropped messages, first=%s", got.Messages[0].Content)
	}
}

func TestSave_KeepsNewestMessageEvenIfOversized(t *testing.T) {
	store, _ := newTestStore(t)
	s := &Session{ID: "s3", Messages: []*schema.Message{
		{Role: schema.User, Content: "old-" + strings.Repeat("x", 100)},
		{Role: schema.User, Content: strings.Repeat("y", 20000)},
	}}

	store.Save(s)
	got := store.GetOrCreate("s3")

	if len(got.Messages) != 1 {
		t.Fatalf("expected only newest message to survive, got %d", len(got.Messages))
	}
	if got.Messages[0].Content != strings.Repeat("y", 20000) {
		t.Fatal("expected newest message content to be preserved")
	}
}

func makeShortMessages(n int) []*schema.Message {
	messages := make([]*schema.Message, 0, n)
	for i := 0; i < n; i++ {
		messages = append(messages, &schema.Message{Role: schema.User, Content: "msg-" + itoa(i)})
	}
	return messages
}

func makeLongMessages(n, size int) []*schema.Message {
	messages := make([]*schema.Message, 0, n)
	for i := 0; i < n; i++ {
		messages = append(messages, &schema.Message{Role: schema.User, Content: "msg-" + itoa(i) + strings.Repeat("x", size)})
	}
	return messages
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
