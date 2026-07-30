package session

import (
	"context"
	"encoding/json"
	"time"

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
	data, _ := json.Marshal(session)
	s.rdb.Set(context.Background(), "session:"+session.ID, data, s.ttl)
}
