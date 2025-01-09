package model

// Backlink represents a back link between two documents.
// This is used to track the relationships between documents.
// Dynamic relationships are created between documents when a link is created.
type Backlink struct {
	ProjectID     string `gorm:"primaryKey;uuid;not null"`
	SourceID      string `gorm:"primaryKey;uuid;not null;index:idx_back_links_source_id_version"`
	SourceVersion int64  `gorm:"primaryKey;not null;index:idx_back_links_source_id_version"`
	TargetID      string `gorm:"primaryKey;uuid;not null;index:idx_back_links_target_id_version"`
	TargetVersion int64  `gorm:"primaryKey;not null;index:idx_back_links_target_id_version"`
}

func (b *Backlink) TableName() string {
	return "back_links"
}
