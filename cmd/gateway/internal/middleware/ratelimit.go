package middleware

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/yourname/know/internal/errcode"
)

// WithRateLimit 用 go-zero PeriodLimit 包裹 handler，对超限请求返回 429。
// Period 秒内允许 Quota 次请求。
// Redis 故障时放行，不阻断业务。
func WithRateLimit(next http.HandlerFunc, rds *redis.Redis, quota, period int) http.HandlerFunc {
	limiter := limit.NewPeriodLimit(period, quota, rds, "knox:rl:")
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Request-Id")
		if key == "" {
			key = r.RemoteAddr
			if idx := strings.LastIndex(key, ":"); idx > 0 {
				key = key[:idx] // 去掉端口，只留 IP
			}
		}
		code, err := limiter.Take(key)
		if err != nil {
			// Redis 不可用时放行，避免限流组件故障把正常流量挡掉
			next(w, r)
			return
		}
		if code == limit.OverQuota {
			httpx.ErrorCtx(r.Context(), w, errcode.New(errcode.RateLimitExceeded))
			return
		}
		next(w, r)
	}
}
