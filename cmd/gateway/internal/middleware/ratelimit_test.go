package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// fakeRedis 启动内存 Redis，返回 go-zero 原生 Redis 供 PeriodLimit 使用
func fakeRedis(t testing.TB) (*redis.Redis, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: mr.Addr(),
		Type: redis.NodeType,
	})
	return rds, mr.Close
}

// okHandler 始终返回 200 的测试用 handler
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// ========== 功能测试 ==========

func TestWithRateLimit_AllowWithinQuota(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 10, 1) // 1秒 10次

	// 前 10 次都应该放行
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d within quota: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestWithRateLimit_RejectExceedQuota(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 3, 1) // 1秒 3次

	allPass := true
	rejectedCount := 0
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code == http.StatusOK {
			continue
		}
		allPass = false
		if rec.Code == http.StatusBadRequest {
			rejectedCount++
		}
	}
	if allPass {
		t.Fatal("expected some requests to be rejected after quota exceeded")
	}
	t.Logf("rejected %d/10 requests (status 400)", rejectedCount)
}

func TestWithRateLimit_DifferentIPs(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 2, 1) // 1秒每IP 2次

	// IP1 用满配额
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("IP1 request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
	// IP1 第 3 次应该被拒绝
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("IP1 3rd request should be rejected after exceeding quota")
	}

	// IP2 应该不受影响
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:54321"
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("IP2: expected 200, got %d", rec2.Code)
	}
}

func TestWithRateLimit_RequestIDPriority(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 2, 1)

	// X-Request-Id 优先于 RemoteAddr 做限流 key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "req-001")
	req.RemoteAddr = "10.0.0.1:11111"

	// 前 2 次通过
	rec1 := httptest.NewRecorder()
	handler(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("req 1: expected 200, got %d", rec1.Code)
	}
	rec2 := httptest.NewRecorder()
	handler(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("req 2: expected 200, got %d", rec2.Code)
	}
	// 第 3 次拒绝
	rec3 := httptest.NewRecorder()
	handler(rec3, req)
	if rec3.Code == http.StatusOK {
		t.Fatal("req 3 should be rejected for same X-Request-Id")
	}
}

// ========== 性能基准测试 ==========

func BenchmarkWithRateLimit_NoLimit(b *testing.B) {
	// 大配额，评估中间件本身开销
	rds, closer := fakeRedis(b)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 1000000000, 1)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

func BenchmarkWithRateLimit_Serial(b *testing.B) {
	rds, closer := fakeRedis(b)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 10000, 1)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

func BenchmarkWithRateLimit_Parallel(b *testing.B) {
	rds, closer := fakeRedis(b)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 100000, 1)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler(rec, req)
		}
	})
}

func BenchmarkWithRateLimit_MultiIP_Parallel(b *testing.B) {
	rds, closer := fakeRedis(b)
	defer closer()

	handler := WithRateLimit(okHandler, rds, 100000, 1)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ip := fmt.Sprintf("10.0.%d.%d", atomic.AddUint64(&counter, 1)%255, atomic.AddUint64(&counter, 1)%255)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip + ":12345"
		_ = req
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler(rec, req)
		}
	})
}

var counter uint64 // 供 MultiIP benchmark 用

// ========== 并发压力测试 ==========

func TestWithRateLimit_ConcurrentStress(t *testing.T) {
	rds, closer := fakeRedis(t)
	_ = closer // keep alive during test
	defer closer()

	const (
		concurrency = 100
		quota       = 50
		period      = 1
	)
	handler := WithRateLimit(okHandler, rds, quota, period)

	var (
		wg       sync.WaitGroup
		okCount  int64
		rejCount int64
	)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/stress", nil)
			req.RemoteAddr = fmt.Sprintf("10.0.0.%d:12345", id%10) // 10 个不同 IP
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code == http.StatusOK {
				atomic.AddInt64(&okCount, 1)
			} else {
				atomic.AddInt64(&rejCount, 1)
			}
		}(i)
	}
	wg.Wait()

	t.Logf("concurrent stress: %d goroutines, ok=%d, rejected=%d", concurrency, okCount, rejCount)
	if okCount+rejCount != concurrency {
		t.Fatalf("count mismatch: %d+%d != %d", okCount, rejCount, concurrency)
	}
}

func TestWithRateLimit_Burst(t *testing.T) {
	rds, closer := fakeRedis(t)
	defer closer()

	const (
		burstSize = 500
		quota     = 100
		period    = 1
	)
	handler := WithRateLimit(okHandler, rds, quota, period)

	var (
		wg       sync.WaitGroup
		okCount  int64
		rejCount int64
	)

	start := time.Now()
	wg.Add(burstSize)
	for i := 0; i < burstSize; i++ {
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/burst", nil)
			req.RemoteAddr = fmt.Sprintf("192.168.1.%d:8080", id%50)
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code == http.StatusOK {
				atomic.AddInt64(&okCount, 1)
			} else {
				atomic.AddInt64(&rejCount, 1)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("burst %d requests in %v (%.0f req/s), ok=%d, rejected=%d",
		burstSize, elapsed, float64(burstSize)/elapsed.Seconds(), okCount, rejCount)
}
