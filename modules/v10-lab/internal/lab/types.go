package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"toolBox/pkg/modulecontract"
	"toolBox/pkg/toolboxruntime"
)

const (
	ModuleID   = "v10-lab"
	ModuleName = "V10 Lab"

	ProductGedixV10 = "gedix-v10"
	KindSystem      = "system"
	KindAPI         = "api"
)

type Product struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type ActionField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     any    `json:"default"`
	Description string `json:"description"`
}

type Action struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Kind        string        `json:"kind"`
	Products    []string      `json:"products"`
	Fields      []ActionField `json:"fields"`
	Execute     ActionExecute `json:"-"`
}

type ActionExecute func(ctx ActionContext, params map[string]any) error

type ActionContext struct {
	Writer io.Writer
	Config Config
	Step   PipelineStep
}

type Config struct {
	Name     string         `json:"name"`
	Product  string         `json:"product"`
	Release  ReleaseConfig  `json:"release"`
	API      APIConfig      `json:"api"`
	Database DatabaseConfig `json:"database"`
	Services []ServiceSpec  `json:"services"`
	Pipeline []PipelineStep `json:"pipeline"`
}

type ReleaseConfig struct {
	SourcePath string `json:"sourcePath"`
	TargetPath string `json:"targetPath"`
}

type APIConfig struct {
	BaseURL  string `json:"baseUrl"`
	TokenRef string `json:"tokenRef"`
}

