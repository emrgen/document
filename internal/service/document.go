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
	"strings"
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

func (d DocumentService) ListDocumentVersions(ctx context.Context, request *v1.ListDocumentVersionsRequest) (*v1.ListDocumentVersionsResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	backups, err := d.store.ListDocumentBackupVersions(ctx, docID)
	if err != nil {
		return nil, err
	}

	var versions []*v1.DocumentVersion
	doc, err := d.store.GetDocument(ctx, docID)
	if err != nil {
		return nil, err
	}
	versions = append(versions, &v1.DocumentVersion{
		Version:   doc.Version,
		CreatedAt: timestamppb.New(doc.CreatedAt),
	})

	for _, backup := range backups {
		versions = append(versions, &v1.DocumentVersion{
			Version:   backup.Version,
			CreatedAt: timestamppb.New(backup.CreatedAt),
		})
	}

	return &v1.ListDocumentVersionsResponse{
		Versions:      versions,
		CreatedAt:     timestamppb.New(doc.CreatedAt),
		LatestVersion: doc.Version,
	}, nil
}

func (d DocumentService) ListBacklinks(ctx context.Context, request *v1.ListBacklinksRequest) (*v1.ListBacklinksResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	backlinks, err := d.store.ListBacklinks(ctx, docID)
	if err != nil {
		return nil, err
	}

	var backlinksProto []*v1.Link
	for _, source := range backlinks {
		backlinksProto = append(backlinksProto, &v1.Link{
			SourceId: source.SourceID,
		})
	}

	return &v1.ListBacklinksResponse{
		Links: backlinksProto,
	}, nil
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
	logrus.Infof("getting document id: %v", request.GetDocumentId())
	// Get document from database
	doc, err := d.store.GetDocument(ctx, uuid.MustParse(request.GetDocumentId()))
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

	links := make(map[string]string)
	if doc.Links != "" {
		linksData, err := d.compress.Decode([]byte(doc.Links))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(linksData, &links)
		if err != nil {
			return nil, err
		}
	}

	return &v1.GetDocumentResponse{
		Document: &v1.Document{
			Id:        doc.ID,
			Content:   string(contentData),
			Meta:      string(metaData),
			Links:     links,
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
			Links:   doc.Links,
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

				err = jsonDoc.Apply(patch)
				if err != nil {
					return err
				}

				data, err := d.compress.Encode([]byte(jsonDoc.String()))
				if err != nil {
					return err
				}
				doc.Meta = string(data)
			}

			if request.Links != nil {
				links := request.GetLinks()
				patch, err := json.Marshal(links)
				if err != nil {
					return err
				}
				linksContent, err := d.compress.Encode(patch)
				if err != nil {
					return err
				}

				jsonDoc := blocktree.NewJsonDoc(linksContent)

				err = jsonDoc.Apply(patch)
				if err != nil {
					return err
				}

				data, err := d.compress.Encode([]byte(jsonDoc.String()))
				if err != nil {
					return err
				}
				doc.Links = string(data)
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
				Links:   doc.Links,
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

			if request.Links != nil {
				links := request.GetLinks()
				linksData, err := json.Marshal(links)
				if err != nil {
					return err
				}
				linksContent, err := d.compress.Encode(linksData)
				if err != nil {
					return err
				}
				doc.Links = string(linksContent)
			}

			if request.Content != nil {
				contentData, err := d.compress.Encode([]byte(request.GetContent()))
				if err != nil {
					return err
				}
				doc.Content = string(contentData)
			}
			doc.Version = doc.Version + 1

			if clone.Meta == doc.Meta && clone.Content == doc.Content && clone.Links == doc.Links {
				return errors.New("document is not changed, skipping update")
			}

			logrus.Infof("updating document id: %v, version: %v", doc.ID, doc.Version)
			err = tx.UpdateDocument(ctx, doc)
			if err != nil {
				return err
			}

			// check if the links are changed and update the backlinks
			if clone.Links != doc.Links {
				logrus.Infof("old links: %v, new links: %v", clone.Links, request.GetLinks())

				newLinks := request.GetLinks()
				oldLinks := make(map[string]string)
				err = json.Unmarshal([]byte(clone.Links), &oldLinks)
				if err != nil {
					return err
				}

				// collect broken links
				brokenLinks := make(map[string]string)
				for key := range oldLinks {
					tokens := strings.Split(key, "@")
					if len(tokens) != 2 {
						return errors.New("invalid link format")
					}

					if _, ok := newLinks[key]; !ok {
						brokenLinks[tokens[0]] = tokens[1]
					}
				}

				// collect new links
				newLinksMap := make(map[string]string)
				publishedDocLinks := make([]*model.PublishedDocument, 0)
				unPublishedDocLinks := make([]*model.Document, 0)
				for key := range newLinks {
					tokens := strings.Split(key, "@")
					if len(tokens) != 2 {
						return errors.New("invalid link format")
					}

					targetID := tokens[0]
					targetVersion := tokens[1]
					if _, ok := oldLinks[key]; !ok || oldLinks[key] != targetVersion {
						newLinksMap[key] = targetVersion

						// check if the target document is unpublished
						if targetVersion == model.UnpublishedDocumentVersion {
							unPublishedDocLinks = append(unPublishedDocLinks, &model.Document{
								ID: targetID,
							})
						}

						publishedDocLinks = append(publishedDocLinks, &model.PublishedDocument{
							ID:      targetID,
							Version: targetVersion,
						})

					}
				}

				// check if the target unpublished documents exist, if not return an error
				if len(unPublishedDocLinks) != 0 {
					problem, err := tx.ExistsDocuments(ctx, unPublishedDocLinks)
					if err != nil {
						return err
					}
					if problem {
						return errors.New("target documents do not exist")
					}
				}

				// check if the target published documents exist, if not return an error
				if len(publishedDocLinks) != 0 {
					problem, err := tx.ExistsPublishedDocuments(ctx, publishedDocLinks)
					if err != nil {
						return err
					}

					if problem {
						return errors.New("target documents do not exist")
					}
				}

				// new link models
				var newLinkModels []*model.Link
				for key, version := range newLinksMap {
					newLinkModels = append(newLinkModels, &model.Link{
						SourceID:      doc.ID,
						TargetID:      key,
						TargetVersion: version,
					})
				}

				// old link models
				var brokenLinkModels []*model.Link
				for key, version := range brokenLinks {
					brokenLinkModels = append(brokenLinkModels, &model.Link{
						SourceID:      doc.ID,
						TargetID:      key,
						TargetVersion: version,
					})
				}

				logrus.Infof("broken links: %v, new links: %v", brokenLinkModels, newLinkModels)

				if len(brokenLinkModels) != 0 {
					// delete old links
					err = tx.DeleteBacklinks(ctx, brokenLinkModels)
					if err != nil {
						return err
					}
				}

				if len(newLinkModels) != 0 {
					// create new links
					err = tx.CreateBacklinks(ctx, newLinkModels)
					if err != nil {
						return err
					}
				}
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

	// Publish the document in a transaction
	err = d.store.Transaction(ctx, func(tx store.Store) error {
		// Get the document from the database
		doc, err := tx.GetDocument(ctx, docID)
		if err != nil {
			return err
		}

		// Get latest published document
		lastPublishedDoc, err := tx.GetLatestPublishedDocument(ctx, docID)
		if err != nil && !errors.Is(store.ErrLatestPublishedDocumentNotFound, err) {
			return err
		}

		// Check if the document is already published with the same content and metadata
		if !request.GetForce() && lastPublishedDoc != nil && doc != nil && lastPublishedDoc.Meta == doc.Meta && lastPublishedDoc.Content == doc.Content && lastPublishedDoc.Links == doc.Links {
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
				ID:        doc.ID,
				ProjectID: doc.ProjectID,
				Version:   version.String(),
				Meta:      doc.Meta,
				Links:     doc.Links,
				Content:   doc.Content,
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
				ID:        doc.ID,
				ProjectID: doc.ProjectID,
				Version:   version.String(),
				Meta:      doc.Meta,
				Links:     doc.Links,
				Content:   doc.Content,
			}

			err = updateLatestPublishedDocumentCache(ctx, d.cache, doc.ID, latestDoc)
			if err != nil {
				logrus.Errorf("error updating cache: %v", err)
			}
		}

		err = tx.PublishDocument(ctx, latestDoc)
		if err != nil {
			return err
		}

		// get the links
		links := make(map[string]interface{})
		if latestDoc.Links != "" {
			linksData, err := d.compress.Decode([]byte(doc.Links))
			if err != nil {
				return err
			}
			err = json.Unmarshal(linksData, &links)
			if err != nil {
				return err
			}
		}

		// create new links
		newLinks := make([]*model.PublishedLink, 0)

		for target := range links {
			tokens := strings.Split(target, "@")
			if len(tokens) != 2 {
				return errors.New("invalid link format")
			}

			targetID := tokens[0]
			targetVersion := tokens[1]

			newLinks = append(newLinks, &model.PublishedLink{
				SourceID:      doc.ID,
				SourceVersion: latestDoc.Version,
				TargetID:      targetID,
				TargetVersion: targetVersion,
			})
		}

		if len(newLinks) > 0 {
			// create new links
			err = tx.CreatePublishedLinks(ctx, newLinks)
			if err != nil {
				return err
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
