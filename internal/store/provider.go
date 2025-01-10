package store

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrStoreNotFound                   = errors.New("store not found")
	ErrLatestPublishedDocumentNotFound = errors.New("latest published document not found")
)

type DocumentStoreProvider interface {
	Provide(projectID uuid.UUID) (Store, error)
}

type ProjectStoreProvider struct {
	stores map[string]Store
}

func NewProjectStoreProvider() *ProjectStoreProvider {
	return &ProjectStoreProvider{
		stores: make(map[string]Store),
	}
}

func (p *ProjectStoreProvider) Provide(projectID uuid.UUID) (Store, error) {
	if store, ok := p.stores[projectID.String()]; ok {
		return store, nil
	}

	return nil, ErrStoreNotFound
}

type DefaultProvider struct {
	store Store
}

func (p *DefaultProvider) Provide(projectID uuid.UUID) (Store, error) {
	return p.store, nil
}
