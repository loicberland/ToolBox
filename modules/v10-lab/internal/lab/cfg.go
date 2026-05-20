package lab

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var servicesWithoutDB = map[string]bool{
	"webserver": true,
	"reactor":   true,
}

func ConfigureGedixCfg(config Config, writer io.Writer) error {
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(paths.CfgPath)
	if err != nil {
		return err
	}
	content := string(data)
	content = setRootKey(content, "fqdn", config.GedixConfig.FQDN, true)
	if config.GedixConfig.Port > 0 {
		content = setPort(content, config.GedixConfig.Port)
	}
	serviceNames := sortedServiceNames(config.GedixConfig.Services)
	for _, serviceName := range serviceNames {
		service := config.GedixConfig.Services[serviceName]
		section := fmt.Sprintf("environments.%s.applications.%s.services.%s", paths.EnvName, paths.AppName, serviceName)
		if !sectionExists(content, section) {
			fmt.Fprintf(writer, "[ERROR] Section introuvable dans gedix.cfg : [%s]\n", section)
			return fmt.Errorf("section introuvable dans gedix.cfg: [%s]", section)
		}
		fmt.Fprintf(writer, "[INFO] Section trouvée : [%s]\n", section)
		if servicesWithoutDB[serviceName] {
			content = removeOrCommentKey(content, section, "db-type")
			content = removeOrCommentKey(content, section, "db-dsn")
			continue
		}
		dbType := strings.ToLower(strings.TrimSpace(service.DBType))
		if dbType == "" || dbType == "sqlite" {
			content = removeOrCommentKey(content, section, "db-type")
			content = removeOrCommentKey(content, section, "db-dsn")
		} else {
			content = setSectionKey(content, section, "db-type", service.DBType, true)
			fmt.Fprintf(writer, "[INFO] Mise à jour clé db-type dans service %s\n", serviceName)
			content = setSectionKey(content, section, "db-dsn", service.DBDSN, true)
			fmt.Fprintf(writer, "[INFO] Mise à jour clé db-dsn dans service %s\n", serviceName)
		}
		for _, key := range sortedMapKeys(service.ExtraKeys) {
			content = setSectionKey(content, section, key, service.ExtraKeys[key], shouldQuote(service.ExtraKeys[key]))
			fmt.Fprintf(writer, "[INFO] Mise à jour clé %s dans service %s\n", key, serviceName)
		}
	}
	for _, connectorName := range sortedConnectorNames(config.GedixConfig.Connectors) {
		connector := config.GedixConfig.Connectors[connectorName]
		section := fmt.Sprintf("environments.%s.applications.%s.connectors.%s", paths.EnvName, paths.AppName, connectorName)
		if !sectionExists(content, section) {
			fmt.Fprintf(writer, "[ERROR] Section connecteur introuvable dans gedix.cfg : [%s]\n", section)
			return fmt.Errorf("section connecteur introuvable dans gedix.cfg: [%s]", section)
		}
		fmt.Fprintf(writer, "[INFO] Section connecteur trouvée : [%s]\n", section)
		content = appendRawConfigToSection(content, section, connector.RawConfig)
	}
	if err := os.WriteFile(paths.CfgPath, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Fprintf(writer, "gedix.cfg configuré: %s\n", paths.CfgPath)
	return nil
}

func setRootKey(content string, key string, value string, quote bool) string {
	if strings.TrimSpace(value) == "" {
		return content
	}
	lines := splitLines(content)
	rendered := renderKey(key, value, quote)
	for index, line := range lines {
		if rootKeyMatches(line, key) {
			lines[index] = rendered
			return joinLines(lines)
		}
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			return joinLines(insertLine(lines, index, rendered))
		}
	}
	return rendered + "\n" + content
}

func setPort(content string, port int) string {
	lines := splitLines(content)
	for index, line := range lines {
		if rootKeyMatches(line, "port") {
			trimmed := strings.TrimSpace(line)
			if port == 80 && strings.HasPrefix(trimmed, "#") {
				return content
			}
			lines[index] = fmt.Sprintf("port=%d", port)
			return joinLines(lines)
		}
	}
	insertAt := 0
	for index, line := range lines {
		if rootKeyMatches(line, "fqdn") {
			insertAt = index + 1
			break
		}
	}
	return joinLines(insertLine(lines, insertAt, fmt.Sprintf("port=%d", port)))
}

func ensureSection(content string, section string) string {
	if sectionExists(content, section) {
		return content
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n[" + section + "]\n"
}

func sectionExists(content string, section string) bool {
	target := "[" + section + "]"
	for _, line := range splitLines(content) {
		if strings.EqualFold(strings.TrimSpace(line), target) {
			return true
		}
	}
	return false
}

func setSectionKey(content string, section string, key string, value string, quote bool) string {
	lines := splitLines(content)
	start, end := sectionRange(lines, section)
	if start == -1 {
		return content
	}
	rendered := renderKey(key, value, quote)
	for index := start + 1; index < end; index++ {
		if activeKeyMatches(lines[index], key) {
			lines[index] = rendered
			return joinLines(lines)
		}
	}
	return joinLines(insertLine(lines, end, rendered))
}

func removeOrCommentKey(content string, section string, key string) string {
	if !sectionExists(content, section) {
		return content
	}
	lines := splitLines(content)
	start, end := sectionRange(lines, section)
	if start == -1 {
		return content
	}
	for index := start + 1; index < end; index++ {
		if activeKeyMatches(lines[index], key) {
			lines[index] = "#" + lines[index]
		}
	}
	return joinLines(lines)
}

func appendRawConfigToSection(content string, section string, raw string) string {
	raw = strings.TrimRight(raw, "\r\n")
	if raw == "" {
		return content
	}
	lines := splitLines(content)
	start, end := sectionRange(lines, section)
	if start == -1 {
		return content
	}
	rawLines := strings.Split(raw, "\n")
	for index := range rawLines {
		rawLines[index] = strings.TrimRight(rawLines[index], "\r")
	}
	next := append([]string{}, lines[:end]...)
	next = append(next, rawLines...)
	next = append(next, lines[end:]...)
	return joinLines(next)
}

func sectionRange(lines []string, section string) (int, int) {
	target := "[" + section + "]"
	start := -1
	for index, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), target) {
			start = index
			break
		}
	}
	if start == -1 {
		return -1, -1
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			end = index
			break
		}
	}
	return start, end
}

func rootKeyMatches(line string, key string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "[") {
		return false
	}
	return keyMatches(line, key)
}

func activeKeyMatches(line string, key string) bool {
	trimmed := strings.TrimSpace(line)
	return !strings.HasPrefix(trimmed, "#") && keyMatches(line, key)
}

func keyMatches(line string, key string) bool {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "#")
	trimmed = strings.TrimSpace(trimmed)
	return strings.HasPrefix(trimmed, key+"=")
}

func renderKey(key string, value string, quote bool) string {
	if quote {
		return fmt.Sprintf("%s=%s", key, quoteValue(value))
	}
	return fmt.Sprintf("%s=%s", key, value)
}

func quoteValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value
	}
	return fmt.Sprintf("%q", value)
}

func shouldQuote(value string) bool {
	value = strings.TrimSpace(value)
	return !(len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`))
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func insertLine(lines []string, index int, line string) []string {
	next := append([]string{}, lines[:index]...)
	next = append(next, line)
	next = append(next, lines[index:]...)
	return next
}

func sortedServiceNames(items map[string]ServiceDBConfig) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedConnectorNames(items map[string]ConnectorConfig) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys(items map[string]string) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
