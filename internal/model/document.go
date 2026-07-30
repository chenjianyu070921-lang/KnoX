package model

import "gorm.io/gorm"

type Document struct {
	gorm.Model
	DocID   string `gorm:"uniqueIndex;size:64;not null" json:"docId"` // 对外暴露的文档 ID
	Title   string `gorm:"size:256;not null" json:"title"`
	Content string `gorm:"type:longtext" json:"content"`
	DocType string `gorm:"size:16;not null" json:"docType"`   // pdf / md / txt
	Version int    `gorm:"default:1;not null" json:"version"` // 乐观锁版本号
}
