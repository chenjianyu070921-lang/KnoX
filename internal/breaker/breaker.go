package breaker

import (
	"errors"

	"github.com/yourname/know/internal/errcode"
	gobreaker "github.com/zeromicro/go-zero/core/breaker"
)

// 每个下游独立命名，各自维护自己的熔断状态机
const (
	ARK    = "ark"
	Milvus = "milvus"
	Ollama = "ollama"
)

// Do 执行 fn，由对应 name 的熔断器保护。
// 熔断打开时直接返回 error，不走 fn；关闭/半开时正常执行。
func Do(name string, fn func() error) error {
	err := gobreaker.Do(name, fn)
	if errors.Is(err, gobreaker.ErrServiceUnavailable) {
		return errcode.New(errcode.CircuitBreakerOpen)
	}
	return err
}
