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
		db, err = OpenDataBase(dbPath)
		if err != nil {
			return
		}
		versions, errGetAllVersion := queries.GetAllVersionOrderByValue(db)
		if errGetAllVersion != nil {
			err = fmt.Errorf("error while trying to get all version : %s", errGetAllVersion)
			return
		}
		if len(base.Versions)-1 != versions[0].Value {
			var errUpdateDB error
			errUpdateDB = UpdateDatabase(db, base, versions[0].Value, sqlFiles)
			if errUpdateDB != nil {
				err = fmt.Errorf("error while trying to update %s : %s", dbPath, errUpdateDB)
				return
			}
		}
	}
	return
}

func RevertDataBase(base Base, sqlFiles embed.FS, versionToRevert int) (db *sql.DB, err error) {
	//Vérifier si base exist
	pathDirectory := filepath.Join(utils.GetCurrentDirectory(), base.Path)
	dbPath := filepath.Join(pathDirectory, base.DBFile)
	exist, errCheck := utils.CheckDir(pathDirectory)
	if errCheck != nil {
		err = fmt.Errorf("error while trying to check existance of '%s': %s", pathDirectory, errCheck)
		return
	}
	//Sinon on quitte
	if !exist {
		log.Printf("[LOG] directory %s does not exist", pathDirectory)
		return
	}
	db, err = OpenDataBase(dbPath)
	if err != nil {
		return
	}
	//Récupère version actuel
	versions, errGetAllVersion := queries.GetAllVersionOrderByValue(db)
	if errGetAllVersion != nil {
		err = fmt.Errorf("error while trying to get all version : %s", errGetAllVersion)
		return
	}
	//Boucle de la version actuel à la versionToRevert
	for _, version := range versions {
		if version.Value <= versionToRevert {
			return
		}
		if errRevert := GetRevert(db, base, version.Value, version.ID, sqlFiles); errRevert != nil {
			err = fmt.Errorf("error while trying to revert version %d : %s", version.Value, errRevert)
			return
		}
	}
	// for indexVersion := versions[0].Value; indexVersion > versionToRevert; indexVersion-- {
	// 	//executer la query de la version à delete
	// 	if errRevert := GetRevert(db, base, indexVersion, versions[indexVersion].ID, sqlFiles); errRevert != nil {
	// 		err = fmt.Errorf("error while trying to revert version %d : %s", versions[indexVersion].Value, errRevert)
	// 		return
	// 	}
	// }

	log.Printf("[LOG] base %s was revert to verion %d", base.DBFile, versionToRevert)

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
	for versionIndex := 0; versionIndex < len(base.Versions); versionIndex++ {
		if errDeploy := GetDeploy(db, base, versionIndex, sqlFiles); errDeploy != nil {
			log.Printf("[LOG] error during database creation (%s) we delete it.", dbFile)
			db.Close()
			if errRemove := os.Remove(dbFile); errRemove != nil {
				err = fmt.Errorf("error while trying to delete database file: %s", errRemove)
				return
			}
			err = fmt.Errorf("error while trying to deploy version %d : %s", versionIndex, errDeploy)
			return
		}
	}
	return
}

func UpdateDatabase(db *sql.DB, base Base, actualVersion int, sqlFiles embed.FS) (err error) {
	for i := actualVersion + 1; i < len(base.Versions); i++ {
		if errDeploy := GetDeploy(db, base, i, sqlFiles); errDeploy != nil {
			err = fmt.Errorf("error while trying to deploy version %d : %s", i, errDeploy)
			return
		}
	}
	return
}
func GetDeploy(db *sql.DB, base Base, version int, sqlFiles embed.FS) (err error) {
	//On récupère les fichier sql de deploy
	sqlFilesDatas, errGetSQlFile := GetSqlRequestFromDeployEmbedFiles(sqlFiles)
	if errGetSQlFile != nil {
		err = fmt.Errorf("error while trying to get sql deploy files : %s", errGetSQlFile)
		return
	}
	//On cherche le fichier de la version que l'on souhaite exécuter
	sqlFileData, errFindRequestFile := findRequestFile(base.Versions[version], sqlFilesDatas)
	if errFindRequestFile != nil {
		err = fmt.Errorf("error while trying to find file data for version %d named %s : %s", version, base.Versions[version], errFindRequestFile)
		return
	}
	//On l'exécute
	errCreateTable := SendQueryWithoutResult(db, sqlFileData.query)
	if errCreateTable != nil {
		err = fmt.Errorf("error while trying to create table in version %d: %s", version, errCreateTable)
		return
	}
	//On ajoute la version dans la base
	if errAddVersion := queries.AddVersion(db, version, base.Versions[version]); errAddVersion != nil {
		err = fmt.Errorf("error while trying to add version %d name %s : %s", version, base.Versions[version], errAddVersion)
		return
	}
	return
}

func GetRevert(db *sql.DB, base Base, version, id int, sqlFiles embed.FS) (err error) {
	//On récupère les fichier sql de revert
	sqlFilesDatas, errGetSQlFile := GetSqlRequestFromRevertEmbedFiles(sqlFiles)
	if errGetSQlFile != nil {
		err = fmt.Errorf("error while trying to get sql revert files : %s", errGetSQlFile)
		return
	}
	//On cherche le fichier de la version que l'on souhaite exécuter
	sqlFileData, errFindRequestFile := findRequestFile(base.Versions[version], sqlFilesDatas)
	if errFindRequestFile != nil {
		err = fmt.Errorf("error while trying to find file data for version %d named %s : %s", version, base.Versions[version], errFindRequestFile)
		return
	}
	//On l'exécute
	errCreateTable := SendQueryWithoutResult(db, sqlFileData.query)
	if errCreateTable != nil {
		err = fmt.Errorf("error while trying to create table in version %d: %s", version, errCreateTable)
		return
	}
	//On delete la version
	if errAddVersion := queries.DeletedVersion(db, id); errAddVersion != nil {
		err = fmt.Errorf("error while trying to add version %d name %s : %s", version, base.Versions[version], errAddVersion)
		return
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
