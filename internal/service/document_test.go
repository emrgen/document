package service

import (
	"context"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/tester"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test_CreateDocument tests the CreateDocument method
func Test_CreateDocument(t *testing.T) {
	defer tester.CleanUp()

	docService := NewDocumentService(tester.TestDB(), nil)

	userCreate, err := userService.CreateUser(context.TODO(), &v1.CreateUserRequest{
		Name:  "sm",
		Email: "sm@mail.com",
	})
	assert.NoError(t, err)

	projectCreate, err := projectService.CreateProject(context.TODO(), &v1.CreateProjectRequest{
		Name:   "new project",
		UserId: userCreate.User.Id,
	})
	assert.NoError(t, err)

	docCreate, err := docService.CreateDocument(context.TODO(), &v1.CreateDocumentRequest{
		Id:        nil,
		Title:     "new document",
		Content:   "",
		ProjectId: projectCreate.Project.Id,
		UserId:    userCreate.User.Id,
	})
	assert.NoError(t, err)

	assert.Equal(t, "new document", docCreate.Document.Title)
}

func Test_UpdateDocument(t *testing.T) {
	defer tester.CleanUp()

	user, err := setupUser("sm", "sm@mail.com")
	assert.NoError(t, err)

	project, err := setupProject(user.Id, "new project")
	assert.NoError(t, err)

	doc, err := setupDocument(project.Id, user.Id, "new document")
	assert.NoError(t, err)

	docService := NewDocumentService(tester.TestDB(), nil)
	title := "updated document"
	content := "updated content"
	docUpdate, err := docService.UpdateDocument(context.TODO(), &v1.UpdateDocumentRequest{
		Id:      doc.Id,
		Title:   &title,
		Content: &content,
	})
	assert.NoError(t, err)

	assert.Equal(t, "updated document", docUpdate.Document.Title)
	assert.Equal(t, "updated content", docUpdate.Document.Content)
}

func setupDocument(projectId, userId, title string) (*v1.Document, error) {
	docService := NewDocumentService(tester.TestDB())
	docCreate, err := docService.CreateDocument(context.TODO(), &v1.CreateDocumentRequest{
		Id:        nil,
		Title:     title,
		Content:   "",
		ProjectId: projectId,
		UserId:    userId,
	})
	if err != nil {
		panic(err)
	}

	return docCreate.Document, nil
}
