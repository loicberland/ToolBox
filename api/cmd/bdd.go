package cmd

import (
	"database/sql"
	"log"
	"toolBox/api/internal/db"
	"toolBox/pkg/database"

	"github.com/spf13/cobra"
)

// root.serverCmd represents the root.server command
var bddCmd = &cobra.Command{
	Use:   "bdd",
	Short: "Database",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.Fatalf("unable to print help: %s\n", err)
		}
	},
}

var initBddCmd = &cobra.Command{
	Use:   "initBdd",
	Short: "Database initilize",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func init() {
	rootCmd.AddCommand(bddCmd)
	bddCmd.AddCommand(initBddCmd)
}
