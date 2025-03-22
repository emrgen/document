package model

const (
	CurrentDocumentVersion = "current"
)

type PublishedLink struct {
	SourceID      string `gorm:"primaryKey;uuid;not null;index:idx_published_backlinks_source_id_version"`
	SourceVersion string `gorm:"primaryKey;not null;default:current;index:idx_published_backlinks_source_id_version"`
	TargetID      string `gorm:"primaryKey;uuid;not null;index:idx_published_backlinks_target_id_version"`
	TargetVersion string `gorm:"primaryKey;not null;default:current;index:idx_published_backlinks_target_id_version"`
}

func (b *PublishedLink) TableName() string {
	return "published_links"
}
