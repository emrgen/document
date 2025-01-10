package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/emrgen/document"
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
	"strings"
)

func init() {
	rootCmd.AddCommand(createDocCmd())
	rootCmd.AddCommand(getDocCmd())
	rootCmd.AddCommand(listDocCmd())
	rootCmd.AddCommand(updateDocCmd())
	rootCmd.AddCommand(publishDocCmd())
	rootCmd.AddCommand(listDocVersionsCmd())

	rootCmd.AddCommand(linkCmd)
	linkCmd.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})
	linkCmd.AddCommand(addLinkCmd())
	linkCmd.AddCommand(removeLinkCmd())
	linkCmd.AddCommand(listLinksCmd())

	rootCmd.AddCommand(childCmd)
	childCmd.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})
	childCmd.AddCommand(addChildCmd())
	childCmd.AddCommand(listChildCmd())
	childCmd.AddCommand(removeChildCmd())

	rootCmd.AddCommand(publishedCmd)
	publishedCmd.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})
	publishedCmd.AddCommand(getPublishedDocCmd())
	publishedCmd.AddCommand(listPublishedDocsCmd())
	publishedCmd.AddCommand(listPublishedVersionsCmd())
	publishedCmd.AddCommand(listPublishedLinksCommand())
	publishedCmd.AddCommand(listPublishedChildrenCmd())
}

func createDocCmd() *cobra.Command {
	var projectID string
	var docID string
	var docTitle string
	var content string

	var required = []string{"project-id"}

	command := &cobra.Command{
		Use:     "create",
		Short:   "create a document",
		Long:    `create a document with the given name and content`,
		Example: "doc create -p <project-id> -d <doc_id> -t <title> -c <content>",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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

	command.Flags().StringVarP(&projectID, "project-id", "p", "", "project id (required)")
	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id")
	command.Flags().StringVarP(&docTitle, "title", "t", "", "title of the document")
	command.Flags().StringVarP(&content, "content", "c", "", "content of the document")

	command.Flags().SortFlags = false

	return command
}

func getDocCmd() *cobra.Command {
	var docID string
	var version int64

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:     "get",
		Short:   "get a document",
		Example: "doc get -d <doc-id> -v <version>",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()

			// return from backup if version is provided
			if version != -1 {
				client := v1.NewDocumentBackupServiceClient(conn)
				req := &v1.GetDocumentBackupRequest{
					DocumentId: docID,
					Version:    version,
				}

				res, err := client.GetDocumentBackup(tokenContext(), req)
				if err != nil {
					logrus.Error(err)
					return
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Version"})
				table.Append([]string{res.Document.Id, strconv.FormatInt(res.Document.Version, 10)})
				table.Render()

				return
			}

			client := v1.NewDocumentServiceClient(conn)

			req := &v1.GetDocumentRequest{
				DocumentId: docID,
			}

			res, err := client.GetDocument(tokenContext(), req)
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version", "Children", "Links"})
			var meta map[string]interface{}
			err = json.Unmarshal([]byte(res.Document.Meta), &meta)
			if err != nil {
				logrus.Error(err)
				return
			}

			doc := res.Document

			logrus.Infof("document: %s", doc.Links)

			table.Append([]string{doc.Id, strconv.FormatInt(doc.Version, 10), strconv.Itoa(len(doc.Children)), strconv.Itoa(len(doc.Links))})
			table.Render()
			printField("Title", getTitle(res.Document.Meta))
			printField("Content", res.Document.Content)
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id (required)")
	command.Flags().Int64VarP(&version, "version", "v", -1, "version of the document")

	command.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})
	command.Flags().SortFlags = false

	return command
}

