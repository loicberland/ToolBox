package paths

import (
	"os"
	"path/filepath"
)

func Root() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func Data(parts ...string) string {
	all := append([]string{Root(), "data"}, parts...)
	return filepath.Join(all...)
}

func Build(parts ...string) string {
	all := append([]string{Root(), "_build"}, parts...)
	return filepath.Join(all...)
}
