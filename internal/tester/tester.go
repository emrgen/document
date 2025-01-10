package tester

import (
	"github.com/emrgen/document/internal/cache"
	"os"

	"github.com/emrgen/document/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testPath = "../../.test/"
)

var (
	db *gorm.DB
)

func Setup() {
	RemoveDBFile()

	_ = os.Setenv("ENV", "test")

	err := os.MkdirAll(testPath+"/db", os.ModePerm)
	if err != nil {
		panic(err)
	}

	db, err = gorm.Open(sqlite.Open(testPath+"db/document.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = model.Migrate(db)
	if err != nil {
		panic(err)
	}
}

func TestDB() *gorm.DB {
	return db
}

func RemoveDBFile() {
	err := os.RemoveAll(testPath)
	if err != nil {
		panic(err)
	}
}

func Redis() *cache.Redis {
	r, err := cache.NewRedis()
	if err != nil {
		panic(err)
	}

	return r
}
