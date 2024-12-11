package store

import (
	"context"
	"github.com/emrgen/document/internal/model"
	"gorm.io/gorm"
)

var _ Store = (*GormStore)(nil)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{
		db: db,
	}
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
