package service

import (
	"context"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/store"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewPublishedDocumentService creates a new PublishedDocumentService.
func NewPublishedDocumentService(compress compress.Compress, store store.Store) *PublishedDocumentService {
	return &PublishedDocumentService{
		store:    store,
		compress: compress,
	}
}

var _ v1.PublishedDocumentServiceServer = (*PublishedDocumentService)(nil)

type PublishedDocumentService struct {
	store    store.Store
	compress compress.Compress
	v1.UnimplementedPublishedDocumentServiceServer
}

// GetPublishedDocument retrieves a published document by ID.
func (p *PublishedDocumentService) GetPublishedDocument(ctx context.Context, request *v1.GetPublishedDocumentRequest) (*v1.GetPublishedDocumentResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, err
	}

	version := request.GetVersion()
	var document *v1.PublishedDocument

	if version == "latest" || version == "" {
		doc, err := p.store.GetLatestPublishedDocument(ctx, id)
		if err != nil {
			return nil, err
		}
		metaData, err := p.compress.Decode([]byte(doc.Meta))
		if err != nil {
			return nil, err
		}
		document = &v1.PublishedDocument{
			Id:      doc.ID,
			Meta:    string(metaData),
			Version: doc.Version,
		}
	} else {
		doc, err := p.store.GetPublishedDocumentByVersion(ctx, id, version)
		if err != nil {
			return nil, err
		}
		metaData, err := p.compress.Decode([]byte(doc.Meta))
		if err != nil {
			return nil, err
		}
		document = &v1.PublishedDocument{
			Id:      doc.ID,
			Meta:    string(metaData),
			Version: doc.Version,
		}
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
	docs, err := p.store.ListLatestPublishedDocuments(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var documents []*v1.PublishedDocument
	for _, doc := range docs {
		metaData, err := p.compress.Decode([]byte(doc.Content))
		if err != nil {
			return nil, err
		}

		documents = append(documents, &v1.PublishedDocument{
			Id:      doc.ID,
			Meta:    string(metaData),
			Version: doc.Version,
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

	var versions []*v1.DocVersion
	for _, meta := range metaList {
		versions = append(versions, &v1.DocVersion{
			Version:   meta.Version,
			CreatedAt: timestamppb.New(meta.CreatedAt),
		})
	}

	req := &v1.ListPublishedDocumentVersionsResponse{
		Id:       docID.String(),
		Versions: versions,
	}

	if len(versions) > 0 {
		latestMeta, err := p.store.GetLatestPublishedDocumentMeta(ctx, docID)
		if err != nil {
			return nil, err
		}
		req.LatestVersion = latestMeta.Version
	}

	return req, nil
}
