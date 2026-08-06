// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/chenjianyu070921-lang/KnoX/internal/breaker"
	"github.com/chenjianyu070921-lang/KnoX/internal/errcode"
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

func (l *SearchLogic) Search(req *types.SearchReq) (resp *types.SearchResp, err error) {
	if err := validateSearchRequest(req); err != nil {
		return nil, err
	}

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

	cfg := l.svcCtx.Config
	topK := normalizeTopK(req.TopK, cfg.Retrieval.DefaultTopK, cfg.Retrieval.MaxTopK)

	//获取LLM的各大组件
	embedding, err := vector.GetEmbeddingClient(l.ctx, cfg.Ollama.URL, cfg.Ollama.Model)
	if err != nil {
		return nil, err
	}
	milvus, err := vector.GetMilvusClient(l.ctx, cfg.Milvus.Addr, cfg.Milvus.DBName)
	if err != nil {
		return nil, err
	}
	retrieverClient, err := vector.RetrieverClient(l.ctx, embedding, milvus, cfg.Milvus.Collection, cfg.Milvus.VectorField, cfg.Retrieval.DefaultTopK)
	if err != nil {
		return nil, err
	}
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

const maxQueryRunes = 500

func validateSearchRequest(req *types.SearchReq) error {
	if strings.TrimSpace(req.Query) == "" {
		return errcode.New(errcode.InvalidParam, "query 不能为空")
	}
	if utf8.RuneCountInString(req.Query) > maxQueryRunes {
		return errcode.New(errcode.InvalidParam, "query 过长")
	}
	return nil
}

func normalizeTopK(requested, defaultTopK, maxTopK int) int {
	if requested <= 0 {
		requested = defaultTopK
	}
	if maxTopK > 0 && requested > maxTopK {
		requested = maxTopK
	}
	if requested <= 0 {
		requested = 5
	}
	return requested
}
