package model

import (
	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	ID          string `gorm:"primaryKey;uuid;not null;"`
	Version     int64
	ProjectID   string `gorm:"uuid;not null"`
	Meta        string `gorm:"not null"`
	Content     string `gorm:"not null"`
	Kind        string // markdown, html, json, etc.
	Compression string // the compression algorithm used to compress the document content
}
