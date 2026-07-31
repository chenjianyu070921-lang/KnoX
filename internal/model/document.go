package model

import "gorm.io/gorm"

type Document struct {
	gorm.Model
	DocID     string  `gorm:"uniqueIndex;size:64;not null" json:"docId"` // 对外暴露的文档 ID
	Title     string  `gorm:"size:256;not null" json:"title"`
	Content   string  `gorm:"type:longtext" json:"content"`
	DocType   string  `gorm:"size:16;not null" json:"docType"` // pdf / md / txt
	FileUrl   string  `gorm:"size:128" json:"fileUrl"`
	RequestID *string `gorm:"uniqueIndex;size:32" json:"requestID"` // 唯一索引，允许 NULL（存量数据兼容）
	Version   int     `gorm:"default:1;not null" json:"version"` // 乐观锁版本号
}
