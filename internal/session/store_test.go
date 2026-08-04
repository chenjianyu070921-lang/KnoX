package session

import (
	"context"
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
