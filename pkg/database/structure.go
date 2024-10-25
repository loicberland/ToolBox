package database

type Base struct {
	DBFile   string
	Path     string
	Deploy   string
	Revert   string
	Versions Version
}

type Version map[int]string
