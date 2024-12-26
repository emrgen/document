package model

import "gorm.io/gorm"

// BackLink represents a back link between two documents.
// This is used to track the relationships between documents.
// Dynamic relationships are created between documents when a link is created.
type BackLink struct {
	gorm.Model
	ProjectID string `gorm:"uuid;not null"`
	SourceID  string `gorm:"uuid;not null"`
	TargetID  string `gorm:"uuid;not null"`
}

func (b *BackLink) TableName() string {
	return "back_links"
}
