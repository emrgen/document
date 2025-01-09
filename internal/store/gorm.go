package store

import (
	"context"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

func (g *GormStore) ListDocumentsFromIDs(ctx context.Context, ids []uuid.UUID) ([]*model.Document, error) {
	var docs []*model.Document
	err := g.db.Where("id in (?)", ids).Find(&docs).Error
	return docs, err
}

func (g *GormStore) EraseDocument(ctx context.Context, id uuid.UUID) error {
	return g.db.WithContext(ctx).Unscoped().Delete(&model.Document{}, id.String()).Error
}

func (g *GormStore) ListPublishedDocumentVersions(ctx context.Context, id uuid.UUID) ([]*model.PublishedDocumentMeta, error) {
	var docs []*model.PublishedDocumentMeta
	err := g.db.Where("id = ?", id).Order("created_at desc").Find(&docs).Error
	if err != nil {
		return nil, err
	}

	return docs, err
}

func (g *GormStore) GetLatestPublishedDocumentMeta(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocumentMeta, error) {
	var doc model.LatestPublishedDocumentMeta
	err := g.db.Where("id = ?", id).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, err
}

func (g *GormStore) GetLatestPublishedDocument(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocument, error) {
	var doc model.LatestPublishedDocument
	err := g.db.Where("id = ?", id).Order("version desc").First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, err
}

// PublishDocument publishes a document, creating a new published document
// NOTE: should run in a transaction
func (g *GormStore) PublishDocument(ctx context.Context, doc *model.PublishedDocument) error {
	latestDocMeta := &model.LatestPublishedDocumentMeta{
		ID:      doc.ID,
		Version: doc.Version,
		Content: doc.Meta,
	}

	latestDoc := &model.LatestPublishedDocument{
		ID:      doc.ID,
		Version: doc.Version,
		Meta:    doc.Meta,
		Content: doc.Content,
	}

	docMeta := &model.PublishedDocumentMeta{
		ID:      doc.ID,
		Version: doc.Version,
		Content: doc.Meta,
	}

	logrus.Infof("Publishing document %s version %s", doc.ID, doc.Version)

	if err := g.db.Save(latestDocMeta).Error; err != nil {
		return err
	}

	if err := g.db.Save(latestDoc).Error; err != nil {
		return err
	}

	if err := g.db.Create(docMeta).Error; err != nil {
		return err
	}

	return g.db.Create(doc).Error
}

// GetPublishedDocumentByVersion creates a new project
func (g *GormStore) GetPublishedDocumentByVersion(ctx context.Context, id uuid.UUID, version string) (*model.PublishedDocument, error) {
	var doc model.PublishedDocument
	err := g.db.Where("id = ? AND version = ?", id, version).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, err
}

// ListLatestPublishedDocuments returns a list of published documents for a project
func (g *GormStore) ListLatestPublishedDocuments(ctx context.Context, docID uuid.UUID) ([]*model.LatestPublishedDocumentMeta, error) {
	var docs []*model.LatestPublishedDocumentMeta
	err := g.db.Where("id = ?", docID).Find(&docs).Error
	return docs, err
}

// UnpublishDocument removes a document from the published documents
func (g *GormStore) UnpublishDocument(ctx context.Context, id uuid.UUID, toVersion string) error {
	return g.db.Where("id = ?", id).Delete(&model.PublishedDocument{}).Error
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
