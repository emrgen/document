package service

import (
	"context"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/store"
	"github.com/emrgen/document/internal/tester"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDocumentService_CreateDocument(t *testing.T) {
	tester.RemoveDBFile()
	tester.Setup()

	client := NewDocumentService(compress.NewNop(), store.NewGormStore(tester.TestDB()), tester.Redis())
	tests := []struct {
		name      string
		projectID string
		docID     string
		Meta      string
		Content   string
		Links     map[string]string
		Children  []string
		service   *v1.DocumentServiceServer
	}{
		{
			name:      "Test CreateDocument",
			projectID: uuid.New().String(),
			docID:     uuid.New().String(),
		},
		{
			name:      "Test CreateDocument",
			projectID: uuid.New().String(),
			docID:     uuid.New().String(),
			Meta:      "{}",
			Content:   "content",
			Links:     map[string]string{"link": "link"},
			Children:  []string{"child"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.CreateDocument(context.TODO(), &v1.CreateDocumentRequest{
				ProjectId:  tt.projectID,
				DocumentId: &tt.docID,
				Meta:       tt.Meta,
				Content:    tt.Content,
				Links:      tt.Links,
				Children:   tt.Children,
			})
			assert.NoError(t, err)
			assert.NotNil(t, res)

			assert.Equal(t, tt.docID, res.Document.Id)
		})

		got, err := client.GetDocument(context.TODO(), &v1.GetDocumentRequest{
			DocumentId: tt.docID,
		})
		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.Equal(t, tt.docID, got.Document.Id)
		assert.Equal(t, tt.Content, got.Document.Content)

		if tt.Meta != "" {
			assert.Equal(t, tt.Meta, got.Document.Meta)
		} else {
			assert.Equal(t, "{}", got.Document.Meta)
		}

		if tt.Links != nil {
			assert.Equal(t, tt.Links, got.Document.Links)
		} else {
			assert.Equal(t, map[string]string{}, got.Document.Links)
		}

		if tt.Children != nil {
			assert.Equal(t, tt.Children, got.Document.Children)
		} else {
			assert.Equal(t, []string{}, got.Document.Children)
		}
	}
}

func TestDocumentService_UpdateDocument(t *testing.T) {
	tester.RemoveDBFile()
	tester.Setup()

	client := NewDocumentService(compress.NewNop(), store.NewGormStore(tester.TestDB()), tester.Redis())

	type Document struct {
		name      string
		projectID string
		docID     string
		Meta      string
		Content   string
		Links     map[string]string
		Children  []string
		Version   int64
	}

	projectID := uuid.New().String()
	docID := uuid.New().String()

	tests := []struct {
		Input  Document
		Update Document
		Output Document
	}{
		{
			Input: Document{
				name:      "Test Document",
				projectID: projectID,
				docID:     docID,
			},
			Update: Document{
				name:      "Test UpdatedDocument",
				projectID: projectID,
				docID:     docID,
				Meta:      "{}",
				Content:   "content",
				Links:     map[string]string{"link@latest": "link"},
				Version:   1,
			},
			Output: Document{
				name:      "Test UpdatedDocument",
				projectID: projectID,
				docID:     docID,
				Meta:      "{}",
				Content:   "content",
				Links:     map[string]string{"link@latest": "link"},
			},
		},
	}

	for _, tt := range tests {
		// create document
		_, err := client.CreateDocument(context.TODO(), &v1.CreateDocumentRequest{
			ProjectId:  tt.Input.projectID,
			DocumentId: &tt.Input.docID,
			Meta:       tt.Input.Meta,
			Content:    tt.Input.Content,
			Links:      tt.Input.Links,
			Children:   tt.Input.Children,
		})
		assert.NoError(t, err)

		// update document
		res, err := client.UpdateDocument(context.TODO(), &v1.UpdateDocumentRequest{
			DocumentId: tt.Update.docID,
			Meta:       &tt.Update.Meta,
			Content:    &tt.Update.Content,
			Links:      tt.Update.Links,
			Children:   tt.Update.Children,
			Version:    tt.Update.Version,
		})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, tt.Output.docID, res.Id)

		// get document
		got, err := client.GetDocument(context.TODO(), &v1.GetDocumentRequest{
			DocumentId: tt.Output.docID,
		})
		assert.NoError(t, err)

		assert.Equal(t, tt.Output.docID, got.Document.Id)
		assert.Equal(t, tt.Output.Content, got.Document.Content)

		if tt.Output.Meta != "" {
			assert.Equal(t, tt.Output.Meta, got.Document.Meta)
		} else {
			assert.Equal(t, "{}", got.Document.Meta)
		}

		if tt.Output.Links != nil {
			assert.Equal(t, tt.Output.Links, got.Document.Links)
		} else {
			assert.Equal(t, map[string]string{}, got.Document.Links)
		}

		if tt.Output.Children != nil {
			assert.Equal(t, tt.Output.Children, got.Document.Children)
		} else {
			assert.Equal(t, []string{}, got.Document.Children)
		}
	}
}
