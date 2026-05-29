package lab

import (
	"encoding/json"
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
		params := paramsWithDefaults(action, step.Params)
		for _, field := range action.Fields {
			if actionFieldHidden(field, params) {
				continue
			}
			value, exists := params[field.Name]
			if field.Required && (!exists || fieldValueIsEmpty(value)) {
				label := strings.TrimSpace(field.Label)
				if label == "" {
					label = field.Name
				}
				errors = append(errors, fmt.Sprintf("Étape %d - %s : le champ %s est obligatoire et ne peut pas être vide.", index+1, step.Action, label))
				continue
			}
			if !exists && field.Default != nil {
				value = field.Default
				exists = true
			}
			if exists && !fieldValueMatchesType(value, field.Type) {
				errors = append(errors, fmt.Sprintf("pipeline[%d].params.%s: type attendu %s", index, field.Name, field.Type))
				continue
			}
			if exists && field.ItemMin != 0 && field.Type == "number[]" {
				items, ok := numberListParamStrict(map[string]any{field.Name: value}, field.Name)
				if !ok {
					errors = append(errors, fmt.Sprintf("pipeline[%d].params.%s: type attendu %s", index, field.Name, field.Type))
					continue
				}
				for _, item := range items {
					if float64(item) < field.ItemMin {
						errors = append(errors, fmt.Sprintf("Étape %d - %s : le champ %s doit contenir uniquement des valeurs supérieures à 0.", index+1, step.Action, firstNonEmpty(field.Label, field.Name)))
						break
					}
				}
			}
		}
	}
	if len(errors) > 0 {
		return ValidationError{Items: errors}
	}
	return nil
}

func fieldValueIsEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}

func actionFieldHidden(field ActionField, params map[string]any) bool {
	for key, expected := range field.HiddenWhen {
		if actionValuesEqual(params[key], expected) {
			return true
		}
	}
	return false
}

func actionValuesEqual(left any, right any) bool {
	if fmt.Sprint(left) == fmt.Sprint(right) {
		return true
	}
	leftNumber, leftIsNumber := anyToFloat(left)
	rightNumber, rightIsNumber := anyToFloat(right)
	if leftIsNumber && rightIsNumber {
		return leftNumber == rightNumber
	}
	return false
}

func anyToFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case string:
		var number json.Number = json.Number(strings.TrimSpace(typed))
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
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
	case "number":
		switch value.(type) {
		case int, int64, float64, json.Number:
			return true
		default:
			return false
		}
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
	case "number[]":
		_, ok := numberListParamStrict(map[string]any{"value": value}, "value")
		return ok
	case "object[]":
		switch value.(type) {
		case []any, []map[string]any, string:
			return true
		default:
			return false
		}
	default:
		return true
	}
}
