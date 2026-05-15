package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"toolBox/pkg/toolboxruntime"
)

const (
	defaultConfigFile = "test-env.json"
)

type Config struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

func Default() Config {
	return Config{
		Name:      "local-test-env",
		Variables: map[string]string{"ENV": "local"},
	}
}

func Ensure(path string) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.MarshalIndent(Default(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func DefaultPath() string {
	layout, err := toolboxruntime.ForModule("test-env")
	if err != nil {
		return filepath.Join("data", defaultConfigFile)
	}
	return filepath.Join(layout.DataDir, defaultConfigFile)
}
