package config

import (
	"os"

	"toolBox/pkg/toolboxconfig"
	"toolBox/pkg/toolboxruntime"
)

type Config struct {
	Addr       string
	WebOrigins []string
}

func Load(configPath string) (Config, error) {
	if configPath == "" {
		layout, err := toolboxruntime.ForApp("")
		if err != nil {
			return Config{}, err
		}
		defaultConfigPath := layout.ConfigPath()
		if _, err := os.Stat(defaultConfigPath); err == nil {
			configPath = defaultConfigPath
		} else if !os.IsNotExist(err) {
			return Config{}, err
		}
	}

	cfg, err := toolboxconfig.Load(configPath, toolboxconfig.Overrides{})
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:       cfg.API.Addr,
		WebOrigins: cfg.CORS.Origins,
	}, nil
}
