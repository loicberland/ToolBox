package cmd

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"toolBox/pkg/toolboxconfig"

	"github.com/spf13/cobra"
)

//go:embed dist/*
var embeddedDist embed.FS

var version = "1.0.0"
var distDirFlag string
var configPathFlag string
var addrFlag string
var apiTargetFlag string

var rootCmd = &cobra.Command{
	Use:     "web-server",
	Short:   "ToolBox web static server",
	Version: version,
}

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the web server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := toolboxconfig.Load(configPathFlag, toolboxconfig.Overrides{
			WebAddr:   addrFlag,
			APITarget: apiTargetFlag,
		})
		if err != nil {
			log.Fatal(err)
		}

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

		apiProxy, err := newAPIProxy(cfg.API.Target)
		if err != nil {
			log.Fatal(err)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/toolbox.config.js", toolboxConfigHandler)
		mux.Handle("/api/", apiProxy)
		mux.Handle("/", handler)

		fmt.Printf("Proxying /api/ to %s\n", cfg.API.Target)
		fmt.Printf("Starting web server at %s\n", cfg.Web.PublicURL)
		if err := http.ListenAndServe(cfg.Web.Addr, mux); err != nil {
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
	serverCmd.Flags().StringVar(&configPathFlag, "config", "", "path to toolbox.cfg")
	serverCmd.Flags().StringVar(&addrFlag, "addr", "", "web server listen address")
	serverCmd.Flags().StringVar(&apiTargetFlag, "api-target", "", "API target URL used by the /api reverse proxy")
}

func toolboxConfigHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(`window.TOOLBOX = {
  services: {
    api: {
      url: "/api"
    }
  }
};
`))
}

func newAPIProxy(apiTarget string) (http.Handler, error) {
	target, err := url.Parse(apiTarget)
	if err != nil {
		return nil, fmt.Errorf("parse API target %q: %w", apiTarget, err)
	}
	if target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("API target must include scheme and host: %q", apiTarget)
	}
	return httputil.NewSingleHostReverseProxy(target), nil
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
