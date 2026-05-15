package toolboxconfig

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPlatformFQDN = "localhost"
	DefaultPlatformPort = 20251
	DefaultPlatformTLS  = false
	DefaultPlatformBind = ""
	DefaultAPIHost      = "127.0.0.1:20250"
)

var DefaultCORSOrigins = []string{"http://localhost:3000", "http://localhost:20251"}

type Config struct {
	Platform PlatformConfig
	Web      WebConfig
	API      APIConfig
	CORS     CORSConfig

	legacyWebAddr   string
	legacyPublicURL string
	legacyAPITarget string
}

type PlatformConfig struct {
	FQDN string
	Port int
	TLS  bool
	Bind string
}

type WebConfig struct {
	Addr      string
	PublicURL string
}

type APIConfig struct {
	Addr   string
	Target string
}

type CORSConfig struct {
	Origins []string
}

type Overrides struct {
	WebAddr     string
	APITarget   string
	CORSOrigins []string
}

func Load(configPath string, overrides Overrides) (Config, error) {
	cfg := Default()

	if err := loadFile(configPath, &cfg); err != nil {
		return Config{}, err
	}
	applyEnv(&cfg)
	derive(&cfg)
	applyLegacy(&cfg)
	applyOverrides(&cfg, overrides)

	return cfg, nil
}

func Default() Config {
	cfg := Config{
		Platform: PlatformConfig{
			FQDN: DefaultPlatformFQDN,
			Port: DefaultPlatformPort,
			TLS:  DefaultPlatformTLS,
			Bind: DefaultPlatformBind,
		},
		API: APIConfig{
			Addr: DefaultAPIHost,
		},
		CORS: CORSConfig{
			Origins: append([]string(nil), DefaultCORSOrigins...),
		},
	}
	derive(&cfg)
	return cfg
}

func derive(cfg *Config) {
	scheme := "http"
	if cfg.Platform.TLS {
		scheme = "https"
	}

	cfg.Web.Addr = fmt.Sprintf(":%d", cfg.Platform.Port)
	if cfg.Platform.Bind != "" {
		cfg.Web.Addr = fmt.Sprintf("%s:%d", cfg.Platform.Bind, cfg.Platform.Port)
	}
	cfg.Web.PublicURL = fmt.Sprintf("%s://%s:%d", scheme, cfg.Platform.FQDN, cfg.Platform.Port)
	cfg.API.Target = "http://" + cfg.API.Addr
}

func applyLegacy(cfg *Config) {
	if cfg.legacyWebAddr != "" {
		cfg.Web.Addr = cfg.legacyWebAddr
	}
	if cfg.legacyPublicURL != "" {
		cfg.Web.PublicURL = cfg.legacyPublicURL
	}
	if cfg.legacyAPITarget != "" {
		cfg.API.Target = cfg.legacyAPITarget
	}
}

func loadFile(configPath string, cfg *Config) error {
	path := configPath
	if path == "" {
		if _, err := os.Stat("toolbox.cfg"); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("read toolbox.cfg: %w", err)
		}
		path = "toolbox.cfg"
	}

	file, err := os.Open(path)
	if err != nil {
		if configPath == "" && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open config %s: %w", path, err)
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "[") && !strings.Contains(value, "]") {
			for scanner.Scan() {
				next := strings.TrimSpace(stripComment(scanner.Text()))
				value += next
				if strings.Contains(next, "]") {
					break
				}
			}
		}
		applyFileValue(cfg, section, key, value)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	return nil
}

func applyFileValue(cfg *Config, section, key, value string) {
	switch section + "." + key {
	case "platform.fqdn":
		cfg.Platform.FQDN = parseString(value)
	case "platform.port":
		if port, err := strconv.Atoi(parseString(value)); err == nil {
			cfg.Platform.Port = port
		}
	case "platform.tls":
		if tls, err := strconv.ParseBool(parseString(value)); err == nil {
			cfg.Platform.TLS = tls
		}
	case "platform.bind":
		cfg.Platform.Bind = parseString(value)
	case "services.api.host":
		cfg.API.Addr = parseString(value)
	case "cors.origins":
		cfg.CORS.Origins = parseList(value)

	// Legacy keys kept so older local toolbox.cfg files do not break abruptly.
	case "web.addr":
		cfg.legacyWebAddr = parseString(value)
	case "web.public_url":
		cfg.legacyPublicURL = parseString(value)
	case "api.addr":
		cfg.API.Addr = parseString(value)
	case "api.target":
		cfg.legacyAPITarget = parseString(value)
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("TOOLBOX_FQDN"); value != "" {
		cfg.Platform.FQDN = value
	}
	if value := os.Getenv("TOOLBOX_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Platform.Port = port
		}
	}
	if value := os.Getenv("TOOLBOX_TLS"); value != "" {
		if tls, err := strconv.ParseBool(value); err == nil {
			cfg.Platform.TLS = tls
		}
	}
	if value := os.Getenv("TOOLBOX_BIND"); value != "" {
		cfg.Platform.Bind = value
	}
	if value := os.Getenv("TOOLBOX_API_HOST"); value != "" {
		cfg.API.Addr = value
	}
	if value := os.Getenv("TOOLBOX_CORS_ORIGINS"); value != "" {
		cfg.CORS.Origins = splitCSV(value)
	}

	// Legacy environment variables remain accepted during the transition.
	if value := os.Getenv("TOOLBOX_WEB_ADDR"); value != "" {
		cfg.legacyWebAddr = value
	}
	if value := os.Getenv("TOOLBOX_WEB_PUBLIC_URL"); value != "" {
		cfg.legacyPublicURL = value
	}
	if value := os.Getenv("TOOLBOX_API_ADDR"); value != "" {
		cfg.API.Addr = value
	}
	if value := os.Getenv("TOOLBOX_API_TARGET"); value != "" {
		cfg.legacyAPITarget = value
	}
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.WebAddr != "" {
		cfg.Web.Addr = overrides.WebAddr
	}
	if overrides.APITarget != "" {
		cfg.API.Target = overrides.APITarget
	}
	if len(overrides.CORSOrigins) > 0 {
		cfg.CORS.Origins = append([]string(nil), overrides.CORSOrigins...)
	}
}

func stripComment(line string) string {
	inQuotes := false
	for i, r := range line {
		if r == '"' {
			inQuotes = !inQuotes
		}
		if r == '#' && !inQuotes {
			return line[:i]
		}
	}
	return line
}

func parseString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return value
}

func parseList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	return splitCSV(value)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = parseString(strings.TrimSpace(part))
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
