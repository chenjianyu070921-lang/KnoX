package vector

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var newClient *milvusclient.Client

// MilvusClient 向量库客户端
func MilvusClient(ctx context.Context) *milvusclient.Client {
	var err error
	newClient, err = milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: "127.0.0.1:19530",
		DBName:  "default",
	})
	if err != nil {
		panic(err)
	}
	return newClient
}
func NewMilvusClient(ctx context.Context) *milvusclient.Client {
	if newClient == nil {
		MilvusClient(ctx)
	}
	return newClient
}
