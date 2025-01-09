package service

import (
	"context"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/store"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewDocumentBackupService creates a new document backup service
func NewDocumentBackupService(store store.Store) *DocumentBackupService {
	return &DocumentBackupService{
		store: store,
	}
}

var _ v1.DocumentBackupServiceServer = (*DocumentBackupService)(nil)

// DocumentBackupService implements v1.DocumentBackupServiceServer
type DocumentBackupService struct {
	store store.Store
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
	//TODO implement me
	panic("implement me")
}

func (d *DocumentBackupService) DeleteDocumentBackup(ctx context.Context, request *v1.DeleteDocumentBackupRequest) (*v1.DeleteDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DocumentBackupService) RestoreDocumentBackup(ctx context.Context, request *v1.RestoreDocumentBackupRequest) (*v1.RestoreDocumentBackupResponse, error) {
	//TODO implement me
	panic("implement me")
}
