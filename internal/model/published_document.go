package model

import "gorm.io/gorm"

// PublishedDocument represents a published document
type PublishedDocument struct {
	gorm.Model
	ID          string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	Meta        string
	Content     string
	Unpublished bool `gorm:"default:false"`
}

// PublishedDocumentMeta represents the metadata of a published document
type PublishedDocumentMeta struct {
	gorm.Model
	ID          string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	Content     string
	Unpublished bool `gorm:"default:false"`
}

type LatestPublishedDocument struct {
	gorm.Model
	ID      string `gorm:"uuid;primaryKey"`
	Version string
	Meta    string
	Content string
}

type LatestPublishedDocumentMeta struct {
	gorm.Model
	ID      string `gorm:"uuid;primaryKey"`
	Version string
	Content string
}
