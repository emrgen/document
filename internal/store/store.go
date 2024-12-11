package store

import (
	"context"

	"github.com/emrgen/document/internal/model"
)

type Store interface {
	DocumentStore
	Transaction(ctx context.Context, f func(ctx context.Context) error) error
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
