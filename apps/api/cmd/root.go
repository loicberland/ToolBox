package cmd

import (
	"os"

	"toolBox/pkg/toolboxversion"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "api",
	Short:   "api toolbox",
	Long:    ``,
	Version: toolboxversion.APIVersion,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	//disable help command
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	//disable completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
