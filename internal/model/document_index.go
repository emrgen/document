package model

import "gorm.io/gorm"

type DocumentIndex struct {
	gorm.Model
	DocumentID string `gorm:"primaryKey;uuid;not null;"`
	Version    string
	Content    string
}

func (DocumentIndex) TableName() string {
	return "document_index"
}
