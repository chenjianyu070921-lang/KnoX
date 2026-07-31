package model

import "gorm.io/gorm"

// ConsumeRecord 消费状态常量
const (
	ConsumeStatusProcessing = "processing"
	ConsumeStatusDone       = "done"
	ConsumeStatusFailed     = "failed"
)

// ConsumeRecord Kafka 消费幂等记录，联合唯一键 (topic, partition, offset) 防止同一消息重复处理
type ConsumeRecord struct {
	gorm.Model
	Topic      string `gorm:"uniqueIndex:idx_tpo;size:128;not null"` // Kafka topic 名
	Partition  int32  `gorm:"uniqueIndex:idx_tpo;not null"`          // 分区号
	Offset     int64  `gorm:"uniqueIndex:idx_tpo;not null"`          // 消息偏移量
	DocID      string `gorm:"size:64;not null"`                     // 关联的文档 ID
	Status     string `gorm:"size:16;not null;default:processing"`  // processing → done / failed
	Message    string `gorm:"type:text"`                            // 失败时的错误信息
	RetryCount int    `gorm:"default:0"`                            // 失败重试次数
}
