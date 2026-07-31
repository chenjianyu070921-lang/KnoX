package distlock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	unlockLuaScript = `
local val = redis.call("GET", KEYS[1])
if val == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`
)

type DistLock struct {
	rdb *redis.Client
}

func NewDistLock(rdb *redis.Client) *DistLock {
	return &DistLock{rdb: rdb}
}
func (r *DistLock) TryLock(ctx context.Context, key string, token string, duration time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, key, token, duration).Result()
}
func (r *DistLock) Unlock(ctx context.Context, key string, token string) error {
	//1.先获取key的值
	//2.在校验token是否相同
	//3.相同则删除 | 不相同则返回

	result, err := r.rdb.Eval(ctx, unlockLuaScript, []string{key}, token).Result()
	if err != nil {
		return err
	}
	// 断言返回值
	deleted, ok := result.(int64)
	if !ok {
		return errors.New("断言失败")
	}
	if deleted != 1 {
		return errors.New("解锁失败：token不匹配，不是锁持有者")
	}
	return nil
}
