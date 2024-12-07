package model

import "gorm.io/gorm"

type ProjectDocument struct {
	gorm.Model
	ProjectID  string    `gorm:"primaryKey;uuid;not null;index:project_id_index"`
	DocumentID string    `gorm:"primaryKey;uuid;not null;index:document_id_index"`
	Document   *Document `gorm:"foreignKey:DocumentID;references:ID"`
}

func CreateProjectDocument(db *gorm.DB, projectDocument *ProjectDocument) error {
	return db.Create(projectDocument).Error
}
