package logic

import (
	"context"

	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/errcode"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DocDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDocDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DocDetailLogic {
	return &DocDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DocDetailLogic) DocDetail(docID string) (*types.DocDetailResp, error) {
	doc, err := l.svcCtx.DocRepo.GetByDocID(l.ctx, docID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(errcode.DocNotFound, "文档不存在")
		}
		return nil, err
	}
	return &types.DocDetailResp{
		DocItem: types.DocItem{
			DocID:     doc.DocID,
			Title:     doc.Title,
			DocType:   doc.DocType,
			FileUrl:   doc.FileUrl,
			CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
			Version:   doc.Version,
		},
		Content: doc.Content,
	}, nil
}
