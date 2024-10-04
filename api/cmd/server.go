/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"toolBox/api/internal/config"
	"toolBox/api/internal/db"
	"toolBox/api/internal/handlers"
	"toolBox/pkg/database"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	conf, err := config.LoadServerConfig()
	if err != nil {
		fmt.Println("Erreur de chargement de la config:", err)
		return
	}
	// Stocker les connexions à chaque base de données
	dbConnections := make([]*sql.DB, 0)
	for _, base := range db.DBConfig {
		db, err := database.InitDB(base)
		if err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}
		dbConnections = append(dbConnections, db) // Ajouter la connexion à la liste
	}

	defer func() {
		for _, dbConn := range dbConnections {
			dbConn.Close() // Fermer proprement chaque connexion
		}
	}()

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
