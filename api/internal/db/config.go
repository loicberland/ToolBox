package db

import "toolBox/pkg/database"

var DBConfig = []database.Base{
	{
		DBFile: "test",
		Path:   "./bdd",
		Deploy: "./api/internal/db/migration/deploy",
		Revert: "./api/internal/db/migration/revert",
		Versions: database.Version{
			0: "init.sql",
			1: "v1.sql",
		},
	},
}
