package store

import (
	"context"

	"github.com/emrgen/document/internal/model"
)

type Store interface {
	DocumentStore
	DocumentBackupStore
	Transaction(ctx context.Context, f func(ctx context.Context) error) error
	Migrate() error
}

type DocumentStore interface {
	// CreateDocument creates a new document.
	CreateDocument(ctx context.Context, doc *model.Document) error
	// GetDocument retrieves a document by ID.
	GetDocument(ctx context.Context, id string) (*model.Document, error)
	// ListDocuments retrieves a list of documents by project ID.
	ListDocuments(ctx context.Context, projectID string) ([]*model.Document, error)
	// UpdateDocument updates a document.
	UpdateDocument(ctx context.Context, doc *model.Document) error
	// DeleteDocument deletes a document by ID.
	DeleteDocument(ctx context.Context, id string) error
}

type DocumentBackupStore interface {
	// CreateDocumentBackup creates a new document backup.
	CreateDocumentBackup(ctx context.Context, backup *model.DocumentBackup) error
	// ListDocumentBackups retrieves a list of document backups by document ID.
	ListDocumentBackups(ctx context.Context, docID string) ([]*model.DocumentBackup, error)
	// GetDocumentBackup retrieves a document backup by document ID and version.
	GetDocumentBackup(ctx context.Context, docID string, version int) (*model.DocumentBackup, error)
	// DeleteDocumentBackup deletes a document backup by document ID and version.
	DeleteDocumentBackup(ctx context.Context, docID string, version int) error
	// RestoreDocument restores a document from a backup.
	RestoreDocument(ctx context.Context, doc *model.Document, backup *model.DocumentBackup) error
}
