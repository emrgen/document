package service

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/cache"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/model"
	"github.com/emrgen/document/internal/store"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// NewPublishedDocumentService creates a new PublishedDocumentService.
func NewPublishedDocumentService(compress compress.Compress, store store.Store, cache *cache.Redis) *PublishedDocumentService {
	return &PublishedDocumentService{
		store:    store,
		cache:    cache,
		compress: compress,
	}
}

var _ v1.PublishedDocumentServiceServer = (*PublishedDocumentService)(nil)

type PublishedDocumentService struct {
	store    store.Store
	compress compress.Compress
	cache    *cache.Redis
	v1.UnimplementedPublishedDocumentServiceServer
}

// GetPublishedDocumentMeta retrieves the meta information of a published document by ID.
func (p *PublishedDocumentService) GetPublishedDocumentMeta(ctx context.Context, request *v1.GetPublishedDocumentMetaRequest) (*v1.GetPublishedDocumentMetaResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	version := request.GetVersion()
	// get the latest published document meta

	var latestPubMeta *model.PublishedDocumentMeta

	latestMeta, err := p.store.GetLatestPublishedDocumentMeta(ctx, docID)
	if err != nil {
		return nil, err
	}
	if latestMeta != nil {
		latestPubMeta = latestMeta.IntoPublishedDocumentMeta()
	}

	var meta *model.PublishedDocumentMeta

	if version == "latest" || version == "" {
		meta = latestPubMeta
	} else {
		meta, err = p.store.GetPublishedDocumentMetaByVersion(ctx, docID, version)
		if err != nil {
			return nil, err
		}
	}

	if meta == nil {
		return nil, fmt.Errorf("document not found")
	}

	var children []string
	if meta.Children != "" {
		err = json.Unmarshal([]byte(meta.Children), &children)
		if err != nil {
			return nil, err
		}
	}

	var links map[string]string
	if meta.Links != "" {
		err = json.Unmarshal([]byte(meta.Links), &links)
		if err != nil {
			return nil, err
		}
	}

	var latestVersion *v1.PublishedDocumentVersion
	if latestPubMeta != nil {
		latestVersion = &v1.PublishedDocumentVersion{
			Version:   latestPubMeta.Version,
			CreatedAt: timestamppb.New(latestPubMeta.CreatedAt),
		}
	}

	return &v1.GetPublishedDocumentMetaResponse{
		Document: &v1.PublishedDocument{
			Id:            meta.ID,
			Meta:          meta.Meta,
			Links:         links,
			Children:      children,
			LatestVersion: latestVersion,
		},
	}, nil
}

// ListPublishedBacklinks retrieves a list of published backlinks by document ID.
func (p *PublishedDocumentService) ListPublishedBacklinks(ctx context.Context, request *v1.ListPublishedBacklinksRequest) (*v1.ListPublishedBacklinksResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	backlinks, err := p.store.ListPublishedBacklinks(ctx, docID, request.GetVersion())
	if err != nil {
		return nil, err
	}

	var backlinksProto []*v1.Link
	for _, link := range backlinks {
		backlinksProto = append(backlinksProto, &v1.Link{
			SourceId:      link.SourceID,
			SourceVersion: link.SourceVersion,
			TargetId:      link.TargetID,
			TargetVersion: link.TargetVersion,
		})
	}

	return &v1.ListPublishedBacklinksResponse{
		Links: backlinksProto,
	}, nil
}

// GetPublishedDocument retrieves a published document by ID.
func (p *PublishedDocumentService) GetPublishedDocument(ctx context.Context, request *v1.GetPublishedDocumentRequest) (*v1.GetPublishedDocumentResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, err
	}

	version := request.GetVersion()
	var publishedDocument *model.PublishedDocument

	if version == "latest" || version == "" {
		// get the latest published publishedDocument
		doc, err := p.store.GetLatestPublishedDocument(ctx, id)
		if err != nil {
			return nil, err
		}
		publishedDocument = doc.IntoPublishedDocument()
	} else {
		// get the published publishedDocument by version
		doc, err := p.store.GetPublishedDocumentByVersion(ctx, id, version)
		if err != nil {
			return nil, err
		}
		publishedDocument = doc
	}

	metaData, err := p.compress.Decode([]byte(publishedDocument.Meta))
	if err != nil {
		return nil, err
	}

	latestDoc, err := p.store.GetLatestPublishedDocumentMeta(ctx, id)
	if err != nil {
		return nil, err
	}

	linkData, err := p.compress.Decode([]byte(publishedDocument.Links))
	if err != nil {
		return nil, err
	}
	links, err := parseLinks(string(linkData))
	if err != nil {
		return nil, err
	}

	childrenData, err := p.compress.Decode([]byte(publishedDocument.Children))
	if err != nil {
		return nil, err
	}
	children, err := parseChildren(string(childrenData))
	if err != nil {
		return nil, err
	}

	latestVersion := &v1.PublishedDocumentVersion{
		Version:   latestDoc.Version,
		CreatedAt: timestamppb.New(latestDoc.UpdatedAt),
	}
	document := &v1.PublishedDocument{
		Id:            publishedDocument.ID,
		Meta:          string(metaData),
		Version:       publishedDocument.Version,
		Content:       publishedDocument.Content,
		Links:         links,
		Children:      children,
		LatestVersion: latestVersion,
	}

	return &v1.GetPublishedDocumentResponse{
		Document: document,
	}, nil
}

