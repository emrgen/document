package cache

import (
	"context"

	"github.com/emrgen/tinydoc/internal/model"
	"github.com/google/uuid"
)

type GetDocumentMode int

const (
	GetDocumentModeView GetDocumentMode = iota
	GetDocumentModeEdit
)

// DocumentCache is a cache for documents.
type DocumentCache interface {
	// GetDocumentVersion gets the version of a document from the cache.
	GetDocumentVersion(ctx context.Context, id uuid.UUID, view GetDocumentMode) (int64, error)
	// GetDocument gets a document from the cache.
	GetDocument(ctx context.Context, id uuid.UUID, view GetDocumentMode) (*model.Document, error)
	// SetDocument sets a document in the cache.
	SetDocument(ctx context.Context, id uuid.UUID, doc *model.Document) error
	// UpdateDocument updates a document in the cache.
	UpdateDocument(ctx context.Context, id uuid.UUID, doc *model.Document) error
	// DeleteDocument deletes a document from the cache.
	DeleteDocument(ctx context.Context, id uuid.UUID) error
}
