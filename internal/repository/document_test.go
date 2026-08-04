package repository

import (
	"context"
	"testing"

	"github.com/chenjianyu070921-lang/KnoX/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newDocumentRepo(t *testing.T) *DocumentRepository {
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
	return NewDocumentRepository(db)
}

func createDoc(t *testing.T, repo *DocumentRepository, doc *model.Document) {
	t.Helper()
	if err := repo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create doc failed: %v", err)
	}
}

func TestDocumentRepository_CreateAndGetByDocID(t *testing.T) {
	repo := newDocumentRepo(t)
	ctx := context.Background()

	createDoc(t, repo, &model.Document{
		DocID:     "doc_1",
		Title:     "hello",
		Content:   "world",
		DocType:   ".md",
		FileUrl:   "http://example.com/hello.md",
		RequestID: strPtr("req-1"),
	})

	doc, err := repo.GetByDocID(ctx, "doc_1")
	if err != nil {
		t.Fatalf("get by doc id failed: %v", err)
	}
	if doc.Title != "hello" || doc.Version != 1 {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}

func TestDocumentRepository_GetByRequestID(t *testing.T) {
	repo := newDocumentRepo(t)
	ctx := context.Background()

	createDoc(t, repo, &model.Document{
		DocID:     "doc_1",
		Title:     "hello",
		DocType:   ".md",
		RequestID: strPtr("req-1"),
	})

	doc, err := repo.GetByRequestID(ctx, "req-1")
	if err != nil {
		t.Fatalf("get by request id failed: %v", err)
	}
	if doc == nil || doc.DocID != "doc_1" {
		t.Fatalf("unexpected doc: %+v", doc)
	}

	missing, err := repo.GetByRequestID(ctx, "req-missing")
	if err != nil {
		t.Fatalf("get missing request id failed: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil doc, got %+v", missing)
	}
}

func TestDocumentRepository_ListWithKeyword(t *testing.T) {
	repo := newDocumentRepo(t)
	ctx := context.Background()

	createDoc(t, repo, &model.Document{DocID: "doc_1", Title: "alpha note", DocType: ".md"})
	createDoc(t, repo, &model.Document{DocID: "doc_2", Title: "beta guide", DocType: ".md"})
	createDoc(t, repo, &model.Document{DocID: "doc_3", Title: "alpha v2", DocType: ".md"})

	docs, total, err := repo.List(ctx, 1, 10, "alpha")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 2 || len(docs) != 2 {
		t.Fatalf("expected 2 docs, got total=%d len=%d", total, len(docs))
	}

	docs, total, err = repo.List(ctx, 1, 10, "")
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
	if total != 3 || len(docs) != 3 {
		t.Fatalf("expected 3 docs, got total=%d len=%d", total, len(docs))
	}
}

func TestDocumentRepository_UpdateVersionOptimisticLock(t *testing.T) {
	repo := newDocumentRepo(t)
	ctx := context.Background()

	createDoc(t, repo, &model.Document{DocID: "doc_1", Title: "old", DocType: ".md"})

	if err := repo.UpdateVersion(ctx, "doc_1", 1, map[string]interface{}{"title": "new"}); err != nil {
		t.Fatalf("update version 1 failed: %v", err)
	}
	if err := repo.UpdateVersion(ctx, "doc_1", 1, map[string]interface{}{"title": "stale"}); err != gorm.ErrRecordNotFound {
		t.Fatalf("expected stale update to fail, got %v", err)
	}

	doc, err := repo.GetByDocID(ctx, "doc_1")
	if err != nil {
		t.Fatalf("get doc failed: %v", err)
	}
	if doc.Version != 2 || doc.Title != "new" {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}

func strPtr(s string) *string {
	return &s
}
