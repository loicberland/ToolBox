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
	dbConnections := make([]*sql.DB, 0)
	sqlFiles := migration.Deploy
	for _, base := range db.DBConfig {
		db, err := database.InitDB(base, sqlFiles)
		if err != nil {
			log.Fatalf("Error initializing database whyle trying to start serveur: %v", err)
		}
		dbConnections = append(dbConnections, db)
	}

	defer func() {
		for _, dbConn := range dbConnections {
			dbConn.Close()
		}
	}()

	listenURL := fmt.Sprintf("http://localhost:20250")
	listenSocket := fmt.Sprintf(":20250")

	r := mux.NewRouter()
	handlers.SetupRoutes(r)

	// CORS
	frontURL := fmt.Sprintf("http://localhost:20251")
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{frontURL},
		AllowCredentials: true,
	})

	// Get server
	fmt.Printf("Starting server at %s \n", listenURL)
	if err := http.ListenAndServe(listenSocket, c.Handler(r)); err != nil {
		fmt.Println(err)
	}
}
