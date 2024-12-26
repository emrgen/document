package store

import (
	"context"
	"github.com/emrgen/document/internal/model"
	"gorm.io/gorm"
)

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{
		db: db,
	}
}

var _ Store = (*GormStore)(nil)

type GormStore struct {
	db *gorm.DB
}

func (g *GormStore) CreateDocumentBackup(ctx context.Context, backup *model.DocumentBackup) error {
	return g.db.Create(backup).Error
}

func (g *GormStore) ListDocumentBackups(ctx context.Context, docID string) ([]*model.DocumentBackup, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) GetDocumentBackup(ctx context.Context, docID string, version int) (*model.DocumentBackup, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) DeleteDocumentBackup(ctx context.Context, docID string, version int) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) RestoreDocument(ctx context.Context, doc *model.Document, backup *model.DocumentBackup) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) Migrate() error {
	return model.Migrate(g.db)
}

func (g *GormStore) Transaction(ctx context.Context, f func(ctx context.Context) error) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) CreateDocument(ctx context.Context, doc *model.Document) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) GetDocument(ctx context.Context, id string) (*model.Document, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) ListDocuments(ctx context.Context, projectID string) ([]*model.Document, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) UpdateDocument(ctx context.Context, doc *model.Document) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) DeleteDocument(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}
