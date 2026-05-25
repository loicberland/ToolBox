package lab

import (
	"encoding/json"
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

	GedixProdV10 = "gedix-prod-v10"
	KindSystem   = "system"
	KindAPI      = "api"
)

type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Label       string `json:"label,omitempty"`
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
	Hidden      bool          `json:"hidden,omitempty"`
	Execute     ActionExecute `json:"-"`
}

type ActionExecute func(ctx ActionContext, params map[string]any) error

type ActionContext struct {
	Writer io.Writer
	Config Config
	Step   PipelineStep
}

type Config struct {
	Name         string                 `json:"name"`
	Product      string                 `json:"product"`
	Release      ReleaseConfig          `json:"release"`
	Maquette     MaquetteConfig         `json:"maquette"`
	GedixConfig  GedixConfig            `json:"gedixConfig"`
	Runtime      RuntimeConfig          `json:"runtime"`
	API          APIConfig              `json:"api"`
	Database     DatabaseConfig         `json:"database"`
	Services     []ServiceSpec          `json:"services"`
	Pipeline     []PipelineStep         `json:"pipeline"`
	LegacyExtras map[string]interface{} `json:"-"`
}

type ReleaseConfig struct {
	ZipPath    string `json:"zipPath"`
	WorkDir    string `json:"workDir"`
	Overwrite  bool   `json:"overwrite"`
	SourcePath string `json:"sourcePath,omitempty"`
	TargetPath string `json:"targetPath,omitempty"`
}

type MaquetteConfig struct {
	TargetPath string `json:"targetPath"`
	EnvName    string `json:"envName"`
	AppName    string `json:"appName"`
}

type GedixConfig struct {
	FQDN       string                     `json:"fqdn"`
	Port       int                        `json:"port"`
	Services   map[string]ServiceDBConfig `json:"services"`
	Connectors map[string]ConnectorConfig `json:"connectors"`
}

type ServiceDBConfig struct {
	DBType    string            `json:"dbType"`
	DBDSN     string            `json:"dbDsn"`
	ExtraKeys map[string]string `json:"extraKeys"`
}

type ConnectorConfig struct {
	RawConfig string `json:"rawConfig"`
}

type RuntimeConfig struct {
	DebugTargets []string `json:"debugTargets"`
	OpenConsole  bool     `json:"openConsole"`
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

type DBTemplate struct {
	Type     string `json:"type"`
	Template string `json:"template"`
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
			{ID: "db-templates", Name: "Templates DB", Description: "Liste les templates de DSN"},
			{ID: "validate", Name: "Valider", Description: "Valide une configuration JSON"},
			{ID: "run", Name: "Executer", Description: "Execute un plan d'actions de maquette"},
			{ID: "register", Name: "Enregistrer", Description: "Enregistre une maquette localement"},
			{ID: "list", Name: "Lister", Description: "Liste les maquettes enregistrees"},
			{ID: "kill-gx-processes", Name: "Couper les services GX", Description: "Coupe manuellement les processus GX"},
		},
	}
}

func Products() []Product {
	return []Product{
		{ID: GedixProdV10, Name: "Gedix V10 prod", Description: "Produit Gedix V10 prod", Label: "Gedix V10 prod"},
	}
}

func DBTemplates() []DBTemplate {
	return []DBTemplate{
		{Type: "sqlite", Template: ""},
		{Type: "mysql", Template: ""},
		{Type: "postgres", Template: "user= password= dbname= sslmode=disable"},
		{Type: "mssql", Template: "server=;instance=;database=;port=;user id=;password="},
		{Type: "oracle", Template: "/@:/"},
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

func SaveRegisteredConfig(config Config) (RegisteredMaquette, error) {
	ApplyDefaults(&config)
	NormalizeConfigForSave(&config)
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
	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return RegisteredMaquette{}, err
	}
	targetPath := filepath.Join(targetDir, "maquette.json")
	if err := os.WriteFile(targetPath, append(payload, '\n'), 0644); err != nil {
		return RegisteredMaquette{}, err
	}
	return RegisteredMaquette{Name: config.Name, Product: config.Product, Path: targetPath}, nil
}

func DeleteRegisteredConfig(name string) error {
	path := filepath.Join(MaquettesDir(), safeDirName(name), "maquette.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func RegisteredLogsDir(name string) string {
	return filepath.Join(MaquettesDir(), safeDirName(name), "logs")
}

func NormalizeConfigForSave(config *Config) {
	ApplyDefaults(config)
	for serviceName, service := range config.GedixConfig.Services {
		dbType := strings.ToLower(strings.TrimSpace(service.DBType))
		if dbType == "sqlite" && strings.TrimSpace(service.DBDSN) == "" && len(service.ExtraKeys) == 0 {
			delete(config.GedixConfig.Services, serviceName)
		}
	}
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
	ApplyDefaults(&config)
	return config, nil
}

func ApplyDefaults(config *Config) {
	if strings.TrimSpace(config.Maquette.AppName) == "" {
		config.Maquette.AppName = "prod"
	}
	if config.GedixConfig.Services == nil {
		config.GedixConfig.Services = map[string]ServiceDBConfig{}
	}
	if config.GedixConfig.Connectors == nil {
		config.GedixConfig.Connectors = map[string]ConnectorConfig{}
	}
}

func MaquettesDir() string {
	layout, err := toolboxruntime.ForModule(ModuleID)
	if err != nil {
		return filepath.Join("data", ModuleID, "maquettes")
	}
	return filepath.Join(layout.DataDir, "maquettes")
}

func MaquettesFilesDir() string {
	layout, err := toolboxruntime.ForModule(ModuleID)
	if err != nil {
		return filepath.Join("files", ModuleID, "maquettes")
	}
	return filepath.Join(layout.FilesDir, "maquettes")
}

func DefaultMaquetteTargetPath(config Config) string {
	return filepath.Join(MaquettesFilesDir(), "Gedix_"+safeDirName(config.Name))
}

func ResolveMaquetteTargetPath(config Config) string {
	if strings.TrimSpace(config.Maquette.TargetPath) != "" {
		return config.Maquette.TargetPath
	}
	return DefaultMaquetteTargetPath(config)
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

func LoadRegisteredConfig(name string) (Config, string, error) {
	path := filepath.Join(MaquettesDir(), safeDirName(name), "maquette.json")
	config, err := LoadConfig(path)
	return config, path, err
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

func stringParam(params map[string]any, key string) string {
	value, _ := params[key].(string)
	return value
}

func boolParam(params map[string]any, key string) bool {
	value, _ := params[key].(bool)
	return value
}
