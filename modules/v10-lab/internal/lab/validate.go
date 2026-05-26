package lab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var allowedDBTypes = map[string]bool{
	"":         true,
	"sqlite":   true,
	"mysql":    true,
	"postgres": true,
	"mssql":    true,
	"oracle":   true,
}

func ValidateConfig(config Config) error {
	ApplyDefaults(&config)
	errors := []string{}
	if strings.TrimSpace(config.Name) == "" {
		errors = append(errors, "name: champ requis manquant")
	} else if safeDirName(config.Name) == "sans-nom" {
		errors = append(errors, "name: nom invalide")
	}
	if strings.TrimSpace(config.Product) == "" {
		errors = append(errors, "product: champ requis manquant")
	} else if !ProductExists(config.Product) {
		errors = append(errors, fmt.Sprintf("product: produit inconnu %q", config.Product))
	}
	if strings.TrimSpace(config.Release.ZipPath) != "" {
		if !strings.EqualFold(filepath.Ext(config.Release.ZipPath), ".zip") {
			errors = append(errors, "release.zipPath: le fichier doit etre un ZIP .zip")
		} else if info, err := os.Stat(config.Release.ZipPath); err != nil {
			errors = append(errors, fmt.Sprintf("release.zipPath: fichier introuvable %q", config.Release.ZipPath))
		} else if info.IsDir() {
			errors = append(errors, fmt.Sprintf("release.zipPath: chemin vers un dossier %q", config.Release.ZipPath))
		}
	}
	if config.GedixConfig.Port < 0 || config.GedixConfig.Port > 65535 {
		errors = append(errors, "gedixConfig.port: port invalide")
	}
	seenDebugTargets := map[string]bool{}
	for _, target := range config.Runtime.DebugTargets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if seenDebugTargets[target] {
			errors = append(errors, fmt.Sprintf("runtime.debugTargets: doublon %q", target))
		}
		seenDebugTargets[target] = true
	}
	for target, flags := range config.Runtime.DebugTargetFlags {
		if strings.TrimSpace(target) == "" {
			errors = append(errors, "runtime.debugTargetFlags: cible vide")
			continue
		}
		seenFlags := map[string]bool{}
		for _, flag := range flags {
			normalized, err := NormalizeDebugFlag(flag)
			if err != nil {
				errors = append(errors, fmt.Sprintf("runtime.debugTargetFlags.%s: %v", target, err))
				continue
			}
			if seenFlags[normalized] {
				errors = append(errors, fmt.Sprintf("runtime.debugTargetFlags.%s: doublon %q", target, normalized))
			}
			seenFlags[normalized] = true
		}
	}
	for serviceName, service := range config.GedixConfig.Services {
		dbType := strings.ToLower(strings.TrimSpace(service.DBType))
		if !allowedDBTypes[dbType] {
			errors = append(errors, fmt.Sprintf("gedixConfig.services.%s.dbType: type inconnu %q", serviceName, service.DBType))
			continue
		}
		if dbType != "" && dbType != "sqlite" && strings.TrimSpace(service.DBDSN) == "" {
			errors = append(errors, fmt.Sprintf("gedixConfig.services.%s.dbDsn: champ requis pour dbType %q", serviceName, service.DBType))
		}
	}
	for index, step := range config.Pipeline {
		action, ok := FindAction(step.Action)
		if !ok {
			errors = append(errors, fmt.Sprintf("pipeline[%d].action: action inconnue %q", index, step.Action))
			continue
		}
		if !action.SupportsProduct(config.Product) {
			errors = append(errors, fmt.Sprintf("pipeline[%d].action: action %q incompatible avec le produit %q", index, step.Action, config.Product))
		}
		if step.Action == "create-env" && strings.TrimSpace(config.Release.ZipPath) == "" && strings.TrimSpace(stringParam(step.Params, "zipPath")) == "" {
			errors = append(errors, fmt.Sprintf("pipeline[%d].release.zipPath: champ requis pour create-env", index))
		}
		params := step.Params
		if params == nil {
			params = map[string]any{}
		}
		for _, field := range action.Fields {
			value, exists := params[field.Name]
			if !exists && field.Default != nil {
				value = field.Default
				exists = true
			}
			if field.Required && !exists {
				errors = append(errors, fmt.Sprintf("pipeline[%d].params.%s: champ requis manquant", index, field.Name))
				continue
			}
			if exists && !fieldValueMatchesType(value, field.Type) {
				errors = append(errors, fmt.Sprintf("pipeline[%d].params.%s: type attendu %s", index, field.Name, field.Type))
			}
		}
	}
	if len(errors) > 0 {
		return ValidationError{Items: errors}
	}
	return nil
}

func fieldValueMatchesType(value any, expected string) bool {
	switch expected {
	case "", "any":
		return true
	case "string":
		_, ok := value.(string)
		return ok
	case "text":
		_, ok := value.(string)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	case "string[]":
		switch items := value.(type) {
		case []any:
			for _, item := range items {
				if _, ok := item.(string); !ok {
					return false
				}
			}
			return true
		case []string:
			return true
		default:
			return false
		}
	default:
		return true
	}
}
