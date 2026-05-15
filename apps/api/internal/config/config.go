package config

import "toolBox/pkg/toolboxconfig"

type Config struct {
	Addr       string
	WebOrigins []string
}

func Load(configPath string) (Config, error) {
	cfg, err := toolboxconfig.Load(configPath, toolboxconfig.Overrides{})
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:       cfg.API.Addr,
		WebOrigins: cfg.CORS.Origins,
	}, nil
}
