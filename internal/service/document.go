package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/emrgen/blocktree"
	_ "github.com/emrgen/blocktree"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/cache"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/model"
	"github.com/emrgen/document/internal/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"time"
)

var (
	_ v1.DocumentServiceServer = (*DocumentService)(nil)
)

// NewDocumentService creates a new DocumentService.
func NewDocumentService(compress compress.Compress, store store.Store, redis *cache.Redis) *DocumentService {
	service := &DocumentService{
		cache:    redis,
		store:    store,
		compress: compress,
	}

	return service
}

// DocumentService is a service for managing documents.
type DocumentService struct {
	compress compress.Compress
	cache    *cache.Redis
	store    store.Store
	v1.UnimplementedDocumentServiceServer
}

// CreateDocument creates a new document.
func (d DocumentService) CreateDocument(ctx context.Context, request *v1.CreateDocumentRequest) (*v1.CreateDocumentResponse, error) {
	var err error

	metaData, err := d.compress.Encode([]byte(request.GetMeta()))
	if err != nil {
		return nil, err
	}

	contentData, err := d.compress.Encode([]byte(request.GetContent()))
	if err != nil {
		return nil, err
	}

	projectID := request.GetProjectId()
	doc := &model.Document{
		ProjectID: projectID,
		Meta:      string(metaData),
		Content:   string(contentData),
		Version:   0,
	}

	if request.DocumentId != nil {
		doc.ID = request.GetDocumentId()
	} else {
		doc.ID = uuid.New().String()
	}

	err = d.store.CreateDocument(ctx, doc)
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
	// doc, err := d.cache.Get(ctx, fmt.Sprintf("document:%s/version:%s", request.Id, request.Version))
	// if err == nil {
	// 	return doc, nil
	// }

	logrus.Infof("getting document id: %v", request.Id)
	// Get document from database
	doc, err := d.store.GetDocument(ctx, uuid.MustParse(request.GetId()))
	if err != nil {
		return nil, err
	}

	logrus.Infof("document found id: %v, version: %v", doc.ID, doc.Version)

	metaData, err := d.compress.Decode([]byte(doc.Meta))
	if err != nil {
		return nil, err
	}

	contentData, err := d.compress.Decode([]byte(doc.Content))
	if err != nil {
		return nil, err
	}

	return &v1.GetDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Content:   string(contentData),
			Meta:      string(metaData),
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
		var ids []uuid.UUID
		for _, id := range request.GetDocumentIds() {
			ids = append(ids, uuid.MustParse(id))
		}

		documents, err = d.store.ListDocumentsFromIDs(ctx, ids)
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
	documents, total, err := d.store.ListDocuments(ctx, projectID)
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
		Total:     int32(total),
	}, nil
}

