package utils

import (
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
