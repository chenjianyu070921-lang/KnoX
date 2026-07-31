package repository

import (
	"context"
	"errors"

	"github.com/yourname/know/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConsumeRecordRepository struct {
	db *gorm.DB
}

func NewConsumeRecordRepository(db *gorm.DB) *ConsumeRecordRepository {
	return &ConsumeRecordRepository{db: db}
}

// CreateOrIgnore INSERT ... ON DUPLICATE KEY DO NOTHING
func (r *ConsumeRecordRepository) CreateOrIgnore(ctx context.Context, record *model.ConsumeRecord) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(record).Error
}

// UpdateStatus 更新状态；failed 时 retry_count+1
func (r *ConsumeRecordRepository) UpdateStatus(ctx context.Context, topic string, partition int32, offset int64, status string, message string) error {
	updates := map[string]interface{}{
		"status":  status,
		"message": message,
	}
	if status == model.ConsumeStatusFailed {
		updates["retry_count"] = gorm.Expr("retry_count + 1")
	}
	return r.db.WithContext(ctx).Model(&model.ConsumeRecord{}).
		Where("topic = ? AND partition = ? AND offset = ?", topic, partition, offset).
		Updates(updates).Error
}

// GetByKey 按联合唯一键查询
func (r *ConsumeRecordRepository) GetByKey(ctx context.Context, topic string, partition int32, offset int64) (*model.ConsumeRecord, error) {
	var record model.ConsumeRecord
	err := r.db.WithContext(ctx).
		Where("topic = ? AND partition = ? AND offset = ?", topic, partition, offset).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}
