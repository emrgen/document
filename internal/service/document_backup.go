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
func NewDocumentBackupService(compress compress.Compress, store store.Store) *DocumentBackupService {
	return &DocumentBackupService{
		store:    store,
		compress: compress,
	}
}

var _ v1.DocumentBackupServiceServer = (*DocumentBackupService)(nil)

// DocumentBackupService implements v1.DocumentBackupServiceServer
type DocumentBackupService struct {
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
			Id: backup.ID,
			Document: &v1.Document{
				Id:        backup.DocumentID,
				Content:   backup.Content,
				Version:   backup.Version,
				CreatedAt: timestamppb.New(backup.CreatedAt),
			},
		})
	}

	return &resp, nil
}

func (d *DocumentBackupService) CreateDocumentBackup(ctx context.Context, request *v1.CreateDocumentBackupRequest) (*v1.CreateDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}

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

func (d *DocumentBackupService) DeleteDocumentBackup(ctx context.Context, request *v1.DeleteDocumentBackupRequest) (*v1.DeleteDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DocumentBackupService) RestoreDocumentBackup(ctx context.Context, request *v1.RestoreDocumentBackupRequest) (*v1.RestoreDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}
