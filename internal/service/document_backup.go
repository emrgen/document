package service

import (
	"context"
	"encoding/json"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewDocumentBackupService creates a new document backup service
func NewDocumentBackupService(compress compress.Compress, store store.Store, docs v1.DocumentServiceServer) *DocumentBackupService {
	return &DocumentBackupService{
		docs:     docs,
		store:    store,
		compress: compress,
	}
}

var _ v1.DocumentBackupServiceServer = (*DocumentBackupService)(nil)

// DocumentBackupService implements v1.DocumentBackupServiceServer
type DocumentBackupService struct {
	docs     v1.DocumentServiceServer
	store    store.Store
	compress compress.Compress
	v1.UnimplementedDocumentBackupServiceServer
}

// ListDocumentBackups lists all document backups
func (d *DocumentBackupService) ListDocumentBackups(ctx context.Context, request *v1.ListDocumentBackupsRequest) (*v1.ListDocumentBackupsResponse, error) {
	docID := uuid.MustParse(request.GetDocumentId())
	backups, err := d.store.ListDocumentBackups(ctx, docID)
	if err != nil {
		return nil, err
	}

	var resp = v1.ListDocumentBackupsResponse{
		Backups: make([]*v1.DocumentBackup, 0, len(backups)),
	}
	for _, backup := range backups {
		resp.Backups = append(resp.Backups, &v1.DocumentBackup{
			Document: &v1.Document{
				Id:        backup.ID,
				Content:   backup.Content,
				Version:   backup.Version,
				CreatedAt: timestamppb.New(backup.CreatedAt),
			},
		})
	}

	return &resp, nil
}

// CreateDocumentBackup creates a document backup for a document at a specific version, this will overwrite any existing backup for the same version
func (d *DocumentBackupService) CreateDocumentBackup(ctx context.Context, request *v1.CreateDocumentBackupRequest) (*v1.CreateDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}

// GetDocumentBackup gets a document backup by id and lamport version
func (d *DocumentBackupService) GetDocumentBackup(ctx context.Context, request *v1.GetDocumentBackupRequest) (*v1.GetDocumentBackupResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	backup, err := d.store.GetDocumentBackup(ctx, docID, request.GetVersion())
	if err != nil {
		return nil, err
	}

	meta, err := d.compress.Decode([]byte(backup.Meta))
	if err != nil {
		return nil, err
	}

	links, err := d.compress.Decode([]byte(backup.Links))
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		links = []byte("{}")
	}

	linkMap := make(map[string]string)
	err = json.Unmarshal(links, &linkMap)
	if err != nil {
		return nil, err
	}

	childrenData, err := parseChildren(backup.Children)

	logrus.Infof("backup: %v", backup.Content)

	return &v1.GetDocumentBackupResponse{
		Document: &v1.Document{
			Id:        backup.ID,
			Content:   backup.Content,
			Version:   backup.Version,
			Meta:      string(meta),
			Links:     linkMap,
			Children:  childrenData,
			CreatedAt: timestamppb.New(backup.CreatedAt),
		},
	}, nil
}

// DeleteDocumentBackup deletes a document backup, this is a soft delete and the backup will still be available for 30 days after deletion
func (d *DocumentBackupService) DeleteDocumentBackup(ctx context.Context, request *v1.DeleteDocumentBackupRequest) (*v1.DeleteDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}

// RestoreDocumentBackup restores a document backup, overwriting the current document
func (d *DocumentBackupService) RestoreDocumentBackup(ctx context.Context, request *v1.RestoreDocumentBackupRequest) (*v1.RestoreDocumentBackupResponse, error) {
	docID, err := uuid.Parse(request.GetDocumentId())
	if err != nil {
		return nil, err
	}

	backup, err := d.store.GetDocumentBackup(ctx, docID, request.GetVersion())
	if err != nil {
		return nil, err
	}

	contentData, err := d.compress.Decode([]byte(backup.Content))
	if err != nil {
		return nil, err
	}
	contentStr := string(contentData)

	childrenData, err := d.compress.Decode([]byte(backup.Children))
	if err != nil {
		return nil, err
	}
	var children []string
	err = json.Unmarshal(childrenData, &children)
	if err != nil {
		return nil, err
	}

	metaData, err := d.compress.Decode([]byte(backup.Meta))
	if err != nil {
		return nil, err
	}
	metaStr := string(metaData)

	linksData, err := d.compress.Decode([]byte(backup.Links))
	if err != nil {
		return nil, err
	}
	if len(linksData) == 0 {
		linksData = []byte("{}")
	}
	var linkMap map[string]string
	err = json.Unmarshal(linksData, &linkMap)
	if err != nil {
		return nil, err
	}

	// use document service to restore by overwriting the document
	_, err = d.docs.UpdateDocument(ctx, &v1.UpdateDocumentRequest{
		DocumentId: docID.String(),
		Version:    -1, // force update
		Meta:       &metaStr,
		Content:    &contentStr,
		Links:      linkMap,
		Children:   children,
	})

	if err != nil {
		return nil, err
	}

	return &v1.RestoreDocumentBackupResponse{
		Document: &v1.Document{
			Id:       backup.ID,
			Version:  backup.Version,
			Content:  backup.Content,
			Meta:     metaStr,
			Links:    linkMap,
			Children: children,
		},
	}, nil
}
