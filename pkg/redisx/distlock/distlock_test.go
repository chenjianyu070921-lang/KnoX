package distlock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniRedis 启动内存 Redis，返回锁实例和 miniredis（用于 FastForward 测过期）
func setupMiniRedis(t *testing.T) (*DistLock, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewDistLock(rdb), mr
}

func TestTryLock_Success(t *testing.T) {
	lock, _ := setupMiniRedis(t)
	ctx := context.Background()

	ok, err := lock.TryLock(ctx, "mykey", "token1", 10*time.Second)
	require.NoError(t, err)
	assert.True(t, ok, "第一次 TryLock 应该成功")
}

func TestTryLock_AlreadyLocked(t *testing.T) {
	lock, _ := setupMiniRedis(t)
	ctx := context.Background()

	// 先拿到锁
	ok, err := lock.TryLock(ctx, "mykey", "token1", 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	// 再抢同一把锁，应该失败
	ok, err = lock.TryLock(ctx, "mykey", "token2", 10*time.Second)
	require.NoError(t, err)
	assert.False(t, ok, "锁已被 token1 持有，token2 应该抢不到")
}

func TestUnlock_Success(t *testing.T) {
	lock, _ := setupMiniRedis(t)
	ctx := context.Background()

	ok, err := lock.TryLock(ctx, "mykey", "token1", 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	err = lock.Unlock(ctx, "mykey", "token1")
	assert.NoError(t, err, "持锁者自己释放应该成功")
}

func TestUnlock_WrongToken(t *testing.T) {
	lock, _ := setupMiniRedis(t)
	ctx := context.Background()

	ok, err := lock.TryLock(ctx, "mykey", "token1", 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	err = lock.Unlock(ctx, "mykey", "token2")
	assert.Error(t, err, "非持锁者释放应该报错")
}

func TestLock_Expire(t *testing.T) {
	lock, mr := setupMiniRedis(t)
	ctx := context.Background()

	// 拿到锁，设 1 秒过期
	ok, err := lock.TryLock(ctx, "mykey", "token1", 1*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	// 还没过期，抢不到
	ok, err = lock.TryLock(ctx, "mykey", "token2", 10*time.Second)
	require.NoError(t, err)
	assert.False(t, ok, "锁未过期，应该抢不到")

	// 快进 2 秒，锁过期
	mr.FastForward(2 * time.Second)

	// 现在能抢到了
	ok, err = lock.TryLock(ctx, "mykey", "token2", 10*time.Second)
	require.NoError(t, err)
	assert.True(t, ok, "锁过期后应该能抢到")
}

func TestUnlock_KeyAlreadyExpired(t *testing.T) {
	lock, mr := setupMiniRedis(t)
	ctx := context.Background()

	ok, err := lock.TryLock(ctx, "mykey", "token1", 1*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	// 快进，锁过期消失
	mr.FastForward(2 * time.Second)

	// key 已经不存在了，Unlock 应该报错（不是持锁者）
	err = lock.Unlock(ctx, "mykey", "token1")
	assert.Error(t, err, "key 已过期，释放应该失败")
}

func TestTryLock_DifferentKeys(t *testing.T) {
	lock, _ := setupMiniRedis(t)
	ctx := context.Background()

	// 两个不同的 key 互不影响
	ok, err := lock.TryLock(ctx, "key1", "t1", 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = lock.TryLock(ctx, "key2", "t2", 10*time.Second)
	require.NoError(t, err)
	assert.True(t, ok, "不同的 key 应该能同时拿到锁")
}
