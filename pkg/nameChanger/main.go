package namechanger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"toolBox/pkg/utils"
)

func Namechanger(choice int, directory, inputSearch, inputReplace string) {
	for {
		if directory == "" {
			dir, err := os.Executable()
			if err != nil {
				fmt.Println("Erreur lors de la récupération du chemin de l'exécutable :", err)
				return
			}
			directory = filepath.Dir(dir)
		}
		findAndReplaceString(directory, inputSearch, inputReplace, choice)
	}

}

func findAndReplaceString(dirPath, searchString, replaceString string, option int) {
	files, errFiles := os.ReadDir(dirPath)
	countRename := 0
	//On récupère le nom de l'exe
	dir, _ := os.Executable()
	exeName := filepath.Base(dir)
	//Vérification path correct
	if errFiles != nil {
		log.Fatal(errFiles)
	}
	for _, file := range files {
		if file.Name() != exeName { //Sécuritée pour pas renommer l'exe
			oldPath := filepath.Join(dirPath, file.Name())
			if utils.IsDirectory(oldPath) {
				if option == 3 {
					continue
				}
			}
			if strings.Contains(file.Name(), searchString) {
				newName := strings.ReplaceAll(file.Name(), searchString, replaceString)
				newPath := filepath.Join(dirPath, newName)
				errRename := os.Rename(oldPath, newPath)
				if errRename != nil {
					log.Fatal(errRename)
				}
				fmt.Println("**", file.Name(), " ==> ", newName)
				countRename++
			}
		}
	}
	if countRename > 1 {
		fmt.Println(countRename, "éléments ont été renommés.")
	} else if countRename == 1 {
		fmt.Println(countRename, "élément a été renommé.")
	} else {
		fmt.Println(countRename, "élément renommé.")
	}
}
