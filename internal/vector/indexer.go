package vector

import (
	"context"
	"sync"

	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var (
	indexerOnce sync.Once
	indexer     *milvus2.Indexer
	indexerErr  error
)

// IndexerClient 创建 Milvus 索引器，collection/vectorField/dimension 由配置注入。
func IndexerClient(ctx context.Context, client *milvusclient.Client, embedder embedding.Embedder, collection, vectorField string, dimension int) (*milvus2.Indexer, error) {
	indexerConfig := milvus2.IndexerConfig{
		Client:     client,     // Milvus 客户端实例
		Collection: collection, // 目标集合名称
		Embedding:  embedder,   // Embedding 向量化实例
		Vector: &milvus2.VectorConfig{
			Dimension:    int64(dimension),                                                // 与 embedding 模型维度匹配
			MetricType:   milvus2.COSINE,                                                  // 距离度量方式，用于向量相似度检索
			IndexBuilder: milvus2.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200), // HNSW 高性能内存索引
			VectorField:  vectorField,
		},
	}
	return milvus2.NewIndexer(ctx, &indexerConfig)
}

// GetIndexerClient 返回进程内单例索引器。
func GetIndexerClient(ctx context.Context, client *milvusclient.Client, embedder embedding.Embedder, collection, vectorField string, dimension int) (*milvus2.Indexer, error) {
	indexerOnce.Do(func() {
		indexer, indexerErr = IndexerClient(ctx, client, embedder, collection, vectorField, dimension)
	})
	return indexer, indexerErr
}
