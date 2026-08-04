package requestid

import "context"

type ctxKey struct{}

// WithID 将请求 ID 放入 context，供逻辑层与埋点使用。
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext 从 context 读取请求 ID，不存在时返回空串。
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
