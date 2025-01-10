package store

import (
	"context"
	"github.com/google/uuid"

	"github.com/emrgen/document/internal/model"
)

type Store interface {
	DocumentStore
	DocumentBackupStore
	PublishedDocumentStore
	Transaction(ctx context.Context, f func(tx Store) error) error
	Migrate() error
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
}

type PublishedDocumentStore interface {
	// ExistsPublishedDocuments checks if a published document exists by ID.
	ExistsPublishedDocuments(ctx context.Context, docs []*model.PublishedDocument) (bool, error)
	// GetPublishedDocumentByVersion retrieves a published document by ID.
	GetPublishedDocumentByVersion(ctx context.Context, id uuid.UUID, version string) (*model.PublishedDocument, error)
	// ListLatestPublishedDocuments retrieves a list of published documents by project ID.
	ListLatestPublishedDocuments(ctx context.Context, projectID uuid.UUID) ([]*model.LatestPublishedDocumentMeta, error)
	// UnpublishDocument unpublishes a document.
	UnpublishDocument(ctx context.Context, id uuid.UUID, version string) error
	// GetLatestPublishedDocument retrieves the latest published document by ID.
	GetLatestPublishedDocument(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocument, error)
	// ListPublishedDocumentVersions retrieves a list of published document versions by ID.
	ListPublishedDocumentVersions(ctx context.Context, id uuid.UUID) ([]*model.PublishedDocumentMeta, error)
	// GetLatestPublishedDocumentMeta retrieves the latest published document meta by ID.
	GetLatestPublishedDocumentMeta(ctx context.Context, id uuid.UUID) (*model.LatestPublishedDocumentMeta, error)
	// CreatePublishedLinks creates a new published document.
	CreatePublishedLinks(ctx context.Context, links []*model.PublishedLink) error
	// ListPublishedBacklinks retrieves a list of backlinks by source ID.
	ListPublishedBacklinks(ctx context.Context, targetID uuid.UUID, targetVersion string) ([]*model.PublishedLink, error)
}
