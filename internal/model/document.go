package model

import (
	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	ID          string `gorm:"primaryKey;uuid;not null;"`
	Version     int64
	ProjectID   string `gorm:"uuid;not null"`
	Meta        string `gorm:"not null;default:{}"`
	Content     string `gorm:"not null"`
	Links       string `gorm:"not null;default:{}"`
	Kind        string // markdown, html, json, etc.
	Compression string // the compression algorithm used to compress the document content
}
