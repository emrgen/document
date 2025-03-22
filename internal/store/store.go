package store

import (
	"context"
	"errors"
	goset "github.com/deckarep/golang-set/v2"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
	"time"
)

var (
	// ErrDocumentNotFound is returned when a document is not found.
	ErrDocumentNotFound = errors.New("document not found")
	// ErrDocumentBackupNotFound is returned when a document backup is not found.
	ErrDocumentBackupNotFound = errors.New("document backup not found")
	// ErrPublishedDocumentNotFound is returned when a published document is not found.
	ErrPublishedDocumentNotFound = errors.New("published document not found")
	// ErrPublishedDocumentVersionNotFound is returned when a published document version is not found.
	ErrPublishedDocumentVersionNotFound = errors.New("published document version not found")
	// ErrPublishedDocumentMetaNotFound is returned when a published document meta is not found.
	ErrPublishedDocumentMetaNotFound = errors.New("published document meta not found")
	// ErrPublishedDocumentVersionExists is returned when a published document version already exists.
	ErrPublishedDocumentVersionExists = errors.New("published document version already exists")
	// ErrPublishedDocumentMetaExists is returned when a published document meta already exists.
	ErrPublishedDocumentMetaExists = errors.New("published document meta already exists")
)

type Store interface {
	DocumentStore
	DocumentIndexStore
	DocumentBackupStore
	PublishedDocumentStore
	Transaction(ctx context.Context, f func(tx Store) error) error
	Migrate() error
}

// DocumentIndexStore is the interface for document index store.
type DocumentIndexStore interface {
	// SaveDocumentTreeIndex saves a document index by ID and version.
	SaveDocumentTreeIndex(ctx context.Context, index *model.DocumentIndex) error
	// GetDocumentTreeIndex retrieves a document index by ID and version.
	GetDocumentTreeIndex(ctx context.Context, docID uuid.UUID, version string) (*model.DocumentIndex, error)
}

type DocumentStore interface {
	// ExistsDocuments checks if a document exists by ID.
	ExistsDocuments(ctx context.Context, docs []*model.Document) (bool, error)
	// CreateDocument creates a new document.
	CreateDocument(ctx context.Context, doc *model.Document) error
	// GetDocument retrieves a document by ID.
	GetDocument(ctx context.Context, id uuid.UUID) (*model.Document, error)
	// ListDocuments retrieves a list of documents by project ID.
	ListDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.Document, int64, error)
	// ListDocumentsFromIDs retrieves a list of documents by IDs.
	ListDocumentsFromIDs(ctx context.Context, ids []uuid.UUID) ([]*model.Document, error)
	// UpdateDocument updates a document.
	UpdateDocument(ctx context.Context, doc *model.Document) error
	// DeleteDocument deletes a document by ID.
	DeleteDocument(ctx context.Context, id uuid.UUID) error
	// EraseDocument erases a document by ID.
	EraseDocument(ctx context.Context, id uuid.UUID) error
	// PublishDocument creates a new published document.
	PublishDocument(ctx context.Context, doc *model.PublishedDocument) error
	// CreateBacklinks creates a new backlink.
	CreateBacklinks(ctx context.Context, links []*model.Link) error
	// DeleteBacklinks deletes backlinks by source ID.
	DeleteBacklinks(ctx context.Context, links []*model.Link) error
	//	ListBacklinks retrieves a list of backlinks by target ID.
	ListBacklinks(ctx context.Context, targetID uuid.UUID) ([]*model.Link, error)
	// ListDocumentProjectIDs retrieves a list of project IDs by document ID.
	ListDocumentProjectIDs(ctx context.Context, docIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
}

type DocumentBackupStore interface {
	// CreateDocumentBackup creates a new document backup.
	CreateDocumentBackup(ctx context.Context, backup *model.DocumentBackup) error
	// ListDocumentBackups retrieves a list of document backups by document ID.
	ListDocumentBackups(ctx context.Context, docID uuid.UUID) ([]*model.DocumentBackup, error)
	// ListDocumentBackupVersions retrieves a list of document versions by ID.
	ListDocumentBackupVersions(ctx context.Context, id uuid.UUID) ([]*model.DocumentBackup, error)
	// GetDocumentBackup retrieves a document backup by document ID and version.
	GetDocumentBackup(ctx context.Context, docID uuid.UUID, version int64) (*model.DocumentBackup, error)
	// DeleteDocumentBackup deletes a document backup by document ID and version.
	DeleteDocumentBackup(ctx context.Context, docID uuid.UUID, version int64) error
	// RestoreDocument restores a document from a backup.
	RestoreDocument(ctx context.Context, doc *model.Document, backup *model.DocumentBackup) error
	// GetDocumentByUpdatedTime retrieves a list of documents by updated time.
	GetDocumentByUpdatedTime(start time.Time, end time.Time) ([]*model.DocumentBackup, error)
	// DeleteDocumentBackups deletes document backups by document ID and versions.
	DeleteDocumentBackups(ctx context.Context, backups map[string]goset.Set[int64]) error
}

type PublishedDocumentStore interface {
	// ExistsPublishedDocuments checks if a published document exists by ID.
	ExistsPublishedDocuments(ctx context.Context, docs []*model.PublishedDocument) (bool, error)
	// GetPublishedDocumentByVersion retrieves a published document by ID.
	GetPublishedDocumentByVersion(ctx context.Context, id uuid.UUID, version string) (*model.PublishedDocument, error)
	// ListLatestPublishedDocuments retrieves a list of published documents by project ID.
	ListLatestPublishedDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.LatestPublishedDocumentMeta, error)
	// ListPublishedDocumentsByIdVersion retrieves a list of published documents by id@version list.
	ListPublishedDocumentsByIdVersion(ctx context.Context, projectID uuid.UUID, idVersions []*model.IDVersion) ([]*model.PublishedDocument, error)
	// UnpublishDocument unpublishes a document.
	UnpublishDocument(ctx context.Context, id uuid.UUID, version string) error
	// GetLatestPublishedDocument retrieves the latest published document by ID.
	GetLatestPublishedDocument(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocument, error)
	// ListPublishedDocumentVersions retrieves a list of published document versions by ID.
	ListPublishedDocumentVersions(ctx context.Context, id uuid.UUID) ([]*model.PublishedDocumentMeta, error)
	// GetLatestPublishedDocumentMeta retrieves the latest published document meta by ID.
	GetLatestPublishedDocumentMeta(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocumentMeta, error)
	// GetPublishedDocumentMetaByVersion retrieves a list of published links by source ID.
	GetPublishedDocumentMetaByVersion(ctx context.Context, id uuid.UUID, version string) (*model.PublishedDocumentMeta, error)
	// CreatePublishedLinks creates a new published document.
	CreatePublishedLinks(ctx context.Context, links []*model.PublishedLink) error
	// ListPublishedBacklinks retrieves a list of backlinks by source ID.
	ListPublishedBacklinks(ctx context.Context, targetID uuid.UUID, targetVersion string) ([]*model.PublishedLink, error)
	// ListPublishedDocumentProjectIDs retrieves a list of project IDs by document ID.
	ListPublishedDocumentProjectIDs(ctx context.Context, docs []*model.IDVersion) (map[uuid.UUID]uuid.UUID, error)
}
