package model

type PublishedBacklink struct {
	SourceID      string `gorm:"primaryKey;uuid;not null;index:idx_published_backlinks_source_id_version"`
	SourceVersion string `gorm:"primaryKey;not null;index:idx_published_backlinks_source_id_version"`
	TargetID      string `gorm:"primaryKey;uuid;not null;index:idx_published_backlinks_target_id_version"`
	TargetVersion string `gorm:"primaryKey;not null;index:idx_published_backlinks_target_id_version"`
}
