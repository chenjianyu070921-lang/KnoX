package vector

import (
	"context"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var retrieverClient *retriever.Retriever

func RetrieverClient(ctx context.Context, embedding *ollama.Embedder, milvus *milvusclient.Client) *retriever.Retriever {
	if retrieverClient == nil {
		var err error
		retrieverClient, err = retriever.NewRetriever(ctx, &retriever.RetrieverConfig{
			Client:       milvus,
			Collection:   "knox_docs",
			VectorField:  "vector",
			OutputFields: []string{"content", "metadata"},
			TopK:         5,
			SearchMode:   search_mode.NewApproximate(retriever.COSINE),
			Embedding:    embedding,
		})
		if err != nil {
			panic(err)
		}
	}
	return retrieverClient
}
