// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"time"

	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/breaker"
	"github.com/yourname/know/internal/vector"

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

func (l *SearchLogic) Search(req *types.SearchReq) (resp *types.SearchResp, err error) {
	start := time.Now()
	var resultCount int
	defer func() {
		l.svcCtx.Analytics.LogSearch(
			time.Since(start).Milliseconds(),
			err == nil,
			"",
			req.Query,
			resultCount,
		)
	}()

	//获取LLM的各大组件
	embedding := vector.GetEmbeddingClient(l.ctx)
	milvus := vector.MilvusClient(l.ctx)
	retriever := vector.RetrieverClient(l.ctx, embedding, milvus)
	//Retriever 返回 []*schema.Document
	var docs []*schema.Document
	err = breaker.Do(breaker.Milvus, func() error {
		var innerErr error
		docs, innerErr = retriever.Retrieve(l.ctx, req.Query)
		return innerErr
	})
	if err != nil {
		return nil, err
	}
	//把搜索结果映射成 API 响应
	resultCount = len(docs)
	results := make([]types.SearchResult, 0, len(docs))
	for _, doc := range docs {
		results = append(results, types.SearchResult{
			DocId:   doc.ID, // MetaData 里取 doc_id
			Content: doc.Content,
			Score:   0, // Eino 有些版本把 score 放 MetaData 里
		})
	}
	return &types.SearchResp{
		Results: results,
	}, nil
}
