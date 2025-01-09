package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/emrgen/document/internal/store"
	"github.com/sirupsen/logrus"

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

	data, err := d.compress.Encode([]byte(request.GetContent()))
	if err != nil {
		return nil, err
	}

	projectID := request.GetProjectId()
	doc := &model.Document{
		ProjectID: projectID,
		Meta:      request.GetMeta(),
		Content:   string(data),
		Version:   0,
	}

	if request.DocumentId != nil {
		doc.ID = request.GetDocumentId()
	} else {
		doc.ID = uuid.New().String()
	}

	err = d.db.Create(doc).Error
	if err != nil {
		return nil, err
	}

	return &v1.CreateDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Meta:      request.GetMeta(),
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
			Meta:      doc.Meta,
			Content:   string(data),
			Version:   doc.Version,
			CreatedAt: timestamppb.New(doc.CreatedAt),
			UpdatedAt: timestamppb.New(doc.UpdatedAt),
		},
	}, nil
}

// ListDocuments lists documents.
func (d DocumentService) ListDocuments(ctx context.Context, request *v1.ListDocumentsRequest) (*v1.ListDocumentsResponse, error) {
	var err error
	projectID, err := uuid.Parse(request.GetProjectId())
	if err != nil {
		return nil, err
	}

	// List documents from ids
	if len(request.GetDocumentIds()) > 0 {
		var documents []*model.Document
		err = d.db.Where("project_id = ? AND id IN ?", projectID.String(), request.GetDocumentIds()).Find(&documents).Error
		if err != nil {
			return nil, err
		}

		var documentsProto []*v1.Document
		for _, doc := range documents {
			documentsProto = append(documentsProto, &v1.Document{
				Id:        doc.ID,
				Meta:      doc.Meta,
				Version:   doc.Version,
				CreatedAt: timestamppb.New(doc.CreatedAt),
				UpdatedAt: timestamppb.New(doc.UpdatedAt),
			})
		}

		return &v1.ListDocumentsResponse{
			Documents: documentsProto,
			Total:     int32(len(documents)),
		}, nil
	}

	// Get documents from database page by page
	var documents []*model.Document
	err = d.db.Where("project_id = ?", projectID.String()).Order("created_at DESC").Find(&documents).Error
	if err != nil {
		return nil, err
	}
	var total int64
	err = d.db.Model(&model.Document{}).Where("project_id = ?", projectID.String()).Count(&total).Error

	var documentsProto []*v1.Document
	for _, doc := range documents {
		documentsProto = append(documentsProto, &v1.Document{
			Id:        doc.ID,
			Meta:      doc.Meta,
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
	var doc *model.Document
	err = d.db.Transaction(func(tx *gorm.DB) error {
		// Get document from database
		doc, err = model.GetDocument(d.db, request.Id)
		if err != nil {
			return err
		}

		logrus.Info("updating document", request.GetVersion(), doc.Version)

		overwrite := request.Version == -1
		versionMatch := request.Version == doc.Version+1

		if !overwrite && versionMatch {
			return errors.New(fmt.Sprintf("current version: %d, expected version %d, provider version: %d, ", doc.Version, doc.Version+1, request.GetVersion()))
		}

		if versionMatch {
			//updateFields := request.Title != nil || request.Data != nil || request.Summary != nil || request.Excerpt != nil || request.Thumbnail != nil
			//if updateFields {
			//	// merge parts into content
			//
			//	// Create a backup of the document
			//	err = d.store.CreateDocumentBackup(ctx, &model.DocumentBackup{
			//		ID:        doc.ID,
			//		Version:   doc.Version,
			//		Content:   doc.Content,
			//		Title:     doc.Title,
			//		Summary:   doc.Summary,
			//		Excerpt:   doc.Excerpt,
			//		Thumbnail: doc.Thumbnail,
			//	})
			//	if err != nil {
			//		return err
			//	}
			//

			// Update document in database
			//if request.Title != nil {
			//	doc.Title = request.GetTitle()
			//}
			//
			//if request.Data != nil {
			//	doc.Data = request.GetData()
			//}
			//
			//if request.Summary != nil {
			//	doc.Summary = request.GetSummary()
			//}
			//
			//if request.Excerpt != nil {
			//	doc.Excerpt = request.GetExcerpt()
			//}
			//
			//if request.Thumbnail != nil {
			//	doc.Thumbnail = request.GetThumbnail()
			//}

			data, err := d.compress.Encode([]byte(request.GetContent()))
			if err != nil {
				return err
			}

			doc.Content = string(data)
			// TODO: if the parts are too large, we need to merge them
			doc.Version = doc.Version + 1
		}

		if overwrite {
			// Create a backup of the document
			//err = d.store.CreateDocumentBackup(ctx, &model.DocumentBackup{
			//	ID:      doc.ID,
			//	Version: doc.Version,
			//	Content: doc.Content,
			//})
			//if err != nil {
			//	return err
			//}

			data, err := d.compress.Encode([]byte(request.GetContent()))
			if err != nil {
				return err
			}
			doc.Content = string(data)
			doc.Version = doc.Version + 1
		}

		// if the content is not nil, update the content
		// otherwise, append the parts to the document
		logrus.Info("updating document", doc.ID, doc.Version)
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
		Version: uint32(doc.Version),
	}, nil
}

// DeleteDocument deletes a document.
func (d DocumentService) DeleteDocument(ctx context.Context, request *v1.DeleteDocumentRequest) (*v1.DeleteDocumentResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, err
	}

	// soft delete the document
	err = model.DeleteDocument(d.db, id.String())
	if err != nil {
		return nil, err
	}

	return &v1.DeleteDocumentResponse{
		Document: &v1.Document{
			Id: id.String(),
		},
	}, nil
}

// EraseDocument erases a document.
func (d DocumentService) EraseDocument(ctx context.Context, request *v1.EraseDocumentRequest) (*v1.EraseDocumentResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, err
	}

	// hard delete the document
	err = d.db.WithContext(ctx).Unscoped().Delete(&model.Document{}, id.String()).Error
	if err != nil {
		return nil, err
	}

	return &v1.EraseDocumentResponse{
		Document: &v1.Document{
			Id: id.String(),
		},
	}, nil
}
