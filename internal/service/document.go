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

	links := request.GetLinks()
	if links == nil {
		links = make(map[string]string)
	}
	linkData, err := json.Marshal(links)
	if err != nil {
		return nil, err
	}

	children := request.GetChildren()
	if children == nil {
		children = make([]string, 0)
	}
	childrenData, err := json.Marshal(children)
	if err != nil {
		return nil, err
	}
	childrenEncode, err := d.compress.Encode(childrenData)
	if err != nil {
		return nil, err
	}

	projectID := request.GetProjectId()
	doc := &model.Document{
		ProjectID: projectID,
		Meta:      string(metaData),
		Content:   string(contentData),
		Links:     string(linkData),
		Children:  string(childrenEncode),
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
			SourceId:      source.SourceID,
			SourceVersion: model.CurrentDocumentVersion,
		})
	}

	return &v1.ListBacklinksResponse{
		Links: backlinksProto,
	}, nil
}

// GetDocument retrieves a document.
func (d DocumentService) GetDocument(ctx context.Context, request *v1.GetDocumentRequest) (*v1.GetDocumentResponse, error) {
	// TODO: first look into cache
	// doc, err := d.cache.Get(ctx, fmt.Sprintf("document:%s/version:%s", request.Id, request.Version))
	// if err == nil {
	// 	return doc, nil
	// }
	// Get document from database
	doc, err := d.store.GetDocument(ctx, uuid.MustParse(request.GetDocumentId()))
	if err != nil {
		return nil, err
	}

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

	children := make([]string, 0)

	if doc.Children != "" {
		childrenData, err := d.compress.Decode([]byte(doc.Children))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(childrenData, &children)
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
			Children:  children,
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

	// List documents from ids (return full documents)
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
		linksData, err := d.compress.Decode([]byte(doc.Links))
		if err != nil {
			return nil, err
		}
		childrenData, err := d.compress.Decode([]byte(doc.Children))
		if err != nil {
			return nil, err
		}

		links, err := parseLinks(string(linksData))
		if err != nil {
			return nil, err
		}

		children, err := parseChildren(string(childrenData))
		if err != nil {
			return nil, err
		}

		documentsProto = append(documentsProto, &v1.Document{
			Id:        doc.ID,
			Meta:      doc.Meta,
			Version:   doc.Version,
			Links:     links,
			Children:  children,
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
		doc, err = tx.GetDocument(ctx, uuid.MustParse(request.GetDocumentId()))
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

		// compress the meta
		if request.Meta != nil {
			metaContent, err := d.compress.Encode([]byte(request.GetMeta()))
			if err != nil {
				return err
			}
			doc.Meta = string(metaContent)
		}

		// overwrite the links
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

		// overwrite the children
		if request.Children != nil {
			children, err := json.Marshal(request.GetChildren())
			if err != nil {
				return err
			}
			childrenData, err := d.compress.Encode(children)
			if err != nil {
				return err
			}
			doc.Children = string(childrenData)
		}

		createBackup := func() error {
			err = tx.CreateDocumentBackup(ctx, &model.DocumentBackup{
				ID:      doc.ID,
				Version: doc.Version,
				Meta:    doc.Meta,
				Content: doc.Content,
				Links:   doc.Links,
			})

			return err
		}

		updateLinks := func() error {
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
						return ErrInvalidLinkFormat
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
						return ErrInvalidLinkFormat
					}

					targetID := tokens[0]
					targetVersion := tokens[1]
					if _, ok := oldLinks[key]; !ok || oldLinks[key] != targetVersion {
						newLinksMap[targetID] = targetVersion

						// check if the target document is unpublished
						if targetVersion == model.CurrentDocumentVersion {
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
					var ids []uuid.UUID
					for _, doc := range unPublishedDocLinks {
						ids = append(ids, uuid.MustParse(doc.ID))
					}

					projectIDs, err := tx.ListDocumentProjectIDs(ctx, ids)
					if err != nil {
						return err
					}

					for _, link := range unPublishedDocLinks {
						if projectID, ok := projectIDs[uuid.MustParse(link.ID)]; ok {
							link.ProjectID = projectID.String()
						} else {
							return errors.New("target documents do not exist")
						}
					}

					problem, err := tx.ExistsDocuments(ctx, unPublishedDocLinks)
					if err != nil {
						return err
					}
					if problem {
						return errors.New("linked documents do not exist")
					}
				}

				// check if the target published documents exist, if not return an error
				if len(publishedDocLinks) != 0 {
					var ids []uuid.UUID
					for _, doc := range publishedDocLinks {
						ids = append(ids, uuid.MustParse(doc.ID))
					}

					projectIDs, err := tx.ListDocumentProjectIDs(ctx, ids)
					if err != nil {
						return err
					}

					for _, link := range publishedDocLinks {
						if projectID, ok := projectIDs[uuid.MustParse(link.ID)]; ok {
							link.ProjectID = projectID.String()
						} else {
							return errors.New("linked published documents do not exist")
						}
					}

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

			return nil
		}

		// if the version clocks matches, update the document
		// if the request contains a JSONDIFF, apply the patch
		if versionMatch && request.GetKind() == v1.UpdateKind_JSONPATCH {
			if overwrite {
				return errors.New("overwrite not allowed for JSONDIFF")
			}

			err := createBackup()
			if err != nil {
				return err
			}

			// patch the content
			if request.Content != nil {
				contentData, err := d.compress.Encode([]byte(request.GetContent()))
				if err != nil {
					return err
				}

				jsonDoc := blocktree.NewJsonDoc(contentData)
				patch := blocktree.JsonPatch(request.GetContent())

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
			doc.Version = doc.Version + 1
			logrus.Infof("updating document id with patch: %v, version: %v", doc.ID, doc.Version)
			err = tx.UpdateDocument(ctx, doc)
			if err != nil {
				return err
			}

			err = updateLinks()
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
			err := createBackup()
			if err != nil {
				return err
			}

			if request.Content != nil {
				contentData, err := d.compress.Encode([]byte(request.GetContent()))
				if err != nil {
					return err
				}
				doc.Content = string(contentData)
			}
			doc.Version = doc.Version + 1

			if clone.Meta == doc.Meta && clone.Content == doc.Content && clone.Links == doc.Links && clone.Children == doc.Children {
				return errors.New("document is not changed, skipping update")
			}

			logrus.Infof("updating document id: %v, version: %v", doc.ID, doc.Version)
			err = tx.UpdateDocument(ctx, doc)
			if err != nil {
				return err
			}

			err = updateLinks()
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
		Id:      request.DocumentId,
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

// PublishDocuments publishes a document. This publishes multiple documents at once.
// This is an atomic operation. If any of the documents fail to publish, the operation is rolled back.
// This is useful for publishing documents that are linked to each other for example in a book.
// TODO: A document is linked to another document. If the linked document is not published, the operation should fail. (optional feature)
func (d DocumentService) PublishDocuments(ctx context.Context, request *v1.PublishDocumentsRequest) (*v1.PublishDocumentsResponse, error) {
	var docIDs []uuid.UUID
	for _, id := range request.GetDocumentIds() {
		docID, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		docIDs = append(docIDs, docID)
	}

	var latestDoc *model.PublishedDocument
	var documents []*v1.PublishedDocument

	// Publish the document in a transaction
	err := d.store.Transaction(ctx, func(tx store.Store) error {
		for _, docID := range docIDs {
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
			if !request.GetForce() &&
				lastPublishedDoc != nil &&
				doc != nil &&
				lastPublishedDoc.Meta == doc.Meta &&
				lastPublishedDoc.Content == doc.Content &&
				lastPublishedDoc.Links == doc.Links &&
				lastPublishedDoc.Children == doc.Children {
				// return an error if the document is already published
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
					Children:  doc.Children,
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
					Children:  doc.Children,
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
					return ErrInvalidLinkFormat
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

			// collect the updated document
			documents = append(documents, &v1.PublishedDocument{
				Id:      doc.ID,
				Version: latestDoc.Version,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.PublishDocumentsResponse{
		Documents: documents,
	}, nil
}

func parseLinks(links string) (map[string]string, error) {
	var linksMap map[string]string
	err := json.Unmarshal([]byte(links), &linksMap)
	if err != nil {
		return nil, err
	}
	return linksMap, nil
}

func parseChildren(children string) ([]string, error) {
	var childrenList []string
	err := json.Unmarshal([]byte(children), &childrenList)
	if err != nil {
		return nil, err
	}
	return childrenList, nil
}
