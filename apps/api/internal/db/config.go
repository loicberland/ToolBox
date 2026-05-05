package db

import "toolBox/pkg/database"

var DBConfig = []database.Base{
	{
		DBFile: "test.db",
		Path:   "BDD",
		Versions: database.Version{
			0: "init.sql",
		},
	},
}
