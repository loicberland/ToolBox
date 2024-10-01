/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net/http"
	"toolBox/api/internal/config"
	"toolBox/api/internal/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

// root.serverCmd represents the root.server command
var rootServerCmd = &cobra.Command{
	Use:   "server",
	Short: "starting server",
	Long:  "Starts the API server.",
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func init() {
	rootCmd.AddCommand(rootServerCmd)
}

func startServer() {
	conf, err := config.LoadServerConfig()
	if err != nil {
		fmt.Println("Erreur de chargement de la config:", err)
		return
	}

	listenURL := fmt.Sprintf("%s://%s:%d", conf.Protocol, conf.FQDN, conf.Port)
	listenSocket := fmt.Sprintf(":%d", conf.Port)

	r := mux.NewRouter()
	handlers.SetupRoutes(r) // On va définir les routes dans un fichier séparé

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Remplacez par l'URL de votre frontend React
		AllowCredentials: true,
	})

	// Lancer le serveur
	fmt.Printf("Starting server at %s \n", listenURL)
	if err := http.ListenAndServe(listenSocket, c.Handler(r)); err != nil {
		fmt.Println(err)
	}
}
