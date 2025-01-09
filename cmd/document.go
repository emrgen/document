package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"strconv"
)

var documentCmd = &cobra.Command{
	Use:   "doc",
	Short: "document management",
}

func init() {
	rootCmd.AddCommand(createDocCmd())
	rootCmd.AddCommand(getDocCmd())
	rootCmd.AddCommand(listDocCmd())
	rootCmd.AddCommand(updateDocCmd())
	rootCmd.AddCommand(publishDocCmd())
	rootCmd.AddCommand(listDocVersionsCmd())
}

func createDocCmd() *cobra.Command {
	var projectID string
	var docID string
	var docTitle string
	var content string

	command := &cobra.Command{
		Use:   "create",
		Short: "create a document",
		Long:  `create a document with the given name and content`,
		Run: func(cmd *cobra.Command, args []string) {
			if projectID == "" {
				logrus.Errorf("missing required flag: --project-id")
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			req := &v1.CreateDocumentRequest{
				ProjectId: projectID,
				Content:   content,
			}
			if docTitle != "" {
				meta := map[string]string{
					"title": docTitle,
				}

				metaData, err := json.Marshal(meta)
				if err != nil {
					logrus.Error(err)
					return
				}
				req.Meta = string(metaData)
			}

			if docID != "" {
				_, err2 := uuid.Parse(docID)
				if err2 != nil {
					logrus.Error("invalid document id, expected a valid uuid")
					return
				}
				req.DocumentId = &docID
			}

			ctx := tokenContext()
			res, err := client.CreateDocument(ctx, req)
			if err != nil {
				logrus.Error(err)
				return
			}

			logrus.Infof("document created with id: %s", res.Document.Id)
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id")
	command.Flags().StringVarP(&projectID, "project", "p", "", "project id to create the document in")
	command.Flags().StringVarP(&docTitle, "title", "t", "", "title of the document")
	command.Flags().StringVarP(&content, "content", "c", "", "content of the document")

	return command
}

func getDocCmd() *cobra.Command {
	var docID string
	var latest bool
	var version string

	command := &cobra.Command{
		Use:   "get",
		Short: "get a document",
		Run: func(cmd *cobra.Command, args []string) {
			if docID == "" {
				logrus.Errorf("missing required flag: --doc-id")
				return
			}

			if latest || version != "" {
				docVersion := version
				if version != "" && version != "latest" {
					// check if valid semver
					_, err := semver.NewVersion(version)
					if err != nil {
						logrus.Error(err)
						return
					}
				}

				conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					logrus.Error(err)
					return
				}
				defer conn.Close()
				client := v1.NewPublishedDocumentServiceClient(conn)
				if err != nil {
					logrus.Error(err)
					return
				}

				res, err := client.GetPublishedDocument(tokenContext(), &v1.GetPublishedDocumentRequest{
					Id:      docID,
					Version: docVersion,
				})
				if err != nil {
					return
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Title", "Version", "Latest"})
				var meta map[string]interface{}
				err = json.Unmarshal([]byte(res.Document.Meta), &meta)
				if err != nil {
					logrus.Error(err)
					return
				}

				title, ok := meta["title"].(string)
				if !ok {
					title = ""
				}

				table.Append([]string{res.Document.Id, title, res.Document.Version, "true"})

				table.Render()
			}

			if version != "" {

			}

			if !latest && version == "" {
				conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					logrus.Error(err)
					return
				}
				defer conn.Close()
				client := v1.NewDocumentServiceClient(conn)

				res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
					Id: docID,
				})
				if err != nil {
					logrus.Error(err)
					return
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Created At"})
				var meta map[string]interface{}
				err = json.Unmarshal([]byte(res.Document.Meta), &meta)
				if err != nil {
					logrus.Error(err)
					return
				}

				title, ok := meta["title"].(string)
				if !ok {
					title = ""
				}
				table.Append([]string{res.Document.Id, res.Document.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				table.Render()
				fmt.Printf("Title: %s\n", title)
				fmt.Printf("Content: %s\n", res.Document.Content)
			}
		},
	}

	command.Flags().StringVarP(&version, "version", "v", "", "version of the document to get")
	command.Flags().BoolVarP(&latest, "latest", "l", false, "get the latest version of the document")
	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id to get")

	return command
}

func listDocCmd() *cobra.Command {
	var projectID string

	command := &cobra.Command{
		Use:   "list",
		Short: "list documents",
		Run: func(cmd *cobra.Command, args []string) {
			if projectID == "" {
				logrus.Errorf("missing required flag: --project-id")
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
			res, err := client.ListDocuments(ctx, &v1.ListDocumentsRequest{
				ProjectId: projectID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Meta", "Version", "CreatedAt"})
			for _, doc := range res.Documents {
				table.Append([]string{doc.Id, doc.Meta, strconv.FormatInt(doc.Version, 10), doc.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
			}

			table.Render()

		},
	}

	command.Flags().StringVarP(&projectID, "project", "p", "", "project id to list documents")

	return command
}

func updateDocCmd() *cobra.Command {
	var docID string
	var docTitle string
	var content string
	var version int64

	command := &cobra.Command{
		Use:   "update",
		Short: "update a document",
		Run: func(cmd *cobra.Command, args []string) {
			if docID == "" {
				logrus.Errorf("missing required flag: --doc-id")
				return
			}

			if version == -1 {
				cmd.Printf("using update version %d\n", version)
				cmd.Printf("overwriting document %s\n", docID)
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			req := &v1.UpdateDocumentRequest{
				Id:      docID,
				Version: version,
				Kind:    v1.UpdateKind_TEXT,
			}

			// update content if provided
			if content != "" {
				req.Content = &content
			}

			// update meta if title is provided
			if docTitle != "" {
				meta := map[string]string{
					"title": docTitle,
				}

				metaData, err := json.Marshal(meta)
				if err != nil {
					logrus.Error(err)
					return
				}
				data := string(metaData)
				req.Meta = &data
			}

			_, err = client.UpdateDocument(tokenContext(), req)
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Title", "Version"})
			table.Append([]string{docID, docTitle, strconv.FormatInt(version, 10)})
			table.Render()
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id to update")
	command.Flags().StringVarP(&docTitle, "title", "t", "", "title of the document")
	command.Flags().StringVarP(&content, "content", "c", "", "content of the document")
	command.Flags().Int64VarP(&version, "version", "v", -1, "version of the document to update")

	return command
}

func publishDocCmd() *cobra.Command {
	var docID string
	var version string

	command := &cobra.Command{
		Use:   "publish",
		Short: "publish a document",
		Run: func(cmd *cobra.Command, args []string) {
			if docID == "" {
				logrus.Errorf("missing required flag: --doc-id")
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			req := &v1.PublishDocumentRequest{
				DocumentId: docID,
			}

			if version != "" {
				_, err := semver.NewVersion(version)
				if err != nil {
					logrus.Error(err)
					return
				}
				req.Version = &version
			}

			_, err = client.PublishDocument(tokenContext(), req)
			if err != nil {
				logrus.Error(err)
				return
			}

			logrus.Infof("document %s published", docID)
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id to publish")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document to publish")

	return command
}

func listDocVersionsCmd() *cobra.Command {
	var docID string

	command := &cobra.Command{
		Use:   "versions",
		Short: "list document versions",
		Run: func(cmd *cobra.Command, args []string) {
			if docID == "" {
				logrus.Errorf("missing required flag: --doc-id")
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewPublishedDocumentServiceClient(conn)

			ctx := tokenContext()
			res, err := client.ListPublishedDocumentVersions(ctx, &v1.ListPublishedDocumentVersionsRequest{
				Id: docID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Version", "Created At"})
			for _, v := range res.Versions {
				if v.Version == res.LatestVersion {
					table.Append([]string{v.Version + " (latest)", v.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				} else {
					table.Append([]string{v.Version, v.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				}
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id to list versions")

	return command
}
