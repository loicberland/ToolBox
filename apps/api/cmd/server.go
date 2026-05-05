package cmd

import (
	"log"
	apihttp "toolBox/apps/api/internal/http"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "starting server",
	Long:  "Starts the API server.",
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func startServer() {
	if err := apihttp.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
