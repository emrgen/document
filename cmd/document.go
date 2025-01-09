package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"strconv"
)

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
				table.SetHeader([]string{"ID", "Version", "Latest"})

				table.Append([]string{res.Document.Id, res.Document.Version, "true"})
				table.Render()

				cmd.Printf("Title: %s\n", getTitle(res.Document.Meta))
				cmd.Printf("Content: %s\n", res.Document.Content)
				color.Cyan("Title")
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

				table.Append([]string{res.Document.Id, res.Document.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				table.Render()
				printField("Title", getTitle(res.Document.Meta))
				printField("Content", res.Document.Content)
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
			table.SetHeader([]string{"ID", "Title", "Version", "CreatedAt"})
			for _, doc := range res.Documents {
				table.Append([]string{doc.Id, getTitle(doc.Meta), strconv.FormatInt(doc.Version, 10), doc.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
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

			res, err := client.UpdateDocument(tokenContext(), req)
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version"})
			table.Append([]string{docID, strconv.FormatInt(int64(res.Version), 10)})
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

			res, err := client.PublishDocument(tokenContext(), req)
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version"})
			table.Append([]string{res.Document.Id, res.Document.Version})
			table.Render()
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

func parseMeta(meta string) map[string]interface{} {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(meta), &m)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	return m
}

func getTitle(meta string) string {
	m := parseMeta(meta)
	title, ok := m["title"].(string)
	if !ok {
		return ""
	}

	return title
}

func printField(label, value string) {
	color.Set(color.FgCyan)
	fmt.Print(label)
	color.Unset()
	fmt.Printf(": %s\n", value)
}
