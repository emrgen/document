package service

import (
	"context"
	"errors"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/cache"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
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
	redis    *cache.Redis
	v1.UnimplementedDocumentServiceServer
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(db *gorm.DB, redis *cache.Redis) *DocumentService {
	service := &DocumentService{
		db:       db,
		redis:    redis,
		compress: compress.NewGZip(),
	}

	return service
}

// CreateDocument creates a new document.
func (d DocumentService) CreateDocument(ctx context.Context, request *v1.CreateDocumentRequest) (*v1.CreateDocumentResponse, error) {
	// TODO: check the user has access to the project
	doc := &model.Document{
		ProjectID: request.GetProjectId(),
	}
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
	// Get document from database
	doc, err := model.GetDocument(d.db, request.Id)
	if err != nil {
		return nil, err
	}

	data, err := d.compress.Decode([]byte(doc.Content))
	if err != nil {
		return nil, err
	}

	return &v1.GetDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Title:     doc.Name,
			Content:   string(data),
			Parts:     doc.Parts,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		},
	}, nil
}

// ListDocuments lists documents.
func (d DocumentService) ListDocuments(ctx context.Context, request *v1.ListDocumentsRequest) (*v1.ListDocumentsResponse, error) {
	var err error
	_, err = uuid.Parse(request.ProjectId)
	if err != nil {
		return nil, err
	}

	// Get documents from database
	var documents []*model.Document
	err = d.db.Where("project_id = ?", request.ProjectId).Find(&documents).Error
	if err != nil {
		return nil, err
	}
	var total int64
	err = d.db.Model(&model.Document{}).Where("project_id = ?", request.ProjectId).Count(&total).Error

	var documentsProto []*v1.Document
	for _, doc := range documents {
		documentsProto = append(documentsProto, &v1.Document{
			Id:        doc.ID,
			Title:     doc.Name,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		})
	}

	return &v1.ListDocumentsResponse{
		Documents: documentsProto,
		Total:     int32(total),
	}, nil
}

// UpdateDocument updates a document.
func (d DocumentService) UpdateDocument(ctx context.Context, request *v1.UpdateDocumentRequest) (*v1.UpdateDocumentResponse, error) {
	err := d.db.Transaction(func(tx *gorm.DB) error {
		// Get document from database
		doc, err := model.GetDocument(d.db, request.Id)
		if err != nil {
			return err
		}

		if doc.Version+1 != request.GetVersion() {
			return errors.New("document version mismatch")
		}

		// Update document in database
		if request.Title != nil {
			doc.Name = request.GetTitle()
		}

		if request.Content != nil {
			data, err := d.compress.Encode([]byte(request.GetContent()))
			if err != nil {
				return err
			}
			doc.Content = string(data)

			if request.Parts != nil {
				doc.Parts = request.GetParts()
			}

			doc.Version = request.GetVersion()
		} else {
			doc.Parts = append(doc.Parts, request.GetParts()...)
			doc.Version = request.GetVersion()
		}

		err = model.UpdateDocument(d.db, request.Id, doc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.UpdateDocumentResponse{
		Id:      request.Id,
		Title:   request.GetTitle(),
		Version: uint32(request.GetVersion()),
	}, nil
}

// DeleteDocument deletes a document.
func (d DocumentService) DeleteDocument(ctx context.Context, request *v1.DeleteDocumentRequest) (*v1.DeleteDocumentResponse, error) {
	id := uuid.MustParse(request.GetId())
	err := model.DeleteDocument(d.db, id.String())
	if err != nil {
		return nil, err
	}

	return &v1.DeleteDocumentResponse{}, nil
}
