package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourname/know/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create 创建文档
func (r *DocumentRepository) Create(ctx context.Context, doc *model.Document) error {
	doc.Version = 1
	return r.db.WithContext(ctx).Create(doc).Error
}

// GetByDocID 根据对外 ID 查询文档
func (r *DocumentRepository) GetByDocID(ctx context.Context, docID string) (*model.Document, error) {
	var doc model.Document
	err := r.db.WithContext(ctx).Where("doc_id = ?", docID).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpdateVersion 乐观锁更新（version + 1）
func (r *DocumentRepository) UpdateVersion(ctx context.Context, docID string, currentVersion int, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&model.Document{}).
		Where("doc_id = ? AND version = ?", docID, currentVersion).
		Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// List 分页查询
func (r *DocumentRepository) List(ctx context.Context, page, size int) ([]model.Document, int64, error) {
	var docs []model.Document
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Document{})
	db.Count(&total)

	offset := (page - 1) * size
	err := db.Offset(offset).Limit(size).Order("created_at desc").Find(&docs).Error
	return docs, total, err
}
func (r *DocumentRepository) GetByRequestID(ctx context.Context, requestId string) (*model.Document, error) {
	var doc model.Document
	err := r.db.WithContext(ctx).Where("request_id = ?", requestId).First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // 未命中不是错误，调用方判 nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// GenDocID 生成对外文档 ID
func GenDocID() string {
	return "doc_" + time.Now().Format("20060102150405.000") + "_" + uuid.NewString()[:8]
}
