package cmd

import (
	"log"

	apihttp "toolBox/apps/api/internal/http"

	"github.com/spf13/cobra"
)

var configPathFlag string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "starting server",
	Long:  "Starts the API server.",
	Run: func(cmd *cobra.Command, args []string) {
		startServer(configPathFlag)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&configPathFlag, "config", "", "path to toolbox.cfg")
}

func startServer(configPath string) {
	if err := apihttp.ListenAndServe(configPath); err != nil {
		log.Fatal(err)
	}
}
