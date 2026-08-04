package vector

import (
	"context"
	"sync"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var (
	newClient *milvusclient.Client
	Once      sync.Once
)

// MilvusClient 向量库客户端
func MilvusClient(ctx context.Context) *milvusclient.Client {
	var err error
	Once.Do(func() {
		newClient, err = milvusclient.New(ctx, &milvusclient.ClientConfig{
			Address: "127.0.0.1:19530",
			DBName:  "default",
		})
		if err != nil {
			panic(err)
		}
	})

	return newClient
}
func NewMilvusClient(ctx context.Context) *milvusclient.Client {
	if newClient == nil {
		MilvusClient(ctx)
	}
	return newClient
}
