package database

type Column struct {
	Identifier string
	Type       string
	Primary    bool
	Autoinc    bool
	Unique     bool
	NotNull    bool
	Default    Default
	Check      string
	ForeignKey ForeignKey
}

type ForeignKey struct {
	Active bool
	table  string
	Column string
}

type Default struct {
	Active        bool
	DefaultStr    string
	DefaultNumber float64
}

type Base struct {
	DBFile string
	Path   string
	Tables []Table
}

type Table struct {
	TableName string
	Columns   []Column
}
