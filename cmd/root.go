package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "doc",
	Short: "document management tool",
	Example: `doc create -p <project-id> -c <content>
doc get -d <doc-id>
doc list -p <project-id> -published -latest
doc update -d <doc-id> -c <content> -t <title>
doc publish -d <doc-id> -v <version>
doc unpublish -d <doc-id>
doc versions -d <doc-id>
doc delete -d <doc-id>`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(dbCmd)
	rootCmd.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	cobra.EnableCommandSorting = false
}
