package cmd

import (
	"database/sql"
	"log"
	"toolBox/api/internal/db"
	"toolBox/api/internal/db/migration"
	"toolBox/pkg/database"

	"github.com/spf13/cobra"
)

var reversVersion int
var dbName string

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
		sqlFiles := migration.Deploy
		for _, base := range db.DBConfig {
			db, err := database.InitDB(base, sqlFiles)
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

var revertCmd = &cobra.Command{
	Use:   "revert",
	Short: "Revert database to old version",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		sqlFiles := migration.Revert
		for _, base := range db.DBConfig {
			if base.DBFile == dbName+".db" {
				db, err := database.RevertDataBase(base, sqlFiles, reversVersion)
				if err != nil {
					log.Fatalf("Error revert database: %v", err)
				}
				defer func() {
					db.Close() // Fermer proprement chaque connexion
				}()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(bddCmd)
	bddCmd.AddCommand(initBddCmd)
	bddCmd.AddCommand(revertCmd)
	revertCmd.Flags().IntVarP(&reversVersion, "revert-version", "v", 0, "Version to which we want to revert")
	revertCmd.MarkFlagRequired("revert-version")
	revertCmd.Flags().StringVarP(&dbName, "db-name", "", "", "Name of database")
	revertCmd.MarkFlagRequired("db-name")
}
