package service

import (
	"context"
	"encoding/json"
	"time"

	v1 "github.com/emrgen/tinydoc/apis/v1"
	"github.com/emrgen/tinydoc/internal/cache"
	"github.com/emrgen/tinydoc/internal/compress"
	"github.com/emrgen/tinydoc/internal/model"
	"github.com/emrgen/tinydoc/internal/queue"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var (
	_ v1.DocumentServiceServer = (*DocumentService)(nil)
)

// DocumentService is a service for managing documents.
type DocumentService struct {
	db       *gorm.DB
	compress compress.Compress
	cache    cache.DocumentCache
	queue    queue.DocumentQueue
	v1.UnimplementedDocumentServiceServer
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(db *gorm.DB, cache cache.DocumentCache) *DocumentService {
	service := &DocumentService{
		db:       db,
		compress: compress.NewGZip(),
		cache:    cache,
	}

	return service
}

// CreateDocument creates a new document.
func (d DocumentService) CreateDocument(ctx context.Context, request *v1.CreateDocumentRequest) (*v1.CreateDocumentResponse, error) {
	doc := &model.Document{}
	var err error

	if request.Id != nil {
		doc.ID = request.GetId()
	} else {
		doc.ID = uuid.New().String()
	}

	doc.Name = request.GetTitle()
	data, err := d.compress.Encode([]byte(request.GetContent()))
	if err != nil {
		return nil, err
	}

	doc.Content = string(data)

	projectDoc := &model.ProjectDocument{
		DocumentID: doc.ID,
		ProjectID:  request.ProjectId,
	}

	err = d.db.Transaction(func(tx *gorm.DB) error {
		err := model.CreateDocument(d.db, doc)
		if err != nil {
			return err
		}

		err = model.CreateProjectDocument(d.db, projectDoc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = d.cache.UpdateDocument(ctx, uuid.MustParse(doc.ID), doc)
	if err != nil {
		return nil, err
	}

	return &v1.CreateDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Title:     doc.Name,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		},
	}, nil
}

// GetDocument retrieves a document.
func (d DocumentService) GetDocument(ctx context.Context, request *v1.GetDocumentRequest) (*v1.GetDocumentResponse, error) {
	if d.cache != nil {
		id, err := uuid.Parse(request.Id)
		if err != nil {
			return nil, err
		}

		// Check cache first before hitting the database
		doc, err := d.cache.GetDocument(ctx, id, cache.GetDocumentModeView)
		if err != nil {
			logrus.Errorf("Error getting document from cache: %v", err)
		}

		if doc != nil {
			logrus.Info("Document found in cache service")
			data, err := json.Marshal(doc.Parts)
			if err != nil {
				return nil, err
			}

			return &v1.GetDocumentResponse{
				Document: &v1.Document{
					Id:        doc.ID,
					Title:     doc.Name,
					Content:   doc.Content,
					Data:      string(data),
					Kind:      &doc.Kind,
					CreatedAt: timestamppb.New(doc.CreatedAt),
					UpdatedAt: timestamppb.New(doc.UpdatedAt),
				},
			}, nil
		}
	}

	logrus.Infof("Document not found in cache: %s", request.Id)
	// Get document from database
	doc, err := model.GetDocument(d.db, request.Id)
	if err != nil {
		return nil, err
	}

	data, err := d.compress.Decode([]byte(doc.Content))
	if err != nil {
		return nil, err
	}

	if d.cache != nil {
		id, err := uuid.Parse(doc.ID)
		if err != nil {
			return nil, err
		}

		// if user has edit permission, then load in to the update cache
		err = d.cache.UpdateDocument(ctx, id, doc)
		if err != nil {
			return nil, err
		}

		// if the user has only view permission, then load in to the read cache
		// err = d.cache.SetDocument(ctx, id, doc)
		// if err != nil {
		// 	logrus.Errorf("Error setting document in cache: %v", err)
		// }
	}

	return &v1.GetDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Title:     doc.Name,
			Content:   string(data),
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		},
	}, nil
}

// ListDocuments lists documents.
func (d DocumentService) ListDocuments(ctx context.Context, request *v1.ListDocumentsRequest) (*v1.ListDocumentsResponse, error) {
	//TODO implement me
	panic("implement me")
}

// UpdateDocument updates a document.
func (d DocumentService) UpdateDocument(ctx context.Context, request *v1.UpdateDocumentRequest) (*v1.UpdateDocumentResponse, error) {
	document := &model.Document{
		ID:      request.Id,
		Name:    request.GetTitle(),
		Content: request.GetContent(),
		Parts:   request.GetParts(),
		Version: request.GetVersion(),
	}

	response := &v1.UpdateDocumentResponse{
		Document: &v1.Document{
			Id:        request.Id,
			CreatedAt: timestamppb.New(document.CreatedAt),
			UpdatedAt: timestamppb.New(document.UpdatedAt),
		},
	}

	// current time
	document.UpdatedAt = time.Now().UTC()

	// Check if cache is enabled
	if d.cache != nil {
		// put the updates in a queue to be processed later
		err := d.queue.PublishChange(ctx, document)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	// Update document in database
	err := model.UpdateDocument(d.db, request.Id, document)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DeleteDocument deletes a document.
func (d DocumentService) DeleteDocument(ctx context.Context, request *v1.DeleteDocumentRequest) (*v1.DeleteDocumentResponse, error) {
	//TODO implement me
	panic("implement me")
}
