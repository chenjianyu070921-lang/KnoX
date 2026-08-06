package vector

import (
	"context"
	"sync"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var (
	retrieverOnce   sync.Once
	retrieverClient *retriever.Retriever
	retrieverErr    error
)

// RetrieverClient 返回按配置初始化的 Milvus 检索器（进程内单例）。
func RetrieverClient(ctx context.Context, embedding *ollama.Embedder, milvus *milvusclient.Client, collection, vectorField string, topK int) (*retriever.Retriever, error) {
	retrieverOnce.Do(func() {
		retrieverClient, retrieverErr = retriever.NewRetriever(ctx, &retriever.RetrieverConfig{
			Client:       milvus,
			Collection:   collection,
			VectorField:  vectorField,
			OutputFields: []string{"content", "metadata"},
			TopK:         topK,
			SearchMode:   search_mode.NewApproximate(retriever.COSINE),
			Embedding:    embedding,
		})
	})
	return retrieverClient, retrieverErr
}
