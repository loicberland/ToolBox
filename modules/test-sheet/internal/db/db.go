package db

import (
	"database/sql"

	"toolBox/modules/test-sheet/pkg/repository"

	_ "github.com/mattn/go-sqlite3"
)

func DefaultPath() string {
	return repository.DefaultPath()
}

func Open(path string) (*sql.DB, error) {
	repo, err := repository.Open(path)
	if err != nil {
		return nil, err
	}
	return repo.DB(), nil
}
