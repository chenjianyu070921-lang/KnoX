package session

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	ID       string
	Messages []*schema.Message
	Created  time.Time
}

type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

const (
	maxHistoryMessages = 20
	maxHistoryTokens   = 12000
)

func NewStore(rdb *redis.Client) *Store {
	return &Store{
		rdb: rdb,
		ttl: 24 * time.Hour,
	}
}

func (s *Store) GetOrCreate(id string) *Session {
	ctx := context.Background()

	if id != "" {
		data, err := s.rdb.Get(ctx, "session:"+id).Bytes()
		if err == nil {
			var session Session
			if json.Unmarshal(data, &session) == nil {
				session.Messages = trimHistory(session.Messages)
				return &session
			}
		}
	}

	session := &Session{
		ID:       uuid.New().String(),
		Messages: make([]*schema.Message, 0),
		Created:  time.Now(),
	}
	s.save(session)
	return session
}

func (s *Store) Save(session *Session) {
	s.save(session)
}

func (s *Store) save(session *Session) {
	session.Messages = trimHistory(session.Messages)
	data, _ := json.Marshal(session)
	s.rdb.Set(context.Background(), "session:"+session.ID, data, s.ttl)
}

// trimHistory 限制历史条数与 token 预算（按字符数近似），始终保留最新消息。
func trimHistory(messages []*schema.Message) []*schema.Message {
	if len(messages) > maxHistoryMessages {
		messages = messages[len(messages)-maxHistoryMessages:]
	}
	total := 0
	for i := len(messages) - 1; i >= 0; i-- {
		tokens := utf8.RuneCountInString(messages[i].Content)
		if total > 0 && total+tokens > maxHistoryTokens {
			messages = messages[i+1:]
			break
		}
		total += tokens
	}
	return messages
}
