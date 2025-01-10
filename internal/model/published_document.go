package model

import "gorm.io/gorm"

// PublishedDocument represents a published document
type PublishedDocument struct {
	gorm.Model
	ID          string `gorm:"uuid;primaryKey"`
	ProjectID   string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	Meta        string
	Content     string
	Links       string
	Unpublished bool `gorm:"default:false"`
}

// PublishedDocumentMeta represents the metadata of a published document
type PublishedDocumentMeta struct {
	gorm.Model
	ID          string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	ProjectID   string `gorm:"uuid;primaryKey"`
	Meta        string
	Links       string
	Unpublished bool `gorm:"default:false"`
}

type LatestPublishedDocument struct {
	gorm.Model
	ID        string `gorm:"uuid;primaryKey"`
	ProjectID string `gorm:"uuid;primaryKey"`
	Version   string
	Meta      string
	Links     string
	Content   string
}

func (l *LatestPublishedDocument) IntoPublishedDocument() *PublishedDocument {
	return &PublishedDocument{
		ID:        l.ID,
		ProjectID: l.ProjectID,
		Version:   l.Version,
		Meta:      l.Meta,
		Links:     l.Links,
		Content:   l.Content,
	}
}

type LatestPublishedDocumentMeta struct {
	gorm.Model
	ID        string `gorm:"uuid;primaryKey"`
	ProjectID string `gorm:"uuid;primaryKey"`
	Version   string
	Meta      string
	Links     string
}
