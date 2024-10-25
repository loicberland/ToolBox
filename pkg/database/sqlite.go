package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // Import nécessaire pour utiliser SQLite avec Go
)

func InitDB(base Base) (db *sql.DB, err error) {
	dbPath := base.Path + "/" + base.DBFile + ".db"
	exist, errCheck := CheckExistDataBase(dbPath)
	if errCheck != nil {
		err = fmt.Errorf("error while trying to check existence of %s: %s", base.DBFile, errCheck)
		return
	}
	if !exist {
		log.Printf("[LOG] '%s' The database does not exist, we will create it.", dbPath)
		var errCreateDB error
		db, errCreateDB = CreateDatabase(dbPath, base)
		if errCreateDB != nil {
			err = fmt.Errorf("error while trying to create %s : %s", dbPath, errCreateDB)
			return
		}
	} else {
		db, err = OpenDataBase(dbPath)
		if err != nil {
			return
		}
	}
	return
}

func CheckExistDataBase(dbFile string) (exist bool, err error) {
	exist = true
	if _, errStat := os.Stat(dbFile); errStat != nil {
		if os.IsNotExist(errStat) {
			exist = false
			return
		}
		err = fmt.Errorf("error while trying to check existance of '%s': %s", dbFile, errStat)
		return
	}
	return
}

func OpenDataBase(dbFile string) (db *sql.DB, err error) {
	db, errOpenDb := sql.Open("sqlite3", dbFile)
	if errOpenDb != nil {
		err = fmt.Errorf("error while trying to open '%s' ", errOpenDb)
		return
	}
	return
}

func CreateDatabase(dbFile string, base Base) (db *sql.DB, err error) {
	parentDir := filepath.Dir(dbFile)
	if _, errStat := os.Stat(parentDir); errStat != nil {
		if !os.IsNotExist(errStat) {
			err = fmt.Errorf("error while trying to check existance of '%s': %s", parentDir, errStat)
			return
		} else {
			if errMkdir := os.Mkdir(parentDir, 0755); errMkdir != nil {
				err = fmt.Errorf("error while trying to create dir '%s': %s", parentDir, errMkdir)
				return
			}
		}
	}
	db, err = OpenDataBase(dbFile)
	if err != nil {
		return
	}
	for versionIndex := range base.Versions {
		file := base.Deploy + "/" + base.Versions[versionIndex]
		query, errGetQuery := GetSQLQueryFromFile(db, file)
		if errGetQuery != nil {
			err = fmt.Errorf("error when trying to get query from file %s : %s", file, errGetQuery)
			return
		}
		errCreateTable := SendQuery(db, query)
		if errCreateTable != nil {
			err = fmt.Errorf("error while trying to create table in version %d: %s", versionIndex, errCreateTable)
			return
		}

	}
	return
}

func UpdateDatabase(dbFile string, base Base, actualVersion int) (db *sql.DB, err error) {
	db, err = OpenDataBase(dbFile)
	if err != nil {
		return
	}
	for i := actualVersion + 1; i < len(base.Versions)-1; i++ {
		file := base.Deploy + "/" + base.Versions[i]
		query, errGetQuery := GetSQLQueryFromFile(db, file)
		if errGetQuery != nil {
			err = fmt.Errorf("error when trying to get query from file %s : %s", file, errGetQuery)
			return
		}
		errCreateTable := SendQuery(db, query)
		if errCreateTable != nil {
			err = fmt.Errorf("error while trying to create table in version %d: %s", i, errCreateTable)
			return
		}

	}
	return
}

func GetActualVersion(db *sql.DB) (version int, err error) {
	// query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s);`, table.TableName, columns)
	// errCreateTable := SendQuery(db, query)
	// 	if errCreateTable != nil {
	// 		err = fmt.Errorf("error while trying to create table in version %d: %s", i, errCreateTable)
	// 		return
	// 	}
	return
}

func GetSQLQueryFromFile(db *sql.DB, file string) (query string, err error) {
	queryRead, errRead := os.ReadFile(file)
	if errRead != nil {
		err = fmt.Errorf("error when trying to read %s : %s", file, errRead)
		return
	}
	query = string(queryRead)
	return
}

func SendQuery(db *sql.DB, query string) (err error) {
	_, errCreateDB := db.Exec(query)
	if errCreateDB != nil {
		err = fmt.Errorf("error while trying to create table with command '%s' : %s", string(query), errCreateDB)
		return
	}
	return
}

func UpdateVersion() {

}
