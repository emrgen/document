package model

import "gorm.io/gorm"

// DocumentBackup represents a backup of a document
// we can keep track of the changes made to a document by storing its backups
// the backups are automatically created when a document is updated
// the cold backups are moved to a different storage like S3(we can keep the backups for a longer period of time in S3)
type DocumentBackup struct {
	gorm.Model
	ID          string    `gorm:"primaryKey:uuid;"`
	Version     int64     `gorm:"primaryKey"`
	Document    *Document `gorm:"foreignKey:ID"`
	Title       string    `gorm:""`
	Content     string    `gorm:""`
	UpdatedBy   string    `gorm:"not null"`
	Compression string
}

func (DocumentBackup) TableName() string {
	return "document_backups"
}
