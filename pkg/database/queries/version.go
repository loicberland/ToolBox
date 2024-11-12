package queries

import (
	"database/sql"
	"fmt"
	"log"
)

type Version struct {
	ID         int
	Value      int
	File       string
	Created_at string
}

func GetVersionByID(db *sql.DB, ID int) (*Version, error) {
	query := "SELECT * FROM VERSION WHERE ID = ?"
	row := db.QueryRow(query, ID)
	version := &Version{}
	err := row.Scan(&version.ID, &version.Value, &version.File)
	if err != nil {
		return nil, err
	}
	return version, nil
}

func GetAllVersionOrderByValue(db *sql.DB) ([]Version, error) {
	versions := []Version{}
	query := "SELECT * FROM VERSION ORDER BY VALUE"
	rows, errQuery := db.Query(query)
	if errQuery != nil {
		return nil, fmt.Errorf("error while trying to exec query '%s' : %s", query, errQuery)

	}
	for rows.Next() {
		version := Version{}
		if err := rows.Scan(&version.ID, &version.Value, &version.File); err != nil {
			return nil, fmt.Errorf("error while scanning row: %s", err)
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func AddVersion(db *sql.DB, value int, fileName string) error {
	// Préparation de la requête SQL
	query := "INSERT INTO VERSION(VALUE,FILE) VALUES (?,?)"

	// Exécution de la requête avec `Exec`
	result, err := db.Exec(query, value, fileName)
	if err != nil {
		return fmt.Errorf("erreur while trying to exec '%s' : %s", query, err)
	}

	// Vérification du nombre de lignes affectées
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erreur while trying to check rows number affected by query '%s' : %s", query, err)
	}
	if rowsAffected == 0 {
		log.Printf("[LOG] the query '%s' did not affect any rows", query)
	}

	return nil
}
