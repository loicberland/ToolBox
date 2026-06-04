package lab

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

type GedixRootConfig struct {
	FQDN string
	Port int
	TLS  bool
}

func ReadGedixRootConfig(config Config) (GedixRootConfig, error) {
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return GedixRootConfig{}, err
	}
	return readGedixRootConfigFromPath(paths.CfgPath)
}

func readGedixRootConfigFromPath(cfgPath string) (GedixRootConfig, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return GedixRootConfig{}, err
	}
	values := map[string]string{}
	for _, rawLine := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(stripGedixCfgComment(rawLine))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			break
		}
		key, value, ok := gedixCfgKeyValue(line)
		if ok {
			values[strings.ToLower(key)] = value
		}
	}
	fqdn := strings.TrimSpace(values["fqdn"])
	if fqdn == "" {
		return GedixRootConfig{}, fmt.Errorf("fqdn absent du fichier cfg")
	}
	port := 80
	if rawPort := strings.TrimSpace(values["port"]); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed <= 0 {
			return GedixRootConfig{}, fmt.Errorf("port Gedix invalide dans le fichier cfg: %q", rawPort)
		}
		port = parsed
	}
	return GedixRootConfig{
		FQDN: fqdn,
		Port: port,
		TLS:  strings.EqualFold(strings.TrimSpace(values["tls"]), "true"),
	}, nil
}

func GedixWebBaseURL(config Config) (string, error) {
	rootConfig, err := ReadGedixRootConfig(config)
	if err != nil {
		return "", err
	}
	base := url.URL{
		Scheme: gedixURLScheme(rootConfig.TLS),
		Host:   fmt.Sprintf("%s:%d", rootConfig.FQDN, rootConfig.Port),
	}
	return strings.TrimRight(base.String(), "/"), nil
}

func GedixAPIBaseURL(config Config) (string, error) {
	if strings.TrimSpace(config.API.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(config.API.BaseURL), "/"), nil
	}
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return "", err
	}
	rootConfig, err := readGedixRootConfigFromPath(paths.CfgPath)
	if err != nil {
		return "", err
	}
	base := url.URL{
		Scheme: gedixURLScheme(rootConfig.TLS),
		Host:   fmt.Sprintf("%s:%d", rootConfig.FQDN, rootConfig.Port),
		Path:   path.Join("env_"+paths.EnvName, "app_"+paths.AppName),
	}
	return strings.TrimRight(base.String(), "/"), nil
}

func gedixURLScheme(tls bool) string {
	if tls {
		return "https"
	}
	return "http"
}

func gedixCfgKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	value = strings.TrimSpace(value)
	return key, value, true
}

func stripGedixCfgComment(line string) string {
	inQuotes := false
	for index, char := range line {
		if char == '"' {
			inQuotes = !inQuotes
		}
		if !inQuotes && (char == '#' || char == ';') {
			return line[:index]
		}
	}
	return line
}
