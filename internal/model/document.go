package model

import (
	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	ID            string `gorm:"primaryKey;uuid;not null;"`
	Version       int64
	ProjectID     string  `gorm:"uuid;not null"`
	Meta          string  `gorm:"not null;default:{}"`
	Content       string  `gorm:"not null"`
	Parts         string  `gorm:"not null;default:[]"`
	Children      string  `gorm:"not null;default:[]"`
	Links         string  `gorm:"not null;default:{}"`
	Backlinks     []*Link `gorm:"foreignKey:TargetID;references:ID"`
	BacklinkCount int     // update trigger
	Kind          string  // markdown, html, json, etc.
	Compression   string  // the compression algorithm used to compress the document content
}

func (d *Document) TableName() string {
	return "documents"
}
