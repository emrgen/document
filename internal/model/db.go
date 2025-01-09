package model

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Document{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&DocumentBackup{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&PublishedDocument{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&PublishedDocumentMeta{}); err != nil {
		return err
	}

	return nil
}
