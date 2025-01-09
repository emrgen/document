package model

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	ID          string `gorm:"primaryKey;uuid;not null;"`
	Version     int64  `gorm:"primaryKey"`
	ProjectID   string `gorm:"uuid;not null"`
	Name        string
	Summary     string
	Excerpt     string
	Thumbnail   string
	Content     string   `gorm:"not null"`
	Parts       []string `gorm:"type:text[]"` // the parts to be merged with the content to get the final document
	Kind        string   // markdown, html, json, etc.
	Compression string   // the compression algorithm used to compress the document content
	Data        string   // the data of the document, lww
}

func CreateDocument(db *gorm.DB, document *Document) error {
	return db.Create(document).Error
}

func GetDocument(db *gorm.DB, id string) (*Document, error) {
	document := &Document{}
	err := db.Where("id = ?", id).First(document).Error
	if err != nil {
		logrus.Errorf("Error getting document: %v", err)
		return nil, err
	}

	return document, nil
}

func GetDocuments(db *gorm.DB, projectID string) ([]*Document, error) {
	documents := make([]*Document, 0)
	err := db.Where("project_id = ?", projectID).Find(&documents).Error
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func UpdateDocument(db *gorm.DB, id string, document *Document) error {
	return db.Model(&Document{}).Where("id = ?", id, document.Version).Updates(document).Error
}

func DeleteDocument(db *gorm.DB, id string) error {
	return db.Where("id = ?", id).Delete(&Document{}).Error
}

func (d *Document) UpdateChanges(db *gorm.DB) error {
	// if the document has content
	if d.Content != "" {
		return db.Model(&Document{}).Where("id = ? AND version < ?", d.ID, d.Version).Updates(d).Error
	}

	if len(d.Parts) > 0 {
		return db.Model(&Document{}).Where("id = ? AND version < ?", d.ID, d.Version).Updates(d).Error
	}

	return nil
}

func (d *Document) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}
