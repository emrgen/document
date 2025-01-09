package store

import (
	"context"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
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

func (g *GormStore) ListDocumentBackups(ctx context.Context, docID uuid.UUID) ([]*model.DocumentBackup, error) {
	var backups []*model.DocumentBackup
	err := g.db.Where("document_id = ?", docID).Find(&backups).Error
	return backups, err
}

func (g *GormStore) GetDocumentBackup(ctx context.Context, docID uuid.UUID, version uint64) (*model.DocumentBackup, error) {
	var backup model.DocumentBackup
	err := g.db.Where("id = ? AND version = ?", docID, version).First(&backup).Error
	return &backup, err
}

func (g *GormStore) DeleteDocumentBackup(ctx context.Context, docID uuid.UUID, version uint64) error {
	return g.db.Where("document_id = ? AND version = ?", docID, version).Delete(&model.DocumentBackup{}).Error
}

// RestoreDocument restores a document from a backup, before restoring the document we need to create a backup of the current document
func (g *GormStore) RestoreDocument(ctx context.Context, doc *model.Document, backup *model.DocumentBackup) error {
	//TODO implement me
	panic("implement me")
}

func (g *GormStore) CreateDocument(ctx context.Context, doc *model.Document) error {
	return g.db.Create(doc).Error
}

func (g *GormStore) GetDocument(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	var doc model.Document
	err := g.db.Where("id = ?", id).First(&doc).Error
	return &doc, err
}

func (g *GormStore) ListDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.Document, error) {
	var docs []*model.Document
	err := g.db.Where("project_id = ?", projectID).Find(&docs).Error
	return docs, err
}

func (g *GormStore) UpdateDocument(ctx context.Context, doc *model.Document) error {
	return g.db.Save(doc).Error
}

func (g *GormStore) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	return g.db.Where("id = ?", id).Delete(&model.Document{}).Error
}

func (g *GormStore) Migrate() error {
	return model.Migrate(g.db)
}

func (g *GormStore) Transaction(ctx context.Context, f func(tx Store) error) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		return f(&GormStore{db: tx})
	})
}
