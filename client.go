package document

import (
	v1 "github.com/emrgen/document/apis/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
)

type Client interface {
	io.Closer
	v1.DocumentServiceClient
	v1.PublishedDocumentServiceClient
	v1.DocumentBackupServiceClient
}

type client struct {
	conn *grpc.ClientConn
	v1.DocumentServiceClient
	v1.PublishedDocumentServiceClient
	v1.DocumentBackupServiceClient
}

func NewClient(port string) (Client, error) {
	conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &client{
		conn:                           conn,
		DocumentServiceClient:          v1.NewDocumentServiceClient(conn),
		PublishedDocumentServiceClient: v1.NewPublishedDocumentServiceClient(conn),
		DocumentBackupServiceClient:    v1.NewDocumentBackupServiceClient(conn),
	}, nil
}

func (c *client) Close() error {
	return c.conn.Close()
}
