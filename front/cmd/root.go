package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed dist/*
var embeddedFiles embed.FS

var version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:     "front",
	Short:   "front toolbox server",
	Long:    ``,
	Version: version,
}

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "starting server",
	Long:  "Starts the Front server.",
	Run: func(cmd *cobra.Command, args []string) {
		listenURL := fmt.Sprintf("http://localhost:20251")
		listenSocket := fmt.Sprintf(":20251")
		// Serve static files
		distFS, _ := fs.Sub(embeddedFiles, "dist")
		http.Handle("/", http.FileServer(http.FS(distFS)))
		// Start server
		fmt.Printf("Starting server at %s\n", listenURL)
		if err := http.ListenAndServe(listenSocket, nil); err != nil {
			log.Fatalf("Failed to start server: %s", err)
		}
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build server",
	Long:  "Build the Front server.",
	Run: func(cmd *cobra.Command, args []string) {
		// React projet path
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %s", err)
		}
		reactDir := filepath.Join(workingDir, "client")

		// Get npm build command `npm run build`
		buildCmd := exec.Command("npm", "run", "build")
		buildCmd.Dir = reactDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			log.Fatalf("Failed to build React project: %s", err)
		}
	},
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

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(buildCmd)
}
