package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"toolBox/pkg/database/queries"
	"toolBox/pkg/utils"

	_ "github.com/mattn/go-sqlite3" // Import nécessaire pour utiliser SQLite avec Go
)

type sqlEmbed struct {
	fileName string
	query    string
}

func InitDB(base Base, sqlFiles embed.FS) (db *sql.DB, err error) {
	pathDirectory := filepath.Join(utils.GetCurrentDirectory(), base.Path)
	if errCheckExistDir := utils.CheckOrCreateDir(pathDirectory); errCheckExistDir != nil {
		err = fmt.Errorf("error while trying to check the existence of %s : %s", pathDirectory, errCheckExistDir)
		return
	}
	dbPath := filepath.Join(pathDirectory, base.DBFile)
	exist, errCheck := CheckExistDataBase(dbPath)
	if errCheck != nil {
		err = fmt.Errorf("error while trying to check existence of %s: %s", base.DBFile, errCheck)
		return
	}
	if !exist {
		log.Printf("[LOG] '%s' The database does not exist, we will create it.", dbPath)
		var errCreateDB error
		db, errCreateDB = CreateDatabase(dbPath, base, sqlFiles)
		if errCreateDB != nil {
			err = fmt.Errorf("error while trying to create %s : %s", dbPath, errCreateDB)
			return
		}
	} else {
		versions, errGetAllVersion := queries.GetAllVersionOrderByValue(db)
		if errGetAllVersion != nil {
			err = fmt.Errorf("error while trying to get all version : %s", errGetAllVersion)
			return
		}
		//Cherche si la dernière version est dans la base
		//Sinon faire une recherche de la dernière version en base et update à partir d'elle
		if len(base.Versions)-1 != versions[0].Value {
			var errUpdateDB error
			db, errUpdateDB = UpdateDatabase(dbPath, base, versions[0].Value, sqlFiles)
			if errUpdateDB != nil {
				err = fmt.Errorf("error while trying to update %s : %s", dbPath, errUpdateDB)
				return //Tester
			}
		}
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

func CreateDatabase(dbFile string, base Base, sqlFiles embed.FS) (db *sql.DB, err error) {
	parentDir := filepath.Dir(dbFile)
	if errCheckExistDir := utils.CheckOrCreateDir(parentDir); errCheckExistDir != nil {
		err = fmt.Errorf("error while trying to check the existence of %s : %s", parentDir, errCheckExistDir)
		return
	}
	db, err = OpenDataBase(dbFile)
	if err != nil {
		return
	}
	for versionIndex := range base.Versions {
		sqlFilesDatas, errGetSQlFile := GetSqlRequestFromDeployEmbedFiles(sqlFiles)
		if errGetSQlFile != nil {
			err = fmt.Errorf("error while trying to get sql deploy files : %s", errGetSQlFile)
			return
		}
		sqlFileData, errFindRequestFile := findRequestFile(base.Versions[versionIndex], sqlFilesDatas)
		if errFindRequestFile != nil {
			err = fmt.Errorf("error while trying to find file data for version %d named %s : %s", versionIndex, base.Versions[versionIndex], errFindRequestFile)
			return
		}
		errCreateTable := SendQueryWithoutResult(db, sqlFileData.query)
		if errCreateTable != nil {
			err = fmt.Errorf("error while trying to create table in version %d: %s", versionIndex, errCreateTable)
			return
		}
		if errAddVersion := queries.AddVersion(db, versionIndex, base.Versions[versionIndex]); errAddVersion != nil {
			err = fmt.Errorf("error while trying to add version %d name %s : %s", versionIndex, base.Versions[versionIndex], errAddVersion)
			return
		}
	}
	return
}

func UpdateDatabase(dbFile string, base Base, actualVersion int, sqlFiles embed.FS) (db *sql.DB, err error) {
	db, err = OpenDataBase(dbFile)
	if err != nil {
		return
	}
	for i := actualVersion + 1; i < len(base.Versions)-1; i++ {
		sqlFilesDatas, errGetSQlFile := GetSqlRequestFromDeployEmbedFiles(sqlFiles)
		if errGetSQlFile != nil {
			err = fmt.Errorf("error while trying to get sql deploy files : %s", errGetSQlFile)
			return
		}
		sqlFileData, errFindRequestFile := findRequestFile(base.Versions[i], sqlFilesDatas)
		if errFindRequestFile != nil {
			err = fmt.Errorf("error while trying to find file data for version %d named %s : %s", i, base.Versions[i], errFindRequestFile)
			return
		}
		errCreateTable := SendQueryWithoutResult(db, sqlFileData.query)
		if errCreateTable != nil {
			err = fmt.Errorf("error while trying to create table in version %d: %s", i, errCreateTable)
			return
		}
		if errAddVersion := queries.AddVersion(db, i, base.Versions[i]); errAddVersion != nil {
			err = fmt.Errorf("error while trying to add version %d name %s : %s", i, base.Versions[i], errAddVersion)
			return
		}
	}
	return
}

func GetSqlRequestFromDeployEmbedFiles(sqlFiles embed.FS) (datas []sqlEmbed, err error) {
	dir := "deploy"
	files, errReadSqlFiles := sqlFiles.ReadDir(dir)
	if errReadSqlFiles != nil {
		err = fmt.Errorf("error while trying to read sqlFiles : %s", errReadSqlFiles)
		return
	}
	return GetSqlRequestFromFiles(sqlFiles, dir, files)
}
func GetSqlRequestFromRevertEmbedFiles(sqlFiles embed.FS) (datas []sqlEmbed, err error) {
	dir := "revert"
	files, errReadSqlFiles := sqlFiles.ReadDir(dir)
	if errReadSqlFiles != nil {
		err = fmt.Errorf("error while trying to read sqlFiles : %s", errReadSqlFiles)
		return
	}
	return GetSqlRequestFromFiles(sqlFiles, dir, files)
}

func GetSqlRequestFromFiles(sqlFiles embed.FS, dir string, files []fs.DirEntry) (datas []sqlEmbed, err error) {
	for _, file := range files {
		data := sqlEmbed{}
		if file.IsDir() {
			log.Printf("[LOG] %s is a dir", file.Name())
			continue
		}
		if filepath.Ext(file.Name()) != ".sql" {
			log.Printf("[LOG] %s was not an sql file", file.Name())
			continue
		}
		fileName := file.Name()
		path := filepath.Join(dir, fileName)
		path = strings.Replace(path, "\\", "/", -1)
		query, errReadFile := sqlFiles.ReadFile(path)
		if errReadFile != nil {
			err = fmt.Errorf("error while trying to read files %s : %s", file.Name(), errReadFile)
			return
		}
		data.fileName = file.Name()
		data.query = string(query)
		datas = append(datas, data)
	}
	return
}

func findRequestFile(fileName string, datas []sqlEmbed) (file sqlEmbed, err error) {
	for _, data := range datas {
		if data.fileName == fileName {
			file = data
			return
		}
	}
	err = fmt.Errorf("error file %s doesn't exist", fileName)
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

func SendQueryWithoutResult(db *sql.DB, query string) (err error) {
	_, errCreateDB := db.Exec(query)
	if errCreateDB != nil {
		err = fmt.Errorf("error while trying to create table with command '%s' : %s", string(query), errCreateDB)
		return
	}
	return
}

func UpdateVersion() {

}
