package model

import "gorm.io/gorm"

type LatestPublishedDocument struct {
	gorm.Model
	ProjectID string `gorm:"uuid;primaryKey"`
	ID        string `gorm:"uuid;primaryKey"`
	Version   string
	Meta      string
	Links     string
	Children  string `gorm:"not null;default:[]"`
	Content   string
}

// IntoPublishedDocument converts LatestPublishedDocument to PublishedDocument
func (l *LatestPublishedDocument) IntoPublishedDocument() *PublishedDocument {
	return &PublishedDocument{
		ID:        l.ID,
		ProjectID: l.ProjectID,
		Version:   l.Version,
		Meta:      l.Meta,
		Content:   l.Content,
		Links:     l.Links,
		Children:  l.Children,
	}
}

// LatestPublishedDocumentMeta represents the metadata of a latest published document
type LatestPublishedDocumentMeta struct {
	gorm.Model
	ProjectID string `gorm:"uuid;primaryKey"`
	ID        string `gorm:"uuid;primaryKey"`
	Version   string
	Meta      string
	Links     string
	Children  string `gorm:"not null;default:[]"`
}

// IntoPublishedDocumentMeta converts LatestPublishedDocumentMeta to PublishedDocumentMeta
func (l *LatestPublishedDocumentMeta) IntoPublishedDocumentMeta() *PublishedDocumentMeta {
	return &PublishedDocumentMeta{
		ID:        l.ID,
		ProjectID: l.ProjectID,
		Version:   l.Version,
		Meta:      l.Meta,
		Links:     l.Links,
		Children:  l.Children,
	}
}

// PublishedDocument represents a published document
type PublishedDocument struct {
	gorm.Model
	ProjectID   string `gorm:"uuid;primaryKey"`
	ID          string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	Meta        string
	Content     string
	Links       string
	Children    string `gorm:"not null;default:[]"`
	Latest      bool   `gorm:"default:false"`
	Unpublished bool   `gorm:"default:false"`
}

// PublishedDocumentMeta represents the metadata of a published document
type PublishedDocumentMeta struct {
	gorm.Model
	ProjectID   string `gorm:"uuid;primaryKey"`
	ID          string `gorm:"uuid;primaryKey"`
	Version     string `gorm:"uuid;primaryKey"` // semantic versioning
	Meta        string
	Links       string
	Children    string `gorm:"not null;default:[]"`
	Latest      bool   `gorm:"default:false"`
	Unpublished bool   `gorm:"default:false"`
}

type IDVersion struct {
	ID      string
	Version string
}
