package model

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Document{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&ProjectDocument{}); err != nil {
		return err
	}

	return nil
}
