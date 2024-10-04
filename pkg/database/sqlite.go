package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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
		for _, table := range base.Tables {
			var errCreateDB error
			db, errCreateDB = CreateDatabase(dbPath, table)
			if errCreateDB != nil {
				err = fmt.Errorf("error while trying to create %s : %s", dbPath, errCreateDB)
				return
			}
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
func CreateDatabase(dbFile string, table Table) (db *sql.DB, err error) {
	db, err = OpenDataBase(dbFile)
	if err != nil {
		return
	}
	errCreateTable := CreateTable(db, table)
	if errCreateTable != nil {
		err = fmt.Errorf("error while trying to create table %s: %s", table.TableName, errCreateTable)
		return
	}
	return
}

func CreateTable(db *sql.DB, table Table) (err error) {
	var columnList []string
	for _, col := range table.Columns {
		column, errConcat := ConcatColumnsToString(col)
		if errConcat != nil {
			err = fmt.Errorf("error while trying to concat columns: %s", errConcat)
			return
		}
		columnList = append(columnList, column)
	}
	columns := strings.Join(columnList, ", ")
	createTableSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s);`, table.TableName, columns)

	_, errCreateDB := db.Exec(createTableSQL)
	if errCreateDB != nil {
		err = fmt.Errorf("error while trying to create table '%s' with command '%s' : %s", table.TableName, createTableSQL, errCreateDB)
		return
	}
	return
}

func ConcatColumnsToString(col Column) (stringCol string, err error) {
	if col.Identifier == "" || col.Type == "" {
		err = fmt.Errorf("error identifier (%s) or Type %s is empty", col.Identifier, col.Type)
		return
	}
	//LBE : Ajouter fonction pour vérifier le type
	stringCol = fmt.Sprintf("%s %s", col.Identifier, col.Type)
	if col.Primary {
		stringCol = fmt.Sprintf("%s PRIMARY KEY", stringCol)
	}
	if col.Autoinc {
		stringCol = fmt.Sprintf("%s AUTOINCREMENT", stringCol)
	}
	if col.Unique {
		stringCol = fmt.Sprintf("%s UNIQUE", stringCol)
	}
	if col.NotNull {
		stringCol = fmt.Sprintf("%s NOT NULL", stringCol)
	}
	if col.Check != "" {
		stringCol = fmt.Sprintf("%s CHECK(%s)", stringCol, col.Check)
	}
	if col.Default.Active {
		defaulCol, errGetDefaul := GetDefaultValueColumn(col)
		if errGetDefaul != nil {
			err = fmt.Errorf("error while trying to : %s", errGetDefaul)
			return
		}
		stringCol = fmt.Sprintf("%s %s", stringCol, defaulCol)
	}
	if col.ForeignKey.Active {
		foreignCol, errGetForeign := GetForeignValueColumn(col)
		if errGetForeign != nil {
			err = fmt.Errorf("error while trying to : %s", errGetForeign)
			return
		}
		stringCol = fmt.Sprintf("%s %s", stringCol, foreignCol)
	}
	return
}

func GetDefaultValueColumn(col Column) (stringCol string, err error) {
	switch col.Type {
	case "TEXT":
		if col.Default.DefaultStr == "" {
			err = fmt.Errorf("error the default string value is empty")
			return
		}
		stringCol = fmt.Sprintf("DEFAULT('%s')", col.Default.DefaultStr)
	case "INTEGER":
		if col.Default.DefaultNumber != 0 {
			err = fmt.Errorf("error the default number value is 0")
			return
		}
		stringCol = fmt.Sprintf("DEFAULT(%f)", col.Default.DefaultNumber)
	default:
		err = fmt.Errorf("error this type (%s) does not exist", col.Type)
	}
	return
}

func GetForeignValueColumn(col Column) (stringCol string, err error) {
	if col.ForeignKey.table == "" {
		err = fmt.Errorf("error the foreignKey table is empty")
		return
	}
	if col.ForeignKey.Column == "" {
		err = fmt.Errorf("error the foreignKey column is empty")
		return
	}
	stringCol = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", col.Identifier, col.ForeignKey.table, col.ForeignKey.Column)
	return
}
