package service

import (
	"context"
	"errors"
	"github.com/emrgen/document/internal/store"
	gox "github.com/emrgen/gopack/x"

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

// NewDocumentService creates a new DocumentService.
func NewDocumentService(db *gorm.DB, store store.Store, redis *cache.Redis) *DocumentService {
	service := &DocumentService{
		db:       db,
		redis:    redis,
		store:    store,
		compress: compress.NewGZip(),
	}

	return service
}

// DocumentService is a service for managing documents.
type DocumentService struct {
	db       *gorm.DB
	compress compress.Compress
	redis    *cache.Redis
	store    store.Store
	v1.UnimplementedDocumentServiceServer
}

// CreateDocument creates a new document.
func (d DocumentService) CreateDocument(ctx context.Context, request *v1.CreateDocumentRequest) (*v1.CreateDocumentResponse, error) {
	var err error

	projectID := request.GetProjectId()
	doc := &model.Document{
		ProjectID: projectID,
	}

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
	err = model.CreateDocument(d.db, doc)
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
	// TODO: first look into cache
	// doc, err := d.redis.Get(ctx, fmt.Sprintf("document:%s/version:%s", request.Id, request.Version))
	// if err == nil {
	// 	return doc, nil
	// }

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
	projectID, err := gox.GetProjectID(ctx)
	if err != nil {
		return nil, err
	}

	// Get documents from database
	var documents []*model.Document
	err = d.db.Where("project_id = ?", projectID.String()).Find(&documents).Error
	if err != nil {
		return nil, err
	}
	var total int64
	err = d.db.Model(&model.Document{}).Where("project_id = ?", projectID.String()).Count(&total).Error

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
	var err error
	userID, err := gox.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	err = d.db.Transaction(func(tx *gorm.DB) error {
		// Get document from database
		doc, err := model.GetDocument(d.db, request.Id)
		if err != nil {
			return err
		}

		err = d.store.CreateDocumentBackup(ctx, &model.DocumentBackup{
			ID:        doc.ID,
			Version:   doc.Version + 1,
			Content:   doc.Content,
			UpdatedBy: userID.String(),
		})
		if err != nil {
			return err
		}

		overwrite := request.Version == -1

		if !overwrite && doc.Version+1 != uint64(request.GetVersion()) {
			return errors.New("document version mismatch")
		}

		// Update document in database
		if request.Title != nil {
			doc.Name = request.GetTitle()
		}

		// if the content is not nil, update the content
		// otherwise, append the parts to the document
		if overwrite || request.Content != nil {
			data, err := d.compress.Encode([]byte(request.GetContent()))
			if err != nil {
				return err
			}
			doc.Content = string(data)

			if request.Parts != nil {
				doc.Parts = request.GetParts()
			}

			doc.Version = doc.Version + 1
		} else {
			doc.Parts = append(doc.Parts, request.GetParts()...)
			// TODO: if the parts are too large, we need to merge them
			doc.Version = doc.Version + 1
		}

		err = model.UpdateDocument(d.db, request.Id, doc)
		if err != nil {
			return err
		}

		// TODO: Set document in cache
		//d.redis.Set(ctx, fmt.Sprintf("document:%s/version:%s", doc.ID, doc.Version), doc, 0)

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
	// soft delete the document
	err := model.DeleteDocument(d.db, id.String())
	if err != nil {
		return nil, err
	}

	return &v1.DeleteDocumentResponse{
		Document: &v1.Document{
			Id: id.String(),
		},
	}, nil
}
