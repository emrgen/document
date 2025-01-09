package model

import "gorm.io/gorm"

// PublishedDocument represents a published document
type PublishedDocument struct {
	gorm.Model
	ID      string `gorm:"primaryKey"`
	Version string `gorm:"primaryKey"`
	Meta    string
	Content string
}

// PublishedDocumentMeta represents the metadata of a published document
type PublishedDocumentMeta struct {
	gorm.Model
	ID      string `gorm:"primaryKey"`
	Version string `gorm:"primaryKey"`
	Content string
}
