package database

type Base struct {
	DBFile   string
	Path     string
	Versions Version
}

type Version map[int]string
