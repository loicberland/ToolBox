package db

import "toolBox/pkg/database"

var DBConfig = []database.Base{
	{
		DBFile: "test",
		Path:   "./bdd",
		Tables: []database.Table{
			{
				TableName: "test",
				Columns: []database.Column{
					{
						Identifier: "ID",
						Type:       "INTERGER",
						Primary:    true,
						Autoinc:    true,
						// Unique     :true,
						// NotNull    :,
						// Default    Default,
						// Check      :,
						// ForeignKey ForeignKey,
					},
					{
						Identifier: "ID",
						Type:       "TEST_NAME",
						// Primary:    true,
						// Autoinc:    true,
						Unique: true,
						// NotNull    :,
						// Default    Default,
						// Check      :,
						// ForeignKey ForeignKey,
					},
				},
			},
		},
	},
}