func listDocCmd() *cobra.Command {
	var projectID string
	var published bool

	var required = []string{"project-id"}
	command := &cobra.Command{
		Use:   "list",
		Short: "list documents",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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
			table.SetHeader([]string{"ID", "Title", "Version", "Links", "Children"})
			for _, doc := range res.Documents {
				table.Append([]string{doc.Id, getTitle(doc.Meta), strconv.FormatInt(doc.Version, 10), strconv.Itoa(len(doc.Links)), strconv.Itoa(len(doc.Children))})
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&projectID, "project-id", "p", "", "project id (required)")
	command.Flags().BoolVarP(&published, "pub", "u", false, "list published documents")
	command.Flags().SortFlags = false

	return command
}

func updateDocCmd() *cobra.Command {
	var docID string
	var docTitle string
	var content string
	var version int64

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:   "update",
		Short: "update a document",
		Long: `Update a document with the given id.

Constraint:
 1. version is not provided => the document will be overwritten with the current version + 1.
 2. version provided => updates the document if (next version == current version + 1).
`,
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			if version == -1 {
				color.Magenta("overwriting document: %s\n", docID)
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

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id (required)")
	command.Flags().StringVarP(&docTitle, "title", "t", "", "title")
	command.Flags().StringVarP(&content, "content", "c", "", "content")
	command.Flags().Int64VarP(&version, "version", "v", -1, "next version")

	command.Flags().SortFlags = false

	return command
}

func publishDocCmd() *cobra.Command {
	var docID string
	var version string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:   "publish",
		Short: "publish a document",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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
	command.Flags().SortFlags = false

	return command
}

func listDocVersionsCmd() *cobra.Command {
	var docID string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:   "versions",
		Short: "list document versions",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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
			res, err := client.ListDocumentVersions(ctx, &v1.ListDocumentVersionsRequest{
				DocumentId: docID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Version", "Created At"})
			for _, v := range res.Versions {
				version := strconv.FormatInt(v.Version, 10)
				if v.Version == res.LatestVersion {
					table.Append([]string{version + " (current)", v.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				} else {
					table.Append([]string{fmt.Sprintf("%-11s", version), v.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
				}
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id to list versions")

	return command
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "manage links between documents",
	Example: `  doc link add -s <source-id> -t <target-id>
  doc link list -d <doc-id> --published --backlink
  doc link remove -s <source-id> -t <target-id>`,
}

func addLinkCmd() *cobra.Command {
	var sourceID string
	var targetID string
	var targetVersion string

	var required = []string{"source-id", "target-id"}

	command := &cobra.Command{
		Use:     "add",
		Short:   "add a link between two documents",
		Example: "doc link -s <source-id> -t <target-id> -v <target-version>",

		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			client, err := document.NewClient("4020")
			if err != nil {
				logrus.Error(err)
				return
			}
			defer client.Close()

			res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
				DocumentId: sourceID,
			})
			if err != nil {
				return
			}

			if res.Document.Links == nil {
				res.Document.Links = make(map[string]string)
			}

			// add link to the document
			res.Document.Links[fmt.Sprintf("%s@%s", targetID, targetVersion)] = ""

			_, err = client.UpdateDocument(tokenContext(), &v1.UpdateDocumentRequest{
				Id:      sourceID,
				Links:   res.Document.Links,
				Version: res.Document.Version + 1,
			})
			if err != nil {
				logrus.Error(err)
				return
			}
		},
	}

	command.Flags().StringVarP(&sourceID, "source-id", "s", "", "source document id (required)")
	command.Flags().StringVarP(&targetID, "target-id", "t", "", "target document id (required)")
	command.Flags().StringVarP(&targetVersion, "target-version", "v", "current", "target document version")

	command.Flags().SortFlags = false

	return command
}

func listLinksCmd() *cobra.Command {
	var docID string
	var version string
	var backlink bool

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:        "list",
		Short:      "list links related to a document",
		Example:    "doc link -d <doc-id> --published --backlink",
		SuggestFor: []string{"links"},
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			client, err := document.NewClient("4020")
			if err != nil {
				logrus.Error(err)
				return
			}
			defer client.Close()

			if !backlink {
				res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
					DocumentId: docID,
				})
				if err != nil {
					logrus.Error(err)
					return
				}

				logrus.Infof("links: %v", res.Document.Links)

				if res.Document.Links == nil || len(res.Document.Links) == 0 {
					logrus.Infof("no links found")
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Version"})
				for link, _ := range res.Document.Links {
					tokens := strings.Split(link, "@")
					if len(tokens) != 2 {
						logrus.Warnf("invalid link: %s, expected format: <id>@<version>", link)
						continue
					}

					table.Append([]string{tokens[0], tokens[1]})
				}

				table.Render()
				return
			}

			if backlink {
				res, err := client.ListBacklinks(tokenContext(), &v1.ListBacklinksRequest{
					DocumentId: docID,
				})
				if err != nil {
					logrus.Error(err)
					return
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Version"})
				for _, link := range res.Links {
					table.Append([]string{link.SourceId, link.SourceVersion})
				}

				table.Render()
				return
			}
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id of the document")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document")
	command.Flags().BoolVarP(&backlink, "backlink", "b", false, "backlink id")

	command.Flags().SortFlags = false

	return command
}

func removeLinkCmd() *cobra.Command {
	var sourceID string
	var targetID string

	var required = []string{"source-id", "target-id"}

	command := &cobra.Command{
		Use:     "remove",
		Short:   "remove a link between two documents",
		Example: "document unlink -s <source-id> -t <target-id>",

		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}
			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
				DocumentId: sourceID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			if res.Document.Links == nil {
				res.Document.Links = make(map[string]string)
			}
			delete(res.Document.Links, targetID)

			_, err = client.UpdateDocument(tokenContext(), &v1.UpdateDocumentRequest{
				Id:      sourceID,
				Links:   res.Document.Links,
				Version: res.Document.Version + 1,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			color.Green("link removed")
		},
	}

	command.Flags().StringVarP(&sourceID, "source-id", "s", "", "source document id (required)")
	command.Flags().StringVarP(&targetID, "target-id", "t", "", "target document id (required)")
	command.Flags().SortFlags = false
	return command
}

var childCmd = &cobra.Command{
	Use:   "child",
	Short: "manage child documents",
	Example: `  doc child add -s <source-id> -t <target-id>
  doc child list -d <doc-id> --published
  doc child remove -s <source-id> -t <target-id>`,
}

func addChildCmd() *cobra.Command {
	var parentID string
	var childID string
	var childVersion string

	var required = []string{"parent-id", "child-id"}

	command := &cobra.Command{
		Use:     "add",
		Short:   "add a child document",
		Example: "doc child add -p <parent-id> -c <child-id> -v <child-version>",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
				DocumentId: parentID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			if res.Document.Children == nil {
				res.Document.Children = make([]string, 0)
			}

			// add link to the document
			res.Document.Children = append(res.Document.Children, fmt.Sprintf("%s@%s", childID, childVersion))
			// remove duplicates
			res.Document.Children = unique(res.Document.Children)

			_, err = client.UpdateDocument(tokenContext(), &v1.UpdateDocumentRequest{
				Id:       parentID,
				Children: res.Document.Children,
				Version:  res.Document.Version + 1,
			})
			if err != nil {
				logrus.Error(err)
				return
			}
		},
	}

	command.Flags().StringVarP(&parentID, "parent-id", "p", "", "source document id (required)")
	command.Flags().StringVarP(&childID, "child-id", "c", "", "target document id (required)")
	command.Flags().StringVarP(&childVersion, "child-version", "v", "current", "child document version")
	command.Flags().SortFlags = false

	return command
}

func listChildCmd() *cobra.Command {
	var docID string
	var version string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:     "list",
		Short:   "list child documents",
		Example: `  doc child list -d <doc-id> --published`,
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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
			res, err := client.GetDocument(ctx, &v1.GetDocumentRequest{
				DocumentId: docID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version"})
			for _, child := range res.Document.Children {
				tokens := strings.Split(child, "@")
				if len(tokens) != 2 {
					logrus.Warnf("invalid child link: %s, expected format: <id>@<version>", child)
					continue
				}
				table.Append([]string{tokens[0], tokens[1]})
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id of the document")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document")

	command.Flags().SortFlags = false

	return command
}

func removeChildCmd() *cobra.Command {
	var sourceID string
	var targetID string

	var required = []string{"source-id", "target-id"}

	command := &cobra.Command{
		Use:     "remove",
		Short:   "remove a child document",
		Example: "document child remove -s <source-id> -t <target-id>",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewDocumentServiceClient(conn)

			res, err := client.GetDocument(tokenContext(), &v1.GetDocumentRequest{
				DocumentId: sourceID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			if res.Document.Children == nil {
				res.Document.Children = make([]string, 0)
			}

			// remove link from the document
			for i, child := range res.Document.Children {
				if child == targetID {
					res.Document.Children = append(res.Document.Children[:i], res.Document.Children[i+1:]...)
				}
			}

			_, err = client.UpdateDocument(tokenContext(), &v1.UpdateDocumentRequest{
				Id:       sourceID,
				Children: res.Document.Children,
				Version:  res.Document.Version + 1,
			})
			if err != nil {
				logrus.Error(err)
				return
			}
		},
	}

	command.Flags().StringVarP(&sourceID, "source-id", "s", "", "source document id (required)")
	command.Flags().StringVarP(&targetID, "target-id", "t", "", "target document id (required)")
	command.Flags().SortFlags = false

	return command
}

var publishedCmd = &cobra.Command{
	Use:   "pub",
	Short: "manage published documents",
	Example: `  doc pub get -d <doc-id> -v <version>
  doc pub list -p <project-id>
  doc pub versions -d <doc-id>`,
}

func getPublishedDocCmd() *cobra.Command {
	var docID string
	var version string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:     "get",
		Short:   "get a document",
		Example: "doc get -d <doc-id> -v <version>",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			docVersion := "latest"
			if version != "" {
				// check if valid semver
				_, err := semver.NewVersion(version)
				if err != nil {
					logrus.Error(err)
					return
				}
				docVersion = version
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
			table.SetHeader([]string{"ID", "Version", "Latest Version", "Last Published"})

			table.Append([]string{res.Document.Id, res.Document.Version, res.LatestVersion.Version, res.LatestVersion.CreatedAt.AsTime().Format("2006-01-02 15:04:05")})
			table.Render()

			printField("Title", getTitle(res.Document.Meta))
			printField("Content", res.Document.Content)
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id (required)")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document")

	return command
}

func listPublishedDocsCmd() *cobra.Command {
	var projectID string

	var required = []string{"project-id"}

	command := &cobra.Command{
		Use:   "list",
		Short: "list published documents",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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
			res, err := client.ListPublishedDocuments(ctx, &v1.ListPublishedDocumentsRequest{
				ProjectId: projectID,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version"})
			for _, doc := range res.Documents {
				table.Append([]string{doc.Id, doc.Version})
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&projectID, "project-id", "p", "", "project id (required)")

	return command
}

func listPublishedVersionsCmd() *cobra.Command {
	var docID string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:   "versions",
		Short: "list published document versions",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
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

func listPublishedLinksCommand() *cobra.Command {
	var docID string
	var backlink bool
	var version string

	command := &cobra.Command{
		Use:   "links",
		Short: "list published links",
		Run: func(cmd *cobra.Command, args []string) {
			if !backlink {
				conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					logrus.Error(err)
					return
				}
				defer conn.Close()
				client := v1.NewPublishedDocumentServiceClient(conn)

				docVersion := "latest"
				if version != "" {
					docVersion = version
				}

				res, err := client.GetPublishedDocument(tokenContext(), &v1.GetPublishedDocumentRequest{
					Id:      docID,
					Version: docVersion,
				})
				if err != nil {
					logrus.Error(err)
					return
				}

				if res.Document.Links == nil || len(res.Document.Links) == 0 {
					logrus.Infof("no links found")
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Version"})
				for link, _ := range res.Document.Links {
					tokens := strings.Split(link, "@")
					if len(tokens) != 2 {
						logrus.Warnf("invalid link: %s, expected format: <id>@<version>", link)
						continue
					}
					table.Append([]string{tokens[0], tokens[1]})
				}
				table.Render()
				return
			}

			if backlink {
				conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					logrus.Error(err)
					return
				}
				defer conn.Close()
				client := v1.NewPublishedDocumentServiceClient(conn)

				docVersion := "latest"
				if version != "" {
					docVersion = version
				}

				ctx := tokenContext()
				res, err := client.ListPublishedBacklinks(ctx, &v1.ListPublishedBacklinksRequest{
					DocumentId: docID,
					Version:    docVersion,
				})
				if err != nil {
					logrus.Error(err)
					return
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Version"})
				for _, link := range res.Links {
					table.Append([]string{link.SourceId, link.SourceVersion})
				}

				table.Render()
				return
			}

		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id of the document")
	command.Flags().BoolVarP(&backlink, "backlink", "b", false, "backlink")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document")

	return command
}

func listPublishedChildrenCmd() *cobra.Command {
	var docID string
	var version string

	var required = []string{"doc-id"}

	command := &cobra.Command{
		Use:   "children",
		Short: "list published children",
		Run: func(cmd *cobra.Command, args []string) {
			if checkMissingFlags(cmd, required) {
				return
			}

			conn, err := grpc.NewClient(":4020", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				logrus.Error(err)
				return
			}
			defer conn.Close()
			client := v1.NewPublishedDocumentServiceClient(conn)

			docVersion := "latest"
			if version != "" {
				if !checkValidSemvar(version) {
					return
				}
				docVersion = version
			}

			ctx := tokenContext()
			res, err := client.GetPublishedDocumentMeta(ctx, &v1.GetPublishedDocumentMetaRequest{
				DocumentId: docID,
				Version:    docVersion,
			})
			if err != nil {
				logrus.Error(err)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Version"})
			for _, child := range res.Document.Children {
				tokens := strings.Split(child, "@")
				if len(tokens) != 2 {
					logrus.Warnf("invalid child link: %s, expected format: <id>@<version>", child)
					continue
				}
				table.Append([]string{tokens[0], tokens[1]})
			}

			table.Render()
		},
	}

	command.Flags().StringVarP(&docID, "doc-id", "d", "", "document id of the document")
	command.Flags().StringVarP(&version, "version", "v", "", "version of the document")

	return command
}

func parseMap(meta string) map[string]interface{} {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(meta), &m)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	return m
}

func getTitle(meta string) string {
	m := parseMap(meta)
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

// checkMissingFlags checks if the required flags are set and returns ok if they are set
func checkMissingFlags(cmd *cobra.Command, flags []string) bool {
	var missingFlags []string
	var providedFlags []string
	for _, required := range flags {
		if cmd.Flag(required).Changed == false {
			missingFlags = append(missingFlags, required)
		} else {
			value := cmd.Flag(required).Value.String()
			providedFlags = append(providedFlags, fmt.Sprintf("--%s=%s", required, value))
		}
	}

	if len(missingFlags) > 0 {
		var msg string
		for _, f := range missingFlags {
			msg += fmt.Sprintf("--%s ", f)
		}

		color.Red("missing: %s\n", msg)
		if len(providedFlags) > 0 {
			provided := strings.Join(providedFlags, " ")
			color.Green("provide: %s\n", provided)
		}

		cmd.Println("")

		cmd.Usage()

		return true
	}

	return false
}

func checkValidSemvar(ver string) bool {
	_, err := semver.NewVersion(ver)
	return err == nil
}

// unique returns a slice with unique elements
func unique(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
