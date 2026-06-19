package lab

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type cfgEntry struct {
	Key   string
	Lines []string
	Start int
	End   int
}

func ConfigureGedixCfg(config Config, writer io.Writer) error {
	ApplyDefaults(&config)
	product, err := ProductDefinitionByID(config.Product)
	if err != nil {
		return err
	}
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
		serviceDefinition, ok := product.Service(serviceName)
		if !ok {
			continue
		}
		service := config.GedixConfig.Services[serviceName]
		section := fmt.Sprintf("environments.%s.applications.%s.services.%s", paths.EnvName, paths.AppName, serviceName)
		if !sectionExists(content, section) {
			if isDefaultServiceDBConfig(service) {
				continue
			}
			fmt.Fprintf(writer, "[ERROR] Section introuvable dans gedix.cfg : [%s]\n", section)
			return fmt.Errorf("section introuvable dans gedix.cfg: [%s]", section)
		}
		fmt.Fprintf(writer, "[INFO] Section trouvée : [%s]\n", section)
		if !serviceDefinition.HasDatabase {
			content = removeOrCommentKey(content, section, "db-type")
			content = removeOrCommentKey(content, section, "db-dsn")
		} else {
			dbType := strings.ToLower(strings.TrimSpace(service.DBType))
			if dbType == "" || (dbType == "sqlite" && strings.TrimSpace(service.DBDSN) == "") {
				content = removeOrCommentKey(content, section, "db-type")
				content = removeOrCommentKey(content, section, "db-dsn")
			} else {
				content = setSectionKey(content, section, "db-type", service.DBType, true)
				fmt.Fprintf(writer, "[INFO] Mise à jour clé db-type dans service %s\n", serviceName)
				content = setSectionKey(content, section, "db-dsn", service.DBDSN, true)
				fmt.Fprintf(writer, "[INFO] Mise à jour clé db-dsn dans service %s\n", serviceName)
			}
		}
		if serviceDefinition.SupportsExtraKeys {
			for _, key := range sortedMapKeys(service.ExtraKeys) {
				content = setSectionRawBlock(content, section, cfgEntry{
					Key:   key,
					Lines: []string{fmt.Sprintf("%s=%s", key, service.ExtraKeys[key])},
				})
				fmt.Fprintf(writer, "[INFO] Mise à jour clé %s dans service %s\n", key, serviceName)
			}
		}
	}
	for _, family := range ProductUnitFamilies(config) {
		definition := family.Definition
		for _, unitName := range sortedUnitNames(family.Units) {
			unit := family.Units[unitName]
			section := fmt.Sprintf("environments.%s.applications.%s.%s.%s", paths.EnvName, paths.AppName, definition.CfgSectionName, unitName)
			if !sectionExists(content, section) {
				fmt.Fprintf(writer, "[ERROR] Section %s introuvable dans gedix.cfg : [%s]\n", definition.SingularLabel, section)
				return fmt.Errorf("section %s introuvable dans gedix.cfg: [%s]", definition.SingularLabel, section)
			}
			fmt.Fprintf(writer, "[INFO] Section %s trouvee : [%s]\n", definition.SingularLabel, section)
			content = applyConnectorRawConfig(content, section, unit.RawConfig)
		}
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
	return applyConnectorRawConfig(content, section, raw)
}

func applyConnectorRawConfig(content string, section string, raw string) string {
	raw = strings.TrimRight(raw, "\r\n")
	if raw == "" {
		return content
	}
	for _, entry := range parseCfgEntries(splitLines(raw)) {
		if strings.EqualFold(entry.Key, "type") || strings.EqualFold(entry.Key, "host") {
			continue
		}
		content = setSectionRawBlock(content, section, entry)
	}
	return content
}

func setSectionRawBlock(content string, section string, entry cfgEntry) string {
	lines := splitLines(content)
	start, end := sectionRange(lines, section)
	if start == -1 {
		return content
	}
	for _, existing := range parseCfgEntryRanges(lines, start+1, end) {
		if strings.EqualFold(existing.Key, entry.Key) {
			next := append([]string{}, lines[:existing.Start]...)
			next = append(next, entry.Lines...)
			next = append(next, lines[existing.End:]...)
			return joinLines(next)
		}
	}
	next := append([]string{}, lines[:end]...)
	next = append(next, entry.Lines...)
	next = append(next, lines[end:]...)
	return joinLines(next)
}

func parseCfgEntries(lines []string) []cfgEntry {
	return parseCfgEntryRanges(lines, 0, len(lines))
}

func parseCfgEntryRanges(lines []string, start int, end int) []cfgEntry {
	entries := []cfgEntry{}
	current := cfgEntry{Start: -1}
	for index := start; index < end; index++ {
		line := strings.TrimRight(lines[index], "\r")
		key, ok := cfgKeyFromLine(line)
		if ok {
			if current.Start != -1 {
				current.End = index
				entries = append(entries, current)
			}
			current = cfgEntry{Key: key, Lines: []string{line}, Start: index}
			continue
		}
		if current.Start != -1 {
			current.Lines = append(current.Lines, line)
		}
	}
	if current.Start != -1 {
		current.End = end
		entries = append(entries, current)
	}
	return entries
}

func cfgKeyFromLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	index := strings.Index(trimmed, "=")
	if index <= 0 {
		return "", false
	}
	key := strings.TrimSpace(trimmed[:index])
	if key == "" || !isCfgKey(key) {
		return "", false
	}
	return key, true
}

func isCfgKey(key string) bool {
	for _, char := range key {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
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
	lineKey, ok := cfgKeyFromLine(trimmed)
	return ok && strings.EqualFold(lineKey, key)
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

func sortedUnitNames(items map[string]ProductUnitConfig) []string {
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

func isDefaultServiceDBConfig(service ServiceDBConfig) bool {
	dbType := strings.ToLower(strings.TrimSpace(service.DBType))
	return (dbType == "" || dbType == "sqlite") && strings.TrimSpace(service.DBDSN) == "" && len(service.ExtraKeys) == 0
}
