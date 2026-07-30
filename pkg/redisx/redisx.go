package redisx

import (
	"github.com/redis/go-redis/v9"
)

var client *redis.Client // 全局单例，Init 后可用

func RedisInit(addr string) *redis.Client {

	client = redis.NewClient(&redis.Options{
		Addr: addr, // 地址，如 127.0.0.1:6379
	})

	return client
}

// GetClient 返回全局 Redis 客户端
func GetClient(addr string) *redis.Client {
	if client == nil {
		RedisInit(addr)
	}
	return client
}
