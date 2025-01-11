package service

import "errors"

var (
	// ErrInvalidLinkFormat is returned when a document is not found.
	ErrInvalidLinkFormat = errors.New("invalid link format, expected format is <id>@<version>")
	// ErrInvalidChildrenLinkFormat is returned when a document is not found.
	ErrInvalidChildrenLinkFormat = errors.New("invalid children link format, expected format is <id>@<version>")
	// ErrDocumentMetaCorrupted is returned when a document is not found.
	ErrDocumentMetaCorrupted = errors.New("document meta is corrupted")
	// ErrDocumentChildrenCorrupted is returned when a document is not found.
	ErrDocumentChildrenCorrupted = errors.New("document children are corrupted")
	// ErrDocumentLinksCorrupted is returned when a document is not found.
	ErrDocumentLinksCorrupted = errors.New("document links are corrupted")
)
