package vector

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
)

var embedder *ollama.Embedder

func EmbeddingClient(ctx context.Context) *ollama.Embedder {
	var err error
	embedder, err = ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: "http://localhost:11434",
		Model:   "bge-m3",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	return embedder
}
func GetEmbeddingClient(ctx context.Context) *ollama.Embedder {
	if embedder == nil {
		EmbeddingClient(ctx)
	}
	return embedder
}
