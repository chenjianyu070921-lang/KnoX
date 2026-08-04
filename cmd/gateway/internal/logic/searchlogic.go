// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"time"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/chenjianyu070921-lang/KnoX/internal/breaker"
	"github.com/chenjianyu070921-lang/KnoX/internal/requestid"
	"github.com/chenjianyu070921-lang/KnoX/internal/vector"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchLogic {
	return &SearchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

const (
	defaultSearchTopK = 5
	maxSearchTopK     = 20
)

func (l *SearchLogic) Search(req *types.SearchReq) (resp *types.SearchResp, err error) {
	start := time.Now()
	var resultCount int
	defer func() {
		l.svcCtx.Analytics.LogSearch(
			time.Since(start).Milliseconds(),
			err == nil,
			requestid.FromContext(l.ctx),
			req.Query,
			resultCount,
		)
	}()

	topK := req.TopK
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	if topK > maxSearchTopK {
		topK = maxSearchTopK
	}

	//获取LLM的各大组件
	embedding := vector.GetEmbeddingClient(l.ctx)
	milvus := vector.MilvusClient(l.ctx)
	retrieverClient := vector.RetrieverClient(l.ctx, embedding, milvus)
	//Retriever 返回 []*schema.Document
	var docs []*schema.Document
	err = breaker.Do(breaker.Milvus, func() error {
		var innerErr error
		docs, innerErr = retrieverClient.Retrieve(l.ctx, req.Query, retriever.WithTopK(topK))
		return innerErr
	})
	if err != nil {
		return nil, err
	}

	//把搜索结果映射成 API 响应
	resultCount = len(docs)
	return &types.SearchResp{
		Results: buildSearchResults(docs),
	}, nil
}

// buildSearchResults 将检索结果映射为 API 响应，score 取自 Eino Document 元数据。
func buildSearchResults(docs []*schema.Document) []types.SearchResult {
	results := make([]types.SearchResult, 0, len(docs))
	for _, doc := range docs {
		results = append(results, types.SearchResult{
			DocId:   doc.ID,
			Content: doc.Content,
			Score:   doc.Score(),
		})
	}
	return results
}