type DatabaseConfig struct {
	Type    string `json:"type"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Service string `json:"service"`
	Schema  string `json:"schema"`
}

type ServiceSpec struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Debug   bool   `json:"debug"`
}

type PipelineStep struct {
	Action string         `json:"action"`
	Label  string         `json:"label"`
	Params map[string]any `json:"params"`
}

type RegisteredMaquette struct {
	Name    string `json:"name"`
	Product string `json:"product"`
	Path    string `json:"path"`
}

type ValidationError struct {
	Items []string
}

func (e ValidationError) Error() string {
	return "validation failed"
}

func (e ValidationError) Format() string {
	if len(e.Items) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("Erreur validation:\n")
	for _, item := range e.Items {
		builder.WriteString("- ")
		builder.WriteString(item)
		builder.WriteByte('\n')
	}
	return strings.TrimRight(builder.String(), "\n")
}

func Info() modulecontract.ModuleInfo {
	return modulecontract.ModuleInfo{
		ID:          ModuleID,
		Name:        ModuleName,
		Description: "Generateur de maquettes V10",
		Actions: []modulecontract.ModuleAction{
			{ID: "products", Name: "Produits", Description: "Liste les produits supportes"},
			{ID: "actions", Name: "Actions", Description: "Liste les actions disponibles"},
			{ID: "validate", Name: "Valider", Description: "Valide une configuration JSON"},
			{ID: "run", Name: "Executer", Description: "Execute fictivement un pipeline"},
			{ID: "register", Name: "Enregistrer", Description: "Enregistre une maquette localement"},
			{ID: "list", Name: "Lister", Description: "Liste les maquettes enregistrees"},
		},
	}
}

func Products() []Product {
	return []Product{
		{ID: ProductGedixV10, Label: "Gedix V10"},
	}
}

func ProductExists(productID string) bool {
	for _, product := range Products() {
		if product.ID == productID {
			return true
		}
	}
	return false
}

func Actions() []Action {
	return []Action{
		{
			ID:          "create-env",
			Label:       "Créer maquette",
			Description: "Prépare le dossier d’une maquette à partir d’une release.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "releasePath", Label: "Release", Type: "string", Required: true},
				{Name: "targetPath", Label: "Dossier cible", Type: "string", Required: true},
				{Name: "overwrite", Label: "Écraser", Type: "bool", Default: false},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Prépare le dossier %s depuis %s\n", stringParam(params, "targetPath"), stringParam(params, "releasePath"))
				return nil
			},
		},
		{
			ID:          "start-services",
			Label:       "Démarrer services",
			Description: "Démarre les services/exécutables configurés pour la maquette.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "debugServices", Label: "Services debug", Type: "string[]"},
				{Name: "services", Label: "Services", Type: "string[]"},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Démarre les services %v avec debug %v\n", params["services"], params["debugServices"])
				return nil
			},
		},
		{
			ID:          "stop-services",
			Label:       "Arrêter services",
			Description: "Arrête les services/exécutables de la maquette.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "taskkill", Label: "Forcer taskkill", Type: "bool", Default: false},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Arrête les services (taskkill=%v)\n", boolParam(params, "taskkill"))
				return nil
			},
		},
		{
			ID:          "create-machine-group",
			Label:       "Créer groupe machine",
			Description: "Crée fictivement un groupe machine Gedix V10.",
			Kind:        KindAPI,
			Products:    []string{ProductGedixV10},
			Fields: []ActionField{
				{Name: "code", Label: "Code", Type: "string", Required: true},
				{Name: "name", Label: "Nom", Type: "string", Required: true},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer groupe machine %s / %s\n", stringParam(params, "code"), stringParam(params, "name"))
				return nil
			},
		},
		{
			ID:          "create-machine",
			Label:       "Créer machine",
			Description: "Crée fictivement une machine Gedix V10.",
			Kind:        KindAPI,
			Products:    []string{ProductGedixV10},
			Fields: []ActionField{
				{Name: "code", Label: "Code", Type: "string", Required: true},
				{Name: "name", Label: "Nom", Type: "string", Required: true},
				{Name: "groupCode", Label: "Groupe", Type: "string"},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer machine %s / %s\n", stringParam(params, "code"), stringParam(params, "name"))
				return nil
			},
		},
		{
			ID:          "create-cnc-folder",
			Label:       "Créer dossier CN",
			Description: "Crée fictivement un dossier CN Gedix V10.",
			Kind:        KindAPI,
			Products:    []string{ProductGedixV10},
			Fields: []ActionField{
				{Name: "machineGroupCode", Label: "Groupe machine", Type: "string", Required: true},
				{Name: "programCode", Label: "Programme", Type: "string", Required: true},
				{Name: "programIndex", Label: "Indice", Type: "string", Default: "A"},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer dossier CN %s indice %s pour groupe %s\n", stringParam(params, "programCode"), stringParam(params, "programIndex"), stringParam(params, "machineGroupCode"))
				return nil
			},
		},
	}
}

func ActionsForProduct(productID string) []Action {
	items := []Action{}
	for _, action := range Actions() {
		if action.SupportsProduct(productID) {
			items = append(items, action)
		}
	}
	return items
}

func FindAction(actionID string) (Action, bool) {
	for _, action := range Actions() {
		if action.ID == actionID {
			return action, true
		}
	}
	return Action{}, false
}

func (a Action) SupportsProduct(productID string) bool {
	if len(a.Products) == 0 {
		return true
	}
	for _, product := range a.Products {
		if product == productID {
			return true
		}
	}
	return false
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func ValidateConfig(config Config) error {
	errors := []string{}
	if strings.TrimSpace(config.Name) == "" {
		errors = append(errors, "name: champ requis manquant")
	}
	if strings.TrimSpace(config.Product) == "" {
		errors = append(errors, "product: champ requis manquant")
	} else if !ProductExists(config.Product) {
		errors = append(errors, fmt.Sprintf("product: produit inconnu %q", config.Product))
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
		params := step.Params
		if params == nil {
			params = map[string]any{}
		}
		for _, field := range action.Fields {
			value, exists := params[field.Name]
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

func RunPipeline(ctx context.Context, config Config, writer io.Writer) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	fmt.Fprintf(writer, "V10 Lab - Exécution maquette %s\n", config.Name)
	fmt.Fprintf(writer, "Produit: %s\n\n", config.Product)
	for index, step := range config.Pipeline {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		action, _ := FindAction(step.Action)
		label := step.Label
		if strings.TrimSpace(label) == "" {
			label = action.Label
		}
		params := paramsWithDefaults(action, step.Params)
		fmt.Fprintf(writer, "[%d/%d] %s - %s\n", index+1, len(config.Pipeline), action.ID, label)
		if err := action.Execute(ActionContext{Writer: writer, Config: config, Step: step}, params); err != nil {
			return err
		}
		fmt.Fprintln(writer)
	}
	fmt.Fprintln(writer, "Exécution terminée.")
	return nil
}

func MaquettesDir() string {
	layout, err := toolboxruntime.ForModule(ModuleID)
	if err != nil {
		return filepath.Join("data", ModuleID, "maquettes")
	}
	return filepath.Join(layout.DataDir, "maquettes")
}

func RegisterConfig(configPath string) (RegisteredMaquette, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return RegisteredMaquette{}, err
	}
	if err := ValidateConfig(config); err != nil {
		return RegisteredMaquette{}, err
	}
	name := safeDirName(config.Name)
	targetDir := filepath.Join(MaquettesDir(), name)
	if err := os.MkdirAll(filepath.Join(targetDir, "data"), 0755); err != nil {
		return RegisteredMaquette{}, err
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "logs"), 0755); err != nil {
		return RegisteredMaquette{}, err
	}
	source, err := os.ReadFile(configPath)
	if err != nil {
		return RegisteredMaquette{}, err
	}
	targetPath := filepath.Join(targetDir, "maquette.json")
	if err := os.WriteFile(targetPath, source, 0644); err != nil {
		return RegisteredMaquette{}, err
	}
	return RegisteredMaquette{Name: config.Name, Product: config.Product, Path: targetPath}, nil
}

func ListMaquettes() ([]RegisteredMaquette, error) {
	root := MaquettesDir()
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []RegisteredMaquette{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []RegisteredMaquette{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "maquette.json")
		config, err := LoadConfig(path)
		if err != nil {
			continue
		}
		items = append(items, RegisteredMaquette{Name: config.Name, Product: config.Product, Path: path})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func paramsWithDefaults(action Action, params map[string]any) map[string]any {
	next := map[string]any{}
	for key, value := range params {
		next[key] = value
	}
	for _, field := range action.Fields {
		if _, exists := next[field.Name]; !exists && field.Default != nil {
			next[field.Name] = field.Default
		}
	}
	return next
}

func fieldValueMatchesType(value any, expected string) bool {
	switch expected {
	case "", "any":
		return true
	case "string":
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

func stringParam(params map[string]any, key string) string {
	value, _ := params[key].(string)
	return value
}

func boolParam(params map[string]any, key string) bool {
	value, _ := params[key].(bool)
	return value
}

func safeDirName(value string) string {
	name := strings.TrimSpace(value)
	name = strings.NewReplacer("\\", "-", "/", "-", ":", "", "*", "", "?", "", `"`, "", "<", "", ">", "", "|", "").Replace(name)
	name = strings.Join(strings.Fields(name), "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-.")
	if name == "" {
		return "sans-nom"
	}
	return name
}
