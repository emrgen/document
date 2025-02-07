package store

import (
	"context"
	"errors"
	goset "github.com/deckarep/golang-set/v2"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
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

func (g *GormStore) DeleteDocumentBackups(ctx context.Context, backups map[string]goset.Set[int64]) error {
	var groupErr error

	for id, versions := range backups {
		versionList := versions.ToSlice()
		err := g.db.Unscoped().Where("id = ? AND version IN (?)", id, versionList).Delete(&model.DocumentBackup{}).Error
		if err != nil {
			groupErr = errors.Join(groupErr, err)
			continue
		}
	}

	return groupErr
}

func (g *GormStore) GetDocumentByUpdatedTime(start time.Time, end time.Time) ([]*model.DocumentBackup, error) {
	var docs []*model.DocumentBackup
	err := g.db.Where("updated_at > ? AND updated_at < ?", start, end).Order("updated_at asc").Find(&docs).Error
	return docs, err
}

func (g *GormStore) GetPublishedDocumentMetaByVersion(ctx context.Context, id uuid.UUID, version string) (*model.PublishedDocumentMeta, error) {
	var doc model.PublishedDocumentMeta
	err := g.db.Where("id = ? AND version = ?", id.String(), version).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, err
}

func (g *GormStore) ListDocumentBackupVersions(ctx context.Context, id uuid.UUID) ([]*model.DocumentBackup, error) {
	var docs []*model.DocumentBackup
	err := g.db.Where("id = ?", id).Order("created_at desc").Find(&docs).Error
	return docs, err
}

func (g *GormStore) ExistsDocuments(ctx context.Context, docs []*model.Document) (bool, error) {
	return false, nil
}

func (g *GormStore) ExistsPublishedDocuments(ctx context.Context, docs []*model.PublishedDocument) (bool, error) {
	return false, nil
}

func (g *GormStore) CreatePublishedLinks(ctx context.Context, links []*model.PublishedLink) error {
	return g.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "target_id"}, {Name: "target_version"}, {Name: "source_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"source_id", "target_id", "target_version", "source_version"}),
	}).Create(links).Error
}

func (g *GormStore) CreateBacklinks(ctx context.Context, links []*model.Link) error {
	return g.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "target_id"}, {Name: "target_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"source_id", "target_id", "target_version"}),
	}).Create(links).Error
}

func (g *GormStore) DeleteBacklinks(ctx context.Context, links []*model.Link) error {
	for _, link := range links {
		if err := g.db.Delete(link).Error; err != nil {
			return err
		}
	}

	return nil
}

// ListBacklinks returns a list of backlinks for a document
func (g *GormStore) ListBacklinks(ctx context.Context, targetID uuid.UUID) ([]*model.Link, error) {
	var backlinks []*model.Link
	err := g.db.Where("target_id = ?", targetID).Find(&backlinks).Error
	return backlinks, err
}

// ListPublishedBacklinks returns a list of backlinks for a published document
func (g *GormStore) ListPublishedBacklinks(ctx context.Context, targetID uuid.UUID, targetVersion string) ([]*model.PublishedLink, error) {
	var backlinks []*model.PublishedLink
	err := g.db.Where("target_id = ? AND target_version = ?", targetID, targetVersion).Find(&backlinks).Error
	return backlinks, err
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

// GetLatestPublishedDocumentMeta retrieves the latest published document meta
func (g *GormStore) GetLatestPublishedDocumentMeta(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocumentMeta, error) {
	var doc model.LatestPublishedDocumentMeta
	err := g.db.Where("id = ?", id).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, err
}

// GetLatestPublishedDocument retrieves the latest published document
func (g *GormStore) GetLatestPublishedDocument(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocument, error) {
	var doc model.LatestPublishedDocument
	err := g.db.Where("id = ?", id.String()).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLatestPublishedDocumentNotFound
		}
		return nil, err
	}
	if &doc == nil {
		return nil, ErrLatestPublishedDocumentNotFound
	}

	return &doc, err
}

// PublishDocument publishes a document, creating a new published document
// NOTE: should run in a transaction
func (g *GormStore) PublishDocument(ctx context.Context, doc *model.PublishedDocument) error {
	latestDocMeta := &model.LatestPublishedDocumentMeta{
		ID:        doc.ID,
		ProjectID: doc.ProjectID,
		Version:   doc.Version,
		Meta:      doc.Meta,
		Links:     doc.Links,
		Children:  doc.Children,
	}

	latestDoc := &model.LatestPublishedDocument{
		ID:        doc.ID,
		ProjectID: doc.ProjectID,
		Version:   doc.Version,
		Meta:      doc.Meta,
		Links:     doc.Links,
		Children:  doc.Children,
		Content:   doc.Content,
	}

	docMeta := &model.PublishedDocumentMeta{
		ID:        doc.ID,
		ProjectID: doc.ProjectID,
		Version:   doc.Version,
		Meta:      doc.Meta,
		Links:     doc.Links,
		Children:  doc.Children,
		Latest:    true,
	}

	// make sure the latest is set to true
	doc.Latest = true

	// make the last published document not the latest
	if err := g.db.Model(&model.PublishedDocument{}).Where("id = ?", doc.ID).Order("created_at desc").Update("latest", false).Error; err != nil {
		return err
	}
	// make the last published document meta not the latest
	if err := g.db.Model(&model.PublishedDocumentMeta{}).Where("id = ?", doc.ID).Order("created_at desc").Update("latest", false).Error; err != nil {
		return err
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
func (g *GormStore) ListLatestPublishedDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.LatestPublishedDocumentMeta, error) {
	var docs []*model.LatestPublishedDocumentMeta
	err := g.db.Where("project_id = ?", projectID).Find(&docs).Error
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

func (g *GormStore) GetDocumentBackup(ctx context.Context, docID uuid.UUID, version int64) (*model.DocumentBackup, error) {
	var backup model.DocumentBackup
	err := g.db.Where("id = ? AND version = ?", docID, version).First(&backup).Error
	return &backup, err
}

func (g *GormStore) DeleteDocumentBackup(ctx context.Context, docID uuid.UUID, version int64) error {
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

// ListDocuments returns a list of documents for a project
func (g *GormStore) ListDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.Document, int64, error) {
	var docs []*model.Document
	// TODO: it should be paginated
	err := g.db.Where("project_id = ?", projectID).Find(&docs).Error
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = g.db.Model(&model.Document{}).Where("project_id = ?", projectID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	return docs, total, nil
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
