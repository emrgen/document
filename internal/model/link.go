package model

// Link represents a back link between two documents.
// This is used to track the relationships between documents.
// Dynamic relationships are created between documents when a link is created.
type Link struct {
	SourceID      string `gorm:"primaryKey;uuid;not null;index:idx_back_links_source_id"`
	TargetID      string `gorm:"primaryKey;uuid;not null;index:idx_back_links_target_id_version"`
	TargetVersion string `gorm:"primaryKey;not null;index:idx_back_links_target_id_version"`
	Pending       bool   `gorm:"not null;default:true"` // pending links are marked false when the target document backlink count is updated
}

func (b *Link) TableName() string {
	return "links"
}
