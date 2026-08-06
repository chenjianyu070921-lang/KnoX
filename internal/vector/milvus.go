package vector

import (
	"context"
	"sync"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var (
	milvusOnce   sync.Once
	milvusClient *milvusclient.Client
	milvusErr    error
)

// GetMilvusClient 返回按配置初始化的 Milvus 客户端（进程内单例）。
func GetMilvusClient(ctx context.Context, addr, dbName string) (*milvusclient.Client, error) {
	milvusOnce.Do(func() {
		milvusClient, milvusErr = milvusclient.New(ctx, &milvusclient.ClientConfig{
			Address: addr,
			DBName:  dbName,
		})
	})
	if milvusErr != nil {
		return nil, milvusErr
	}
	return milvusClient, nil
}
