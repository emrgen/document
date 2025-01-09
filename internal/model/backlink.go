package model

// Backlink represents a back link between two documents.
// This is used to track the relationships between documents.
// Dynamic relationships are created between documents when a link is created.
type Backlink struct {
	SourceID string `gorm:"primaryKey;uuid;not null;index:idx_back_links_source_id"`
	TargetID string `gorm:"primaryKey;uuid;not null;index:idx_back_links_target_id"`
}

func (b *Backlink) TableName() string {
	return "back_links"
}
