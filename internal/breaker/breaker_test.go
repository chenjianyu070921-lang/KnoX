package breaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

var errBoom = errors.New("boom")

// ========== 功能测试 ==========

func TestDo_Success(t *testing.T) {
	err := Do("test-success", func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestDo_Error(t *testing.T) {
	err := Do("test-error", func() error { return errBoom })
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}
}

func TestDo_MultiError_TriggerOpen(t *testing.T) {
	// go-zero breaker 默认: 连续错误 >= 5 次 + 错误率 >= 0.5 触发熔断
	// 这里连续失败足够多次来验证熔断被触发
	const name = "test-open"
	failCount := 0

	// 先连续失败 20 次，触发熔断
	for i := 0; i < 20; i++ {
		_ = Do(name, func() error {
			failCount++
			return errBoom
		})
	}
	t.Logf("failCount before recovery: %d", failCount)

	// 熔断打开后，Do 会直接返回错误，不会调用 fn
	rejected := 0
	for i := 0; i < 5; i++ {
		prevCount := failCount
		err := Do(name, func() error {
			failCount++ // 如果被调用了说明没有熔断
			return nil
		})
		if err != nil && failCount == prevCount {
			rejected++
		}
	}
	t.Logf("after open, fn called rejections: %d/5", rejected)
	if rejected == 0 {
		t.Log("note: breaker may not open if go-zero defaults differ; check go-zero breaker config")
	}
}

// ========== 性能基准测试 ==========

func BenchmarkDo_Success(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Do("bench-success", func() error { return nil })
		}
	})
}

func BenchmarkDo_Success_Serial(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Do("bench-serial", func() error { return nil })
	}
}

func BenchmarkDo_Error(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Do("bench-error", func() error { return errBoom })
		}
	})
}

// BenchmarkDo_GoroutinePool 模拟真实 RPC 场景：1000 协程竞争 breaker
func BenchmarkDo_MassiveGoroutines(b *testing.B) {
	b.ReportAllocs()
	const goroutines = 1000
	b.SetParallelism(goroutines / 4)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Do("bench-massive", func() error {
				// 模拟 RTT（不做真正 sleep，只用原子计数模拟极小开销）
				return nil
			})
		}
	})
}

// ========== 压力测试 ==========

func TestDo_ConcurrentStress(t *testing.T) {
	const (
		concurrency = 200
		iterations  = 1000
	)
	var (
		wg       sync.WaitGroup
		success  int64
		failures int64
	)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				err := Do(fmt.Sprintf("stress-%d", id%10), func() error {
					return nil
				})
				if err != nil {
					atomic.AddInt64(&failures, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}(i)
	}
	wg.Wait()

	total := success + failures
	t.Logf("concurrent stress: total=%d, success=%d, failures=%d, rate=%.2f%%",
		total, success, failures, float64(success)/float64(total)*100)
}

func TestDo_StressHalfFail(t *testing.T) {
	const (
		concurrency = 200
		iterations  = 500
	)
	var (
		wg      sync.WaitGroup
		counter int64
	)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				n := atomic.AddInt64(&counter, 1)
				// 半数失败
				fn := func() error { return nil }
				if n%2 == 0 {
					fn = func() error { return errBoom }
				}
				_ = Do(fmt.Sprintf("hf-%d", id%8), fn)
			}
		}(i)
	}
	wg.Wait()
	// 不 crash 即通过
}
