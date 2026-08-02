package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/yourname/know/internal/model"
	"gorm.io/gorm"
)

func newInMemoryRepo(t *testing.T) *ConsumeRecordRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.ConsumeRecord{}); err != nil {
		t.Fatalf("migrate consume_records failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewConsumeRecordRepository(db)
}

func insertConsumeRecord(t *testing.T, repo *ConsumeRecordRepository, rec *model.ConsumeRecord) {
	t.Helper()
	if err := repo.db.WithContext(context.Background()).Create(rec).Error; err != nil {
		t.Fatalf("insert consume_record failed: %v", err)
	}
}

func TestTakeOver(t *testing.T) {
	repo := newInMemoryRepo(t)
	ctx := context.Background()
	staleBefore := time.Now().Add(-time.Minute)

	t.Run("failed 未到上限可接管", func(t *testing.T) {
		insertConsumeRecord(t, repo, &model.ConsumeRecord{
			Topic: "doc-embedding", Partition: 0, Offset: 1, DocID: "doc-1",
			Status: model.ConsumeStatusFailed, RetryCount: 1,
		})

		claimed, err := repo.TakeOver(ctx, "doc-embedding", 0, 1, 3, staleBefore)
		if err != nil {
			t.Fatalf("TakeOver returned error: %v", err)
		}
		if !claimed {
			t.Fatal("failed 且 retry_count < max，应接管成功")
		}

		var rec model.ConsumeRecord
		if err := repo.db.WithContext(ctx).
			Where("topic = ? AND partition = ? AND offset = ?", "doc-embedding", 0, 1).
			First(&rec).Error; err != nil {
			t.Fatalf("query record failed: %v", err)
		}
		if rec.Status != model.ConsumeStatusProcessing {
			t.Fatalf("接管后 status = %s, want processing", rec.Status)
		}
	})

	t.Run("failed 已到上限不可接管", func(t *testing.T) {
		insertConsumeRecord(t, repo, &model.ConsumeRecord{
			Topic: "doc-embedding", Partition: 0, Offset: 2, DocID: "doc-2",
			Status: model.ConsumeStatusFailed, RetryCount: 3,
		})

		claimed, err := repo.TakeOver(ctx, "doc-embedding", 0, 2, 3, staleBefore)
		if err != nil {
			t.Fatalf("TakeOver returned error: %v", err)
		}
		if claimed {
			t.Fatal("failed 且 retry_count >= max，不应接管")
		}
	})

	t.Run("processing 已过期可接管", func(t *testing.T) {
		rec := &model.ConsumeRecord{
			Topic: "doc-embedding", Partition: 0, Offset: 3, DocID: "doc-3",
			Status: model.ConsumeStatusProcessing,
		}
		insertConsumeRecord(t, repo, rec)
		if err := repo.db.Model(&model.ConsumeRecord{}).
			Where("topic = ? AND partition = ? AND offset = ?", "doc-embedding", 0, 3).
			Update("updated_at", time.Now().Add(-2*time.Minute)).Error; err != nil {
			t.Fatalf("backdate record failed: %v", err)
		}

		claimed, err := repo.TakeOver(ctx, "doc-embedding", 0, 3, 3, staleBefore)
		if err != nil {
			t.Fatalf("TakeOver returned error: %v", err)
		}
		if !claimed {
			t.Fatal("processing 已过期，应接管成功")
		}
	})

	t.Run("processing 很新鲜不可接管", func(t *testing.T) {
		insertConsumeRecord(t, repo, &model.ConsumeRecord{
			Topic: "doc-embedding", Partition: 0, Offset: 4, DocID: "doc-4",
			Status: model.ConsumeStatusProcessing,
		})

		claimed, err := repo.TakeOver(ctx, "doc-embedding", 0, 4, 3, staleBefore)
		if err != nil {
			t.Fatalf("TakeOver returned error: %v", err)
		}
		if claimed {
			t.Fatal("processing 很新鲜，不应接管")
		}
	})
}