// ListPublishedDocuments retrieves a list of published documents by project ID.
func (p *PublishedDocumentService) ListPublishedDocuments(ctx context.Context, request *v1.ListPublishedDocumentsRequest) (*v1.ListPublishedDocumentsResponse, error) {
	projectID, err := uuid.Parse(request.GetProjectId())
	if err != nil {
		return nil, err
	}

	// get full document when idVersions are provided
	if len(request.GetIdVersions()) > 0 {
		var idVersions []*model.IDVersion
		for _, idVersion := range request.GetIdVersions() {
			idVersions = append(idVersions, &model.IDVersion{
				ID:      idVersion.GetId(),
				Version: idVersion.GetVersion(),
			})
		}

		docs, err := p.store.ListPublishedDocumentsByIdVersion(ctx, projectID, idVersions)
		if err != nil {
			return nil, err
		}

		var documents []*v1.PublishedDocument
		for _, doc := range docs {
			metaData, err := p.compress.Decode([]byte(doc.Meta))
			if err != nil {
				return nil, err
			}

			linksData, err := p.compress.Decode([]byte(doc.Links))
			if err != nil {
				return nil, err
			}
			childrenData, err := p.compress.Decode([]byte(doc.Children))
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

			content, err := p.compress.Decode([]byte(doc.Content))

			documents = append(documents, &v1.PublishedDocument{
				Id:       doc.ID,
				Meta:     string(metaData),
				Links:    links,
				Children: children,
				Version:  doc.Version,
				Content:  string(content),
			})
		}
	}

	docs, err := p.store.ListLatestPublishedDocuments(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var documents []*v1.PublishedDocument
	for _, doc := range docs {
		metaData, err := p.compress.Decode([]byte(doc.Meta))
		if err != nil {
			return nil, err
		}

		linksData, err := p.compress.Decode([]byte(doc.Links))
		if err != nil {
			return nil, err
		}
		childrenData, err := p.compress.Decode([]byte(doc.Children))
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

		documents = append(documents, &v1.PublishedDocument{
			Id:       doc.ID,
			Meta:     string(metaData),
			Links:    links,
			Children: children,
			Version:  doc.Version,
		})
	}

	return &v1.ListPublishedDocumentsResponse{
		Documents: documents,
	}, nil

}

// ListPublishedDocumentVersions retrieves a list of published document versions by ID.
func (p *PublishedDocumentService) ListPublishedDocumentVersions(ctx context.Context, request *v1.ListPublishedDocumentVersionsRequest) (*v1.ListPublishedDocumentVersionsResponse, error) {
	docID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, err
	}

	metaList, err := p.store.ListPublishedDocumentVersions(ctx, docID)
	if err != nil {
		return nil, err
	}

	var versions []*v1.PublishedDocumentVersion
	for _, meta := range metaList {
		versions = append(versions, &v1.PublishedDocumentVersion{
			Version:   meta.Version,
			CreatedAt: timestamppb.New(meta.CreatedAt),
		})
	}

	req := &v1.ListPublishedDocumentVersionsResponse{
		Id:       docID.String(),
		Versions: versions,
	}

	// get the latest version if there are versions available
	// this is useful for clients that want to get the latest version along with the versions list
	if len(versions) > 0 {
		latestMeta, err := p.store.GetLatestPublishedDocumentMeta(ctx, docID)
		if err != nil {
			return nil, err
		}
		req.LatestVersion = latestMeta.Version
	}

	return req, nil
}

func getPublishedDocumentByVersion(ctx context.Context, cache *cache.Redis, id uuid.UUID, version string) (*v1.PublishedDocument, error) {
	cached, err := cache.Get(ctx, fmt.Sprintf("%s@%s", id, version))
	if err == nil {
		document := &v1.PublishedDocument{}
		data, ok := cached.(string)
		if ok {
			err = json.Unmarshal([]byte(data), document)
			if err != nil {
				return nil, err
			}

			return document, nil
		}
	}

	return nil, err
}

// updateLatestPublishedDocumentCache updates the latest published document cache.
// NOTE: without cache update the latest document will not be available immediately.
func updateLatestPublishedDocumentCache(ctx context.Context, cache *cache.Redis, id string, doc *model.PublishedDocument) error {
	links := make(map[string]string)
	if doc.Links != "" {
		err := json.Unmarshal([]byte(doc.Links), &links)
		if err != nil {
			return err
		}
	}

	docProto := &v1.PublishedDocument{
		Id:      doc.ID,
		Version: doc.Version,
		Meta:    doc.Meta,
		Links:   links,
		Content: doc.Content,
	}
	docData, err := json.Marshal(docProto)
	if err != nil {
		return err
	}

	err = cache.Set(ctx, fmt.Sprintf("%s@%s", id, "latest"), string(docData), time.Minute*5)
	if err != nil {
		return err
	}

	err = cache.Set(ctx, fmt.Sprintf("%s@%s", id, doc.Version), string(docData), time.Minute*5)
	if err != nil {
		return err
	}

	return nil
}
