package db

import "toolBox/pkg/database"

var DBConfig = []database.Base{
	{
		DBFile: "test.db",
		Path:   "bdd",
		Versions: database.Version{
			0: "init.sql",
			1: "v1.sql",
		},
	},
}
