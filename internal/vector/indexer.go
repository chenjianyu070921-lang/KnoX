package vector

import (
	"context"

	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var indexer *milvus2.Indexer

func IndexerClient(ctx context.Context, newClient *milvusclient.Client, embedder embedding.Embedder) (*milvus2.Indexer, error) {
	indexerConfig := milvus2.IndexerConfig{
		Client:     newClient,   //Milvus 客户端实例
		Collection: "knox_docs", //目标集合名称
		Embedding:  embedder,    //Embedding 向量化实例
		Vector: &milvus2.VectorConfig{
			Dimension:    1024,                                                            // 与 embedding 模型维度匹配
			MetricType:   milvus2.COSINE,                                                  //距离度量方式，用于向量相似度检索
			IndexBuilder: milvus2.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200), //用来定义 Milvus 向量索引类型，你这里使用 HNSW（主流高性能内存索引）
		},
	}
	var err error
	indexer, err = milvus2.NewIndexer(ctx, &indexerConfig)
	if err != nil {
		panic(err)
	}
	return indexer, nil
}
func GetIndexerClient(ctx context.Context, newClient *milvusclient.Client, embedder embedding.Embedder) *milvus2.Indexer {
	var err error
	if indexer == nil {
		indexer, err = IndexerClient(ctx, newClient, embedder)
		if err != nil {
			panic(err)
		}
	}
	return indexer
}
