/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"toolBox/pkg/server"

	"github.com/spf13/cobra"
)

var version = "0.0.1"

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
		// Chemin du projet React
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %s", err)
		}

		// Construire le chemin vers le dossier React
		reactDir := filepath.Join(workingDir, "client")
		conf, err := server.LoadServerConfig("FRONT")
		if err != nil {
			fmt.Println("Erreur de chargement de la config:", err)
			return
		}
		listenURL := fmt.Sprintf("%s://%s:%d", conf.Protocol, conf.FQDN, conf.Port)
		listenSocket := fmt.Sprintf(":%d", conf.Port)
		// Servir les fichiers statiques
		buildDir := reactDir + "/dist"
		log.Printf("Serving React build from: %s", buildDir)
		http.Handle("/", http.FileServer(http.Dir(buildDir)))
		// Lancer le serveur
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
		// Chemin du projet React
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %s", err)
		}

		// Construire le chemin vers le dossier React
		reactDir := filepath.Join(workingDir, "client")

		// Exécuter la commande `npm run build`
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
