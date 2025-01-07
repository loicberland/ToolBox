/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"toolBox/api/internal/db"
	"toolBox/api/internal/db/migration"
	"toolBox/api/internal/handlers"
	"toolBox/pkg/database"
	"toolBox/pkg/server"

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
	confAPI, errConfApi := server.LoadServerConfig("API")
	if errConfApi != nil {
		fmt.Println("Erreur de chargement de la config:", errConfApi)
		return
	}
	confFront, errConfFront := server.LoadServerConfig("FRONT")
	if errConfFront != nil {
		fmt.Println("Erreur de chargement de la config:", errConfFront)
		return
	}
	// Stocker les connexions à chaque base de données
	dbConnections := make([]*sql.DB, 0)
	sqlFiles := migration.Deploy
	for _, base := range db.DBConfig {
		db, err := database.InitDB(base, sqlFiles)
		if err != nil {
			log.Fatalf("Error initializing database whyle trying to start serveur: %v", err)
		}
		dbConnections = append(dbConnections, db) // Ajouter la connexion à la liste
	}

	defer func() {
		for _, dbConn := range dbConnections {
			dbConn.Close() // Fermer proprement chaque connexion
		}
	}()

	listenURL := fmt.Sprintf("%s://%s:%d", confAPI.Protocol, confAPI.FQDN, confAPI.Port)
	listenSocket := fmt.Sprintf(":%d", confAPI.Port)

	r := mux.NewRouter()
	handlers.SetupRoutes(r) // On va définir les routes dans un fichier séparé

	// Configure CORS
	frontURL := fmt.Sprintf("%s://%s:%d", confFront.Protocol, confFront.FQDN, confFront.Port)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{frontURL}, // Remplacez par l'URL de votre frontend React
		AllowCredentials: true,
	})

	// Lancer le serveur
	fmt.Printf("Starting server at %s \n", listenURL)
	if err := http.ListenAndServe(listenSocket, c.Handler(r)); err != nil {
		fmt.Println(err)
	}
}
