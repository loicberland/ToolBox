package cmd

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed dist/*
var embeddedDist embed.FS

var version = "1.0.0"
var distDirFlag string

var rootCmd = &cobra.Command{
	Use:     "web-server",
	Short:   "ToolBox web static server",
	Version: version,
}

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the web server",
	Run: func(cmd *cobra.Command, args []string) {
		listenURL := "http://localhost:20251"
		listenSocket := ":20251"
		var handler http.Handler

		if distDirFlag != "" {
			distDir, err := filepath.Abs(distDirFlag)
			if err != nil {
				log.Fatal(err)
			}
			handler = http.FileServer(http.Dir(distDir))
			fmt.Printf("Serving %s\n", distDir)
		} else {
			distFS, err := fs.Sub(embeddedDist, "dist")
			if err != nil {
				log.Fatalf("embedded web dist not available: %s", err)
			}
			handler = http.FileServer(http.FS(distFS))
			fmt.Println("Serving embedded web dist")
		}

		http.Handle("/", handler)
		fmt.Printf("Starting web server at %s\n", listenURL)
		if err := http.ListenAndServe(listenSocket, nil); err != nil {
			log.Fatalf("failed to start web server: %s", err)
		}
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the React web app",
	Run: func(cmd *cobra.Command, args []string) {
		build := exec.Command("npm", "run", "build")
		build.Dir = filepath.Join("apps", "web")
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			log.Fatalf("failed to build React project: %s", err)
		}
		if err := syncDist(filepath.Join("apps", "web", "dist"), filepath.Join("apps", "web-server", "cmd", "dist")); err != nil {
			log.Fatalf("failed to sync web dist: %s", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(buildCmd)
	serverCmd.Flags().StringVar(&distDirFlag, "dist", "", "serve a dist directory from disk instead of the embedded build")
}

func syncDist(sourceDir, targetDir string) error {
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, relativePath)

		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(sourcePath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	return err
}
