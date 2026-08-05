package logic

import (
	"context"
	"time"

	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DocListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDocListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DocListLogic {
	return &DocListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DocListLogic) DocList(req *types.DocListReq) (*types.DocListResp, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.Size
	if size < 1 || size > 100 {
		size = 20
	}

	start := time.Now()
	docs, total, err := l.svcCtx.DocRepo.List(l.ctx, page, size)
	if err != nil {
		l.svcCtx.Analytics.LogSearch(
			time.Since(start).Milliseconds(),
			false, "", "doc_list", 0,
		)
		return nil, err
	}

	items := make([]types.DocItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, types.DocItem{
			DocID:     doc.DocID,
			Title:     doc.Title,
			DocType:   doc.DocType,
			FileUrl:   doc.FileUrl,
			CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
			Version:   doc.Version,
		})
	}

	l.svcCtx.Analytics.LogSearch(
		time.Since(start).Milliseconds(),
		true, "", "doc_list", len(items),
	)

	return &types.DocListResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}