// UpdateDocument updates a document.
func (d DocumentService) UpdateDocument(ctx context.Context, request *v1.UpdateDocumentRequest) (*v1.UpdateDocumentResponse, error) {
	var err error
	var doc *model.Document

	err = d.store.Transaction(ctx, func(tx store.Store) error {
		// Get document from database
		doc, err = tx.GetDocument(ctx, uuid.MustParse(request.GetId()))
		clone := &model.Document{
			ID:      doc.ID,
			Version: doc.Version,
			Meta:    doc.Meta,
			Content: doc.Content,
		}
		if err != nil {
			return err
		}

		logrus.Infof("old version: %v, new version: %v", doc.Version, request.GetVersion())

		overwrite := request.Version == -1
		versionMatch := request.Version == doc.Version+1

		if !overwrite && !versionMatch {
			return status.New(codes.FailedPrecondition, fmt.Sprintf("current version: %d, expected version %d, provider version: %d, ", doc.Version, doc.Version+1, request.GetVersion())).Err()
		}

		// if the version clocks matches, update the document
		if versionMatch && request.GetKind() == v1.UpdateKind_JSONPATCH {
			if request.Meta != nil {
				metaContent, err := d.compress.Encode([]byte(request.GetMeta()))
				if err != nil {
					return err
				}

				jsonDoc := blocktree.NewJsonDoc(metaContent)
				patch := blocktree.JsonPatch(request.GetMeta())

				logrus.Infof("applying patch: %v", patch)

				err = jsonDoc.Apply(patch)
				if err != nil {
					return err
				}

				logrus.Infof("patched content: %v", jsonDoc.String())

				data, err := d.compress.Encode([]byte(jsonDoc.String()))
				if err != nil {
					return err
				}
				doc.Meta = string(data)
			}

			if request.Content != nil {
				contentData, err := d.compress.Encode([]byte(request.GetContent()))
				if err != nil {
					return err
				}

				// merge the content data
				jsonDoc := blocktree.NewJsonDoc(contentData)
				patch := blocktree.JsonPatch(request.GetMeta())

				err = jsonDoc.Apply(patch)
				if err != nil {
					return err
				}

				data, err := d.compress.Encode([]byte(jsonDoc.String()))
				if err != nil {
					return err
				}

				doc.Content = string(data)

			}
			// TODO: if the parts are too large, we need to merge them
			doc.Version = doc.Version + 1
			logrus.Infof("updating document id: %v, version: %v", doc.ID, doc.Version)
			err = tx.UpdateDocument(ctx, doc)
			if err != nil {
				return err
			}
		}

		// explicitly overwrite the document
		// or the version matches and the kind is not JSONDIFF as JSONDIFF is handled above
		if overwrite || versionMatch && request.GetKind() != v1.UpdateKind_JSONPATCH {
			// TODO: check if document hash is the same

			// Create a backup of the document
			logrus.Infof("creating backup for document id: %v, version: %v", doc.ID, doc.Version)
			err = tx.CreateDocumentBackup(ctx, &model.DocumentBackup{
				ID:      doc.ID,
				Version: doc.Version,
				Meta:    doc.Meta,
				Content: doc.Content,
			})
			if err != nil {
				return err
			}

			if request.Meta != nil {
				metaContent, err := d.compress.Encode([]byte(request.GetMeta()))
				if err != nil {
					return err
				}
				doc.Meta = string(metaContent)
			}

			if request.Content != nil {
				contentData, err := d.compress.Encode([]byte(request.GetContent()))
				if err != nil {
					return err
				}
				doc.Content = string(contentData)
			}
			doc.Version = doc.Version + 1

			if clone.Meta == doc.Meta && clone.Content == doc.Content {
				return errors.New("document is not changed, skipping update")
			}

			logrus.Infof("updating document id: %v, version: %v", doc.ID, doc.Version)
			err = tx.UpdateDocument(ctx, doc)
			if err != nil {
				return err
			}
		}

		// TODO: Set document in cache
		//d.cache.Set(ctx, fmt.Sprintf("document:%s/version:%s", doc.ID, doc.Version), doc, 0)

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
	err = d.store.DeleteDocument(ctx, id)
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
	err = d.store.EraseDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.EraseDocumentResponse{
		Document: &v1.Document{
			Id: id.String(),
		},
	}, nil
}

// PublishDocument publishes a document.
func (d DocumentService) PublishDocument(ctx context.Context, request *v1.PublishDocumentRequest) (*v1.PublishDocumentResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	var latestDoc *model.PublishedDocument

	err = d.store.Transaction(ctx, func(tx store.Store) error {
		// Get the document from the database
		doc, err := tx.GetDocument(ctx, docID)
		if err != nil {
			return err
		}

		// Get latest published document
		lastPublishedDoc, err := tx.GetLatestPublishedDocument(ctx, docID)
		if err != nil && !errors.Is(gorm.ErrRecordNotFound, err) {
			return err
		}

		// Check if the document is already published with the same content and metadata
		if !request.GetForce() && lastPublishedDoc != nil && doc != nil && lastPublishedDoc.Meta == doc.Meta && lastPublishedDoc.Content == doc.Content {
			return errors.New("document is already published with version: " + lastPublishedDoc.Version)
		}

		// Create a new published document
		if lastPublishedDoc == nil {
			version, err := semver.NewVersion("0.0.1") // initial version
			if err != nil {
				return err
			}

			// if the version is provided, use it
			if request.GetVersion() != "" {
				newVersion, err := semver.NewVersion(request.GetVersion())
				if err != nil {
					return err
				}
				version = newVersion
			}

			latestDoc = &model.PublishedDocument{
				ID:      doc.ID,
				Version: version.String(),
				Meta:    doc.Meta,
				Content: doc.Content,
			}
			err = tx.PublishDocument(ctx, latestDoc)
			if err != nil {
				return err
			}
			err = updateLatestPublishedDocumentCache(ctx, d.cache, doc.ID, latestDoc)
			if err != nil {
				logrus.Errorf("error updating cache: %v", err)
			}
		} else {
			// Update the published document
			version, err := semver.NewVersion(lastPublishedDoc.Version)
			if err != nil {
				return err
			}
			*version = version.IncPatch()

			if request.GetVersion() != "" {
				newVersion, err := semver.NewVersion(request.GetVersion())
				if err != nil {
					return err
				}
				if newVersion.LessThan(version) {
					return fmt.Errorf("new version must be greater than current version")
				}

				version = newVersion
			}
			latestDoc = &model.PublishedDocument{
				ID:      doc.ID,
				Version: version.String(),
				Meta:    doc.Meta,
				Content: doc.Content,
			}
			err = tx.PublishDocument(ctx, latestDoc)
			if err != nil {
				return err
			}
			err = updateLatestPublishedDocumentCache(ctx, d.cache, doc.ID, latestDoc)
			if err != nil {
				logrus.Errorf("error updating cache: %v", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.PublishDocumentResponse{
		Document: &v1.PublishedDocument{
			Id:      latestDoc.ID,
			Version: latestDoc.Version,
		},
	}, nil
}

// updateLatestPublishedDocumentCache updates the latest published document cache.
// NOTE: without cache update the latest document will not be available immediately.
func updateLatestPublishedDocumentCache(ctx context.Context, cache *cache.Redis, id string, doc *model.PublishedDocument) error {
	docProto := &v1.PublishedDocument{
		Id:      doc.ID,
		Version: doc.Version,
		Meta:    doc.Meta,
		Content: doc.Content,
	}
	docData, err := json.Marshal(docProto)
	if err != nil {
		return err
	}

	err = cache.Set(ctx, fmt.Sprintf("%s-%s", id, "latest"), string(docData), time.Minute*5)
	if err != nil {
		return err
	}

	err = cache.Set(ctx, fmt.Sprintf("%s-%s", id, doc.Version), string(docData), time.Minute*5)
	if err != nil {
		return err
	}

	return nil
}
