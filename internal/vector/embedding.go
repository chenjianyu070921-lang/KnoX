package vector

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
)

var (
	embedderOnce sync.Once
	embedder     *ollama.Embedder
	embedderErr  error
)

// GetEmbeddingClient 返回按配置初始化的 Embedding 客户端（进程内单例）。
func GetEmbeddingClient(ctx context.Context, baseURL, model string) (*ollama.Embedder, error) {
	embedderOnce.Do(func() {
		embedder, embedderErr = ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
			BaseURL: baseURL,
			Model:   model,
			Timeout: 10 * time.Second,
		})
	})
	if embedderErr != nil {
		return nil, embedderErr
	}
	return embedder, nil
}
