package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func GetCurrentDirectory() string {
	curDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return curDir
}

func CheckOrCreateDir(path string) (err error) {
	_, errStat := os.Stat(path)
	if errStat != nil {
		if !os.IsNotExist(errStat) {
			err = fmt.Errorf("error while trying to check existance of '%s': %s", path, errStat)
			return
		} else {
			log.Printf("[LOG] create directory %s.", path)
			if errMkdir := os.Mkdir(path, 0755); errMkdir != nil {
				err = fmt.Errorf("error while trying to create dir '%s': %s", path, errMkdir)
				return
			}
		}
	}
	return
}
