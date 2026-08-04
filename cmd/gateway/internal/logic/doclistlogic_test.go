package logic

import (
	"context"
	"testing"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/svc"
	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/chenjianyu070921-lang/KnoX/internal/analytics"
	"github.com/chenjianyu070921-lang/KnoX/internal/model"
	"github.com/chenjianyu070921-lang/KnoX/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newDocListSvcCtx(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}); err != nil {
		t.Fatalf("migrate documents failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	return &svc.ServiceContext{
		DocRepo:   repository.NewDocumentRepository(db),
		Analytics: analytics.New(nil),
	}
}

func TestDocListLogic_PaginationAndKeyword(t *testing.T) {
	svcCtx := newDocListSvcCtx(t)
	ctx := context.Background()

	for i, title := range []string{"alpha note", "beta guide", "alpha v2"} {
		doc := &model.Document{DocID: "doc_" + string(rune('1'+i)), Title: title, DocType: ".md"}
		if err := svcCtx.DocRepo.Create(ctx, doc); err != nil {
			t.Fatalf("seed doc failed: %v", err)
		}
	}

	l := NewDocListLogic(ctx, svcCtx)

	page1, err := l.DocList(&types.DocListReq{Page: 1, Size: 2})
	if err != nil {
		t.Fatalf("doc list failed: %v", err)
	}
	if page1.Total != 3 || len(page1.Items) != 2 || page1.Page != 1 {
		t.Fatalf("unexpected page1: total=%d len=%d page=%d", page1.Total, len(page1.Items), page1.Page)
	}

	filtered, err := l.DocList(&types.DocListReq{Page: 1, Size: 10, Keyword: "alpha"})
	if err != nil {
		t.Fatalf("filtered doc list failed: %v", err)
	}
	if filtered.Total != 2 || len(filtered.Items) != 2 {
		t.Fatalf("expected 2 filtered docs, got total=%d len=%d", filtered.Total, len(filtered.Items))
	}
}
