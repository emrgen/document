package cmd

import (
	"fmt"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var documentCmd = &cobra.Command{
	Use:   "doc",
	Short: "document management",
}

func init() {
	documentCmd.AddCommand(createDocCmd())
}

func createDocCmd() *cobra.Command {
	var projectID string
	var docTitle string

	command := &cobra.Command{
		Use:   "create",
		Short: "create a document",
		Long:  `create a document with the given name and content`,
		Run: func(cmd *cobra.Command, args []string) {
			if projectID == "" {
				fmt.Println("project id is required")
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			ctx := tokenContext()
			res, err := client.CreateDocument(ctx, &v1.CreateDocumentRequest{
				Title:     docTitle,
				ProjectId: projectID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			logrus.Infof("document created with id: %s", res.Document.Id)
		},
	}

	command.Flags().StringVarP(&projectID, "project", "p", "", "project id to create the document in")
	command.Flags().StringVarP(&docTitle, "title", "t", "", "title of the document")

	return command
}
