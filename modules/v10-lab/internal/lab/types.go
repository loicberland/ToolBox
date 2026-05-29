package lab

import (
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

	KindSystem = "system"
	KindAPI    = "api"
)

type ActionField struct {
	Name            string           `json:"name"`
	Label           string           `json:"label"`
	Type            string           `json:"type"`
	Required        bool             `json:"required"`
	Default         any              `json:"default"`
	Description     string           `json:"description"`
	Options         []ActionOption   `json:"options,omitempty"`
	OptionsSource   string           `json:"optionsSource,omitempty"`
	HiddenWhen      map[string]any   `json:"hiddenWhen,omitempty"`
	HiddenWhenAny   []map[string]any `json:"hiddenWhenAny,omitempty"`
	ItemFields      []ActionField    `json:"itemFields,omitempty"`
	UniqueItemField string           `json:"uniqueItemField,omitempty"`
	Min             float64          `json:"min,omitempty"`
	ItemMin         float64          `json:"itemMin,omitempty"`
	Multiple        bool             `json:"multiple,omitempty"`
}

type ActionOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
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
	GroupName    string                 `json:"groupName,omitempty"`
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
	FQDN       string                       `json:"fqdn"`
	Port       int                          `json:"port"`
	Services   map[string]ServiceDBConfig   `json:"services"`
	Connectors map[string]ProductUnitConfig `json:"connectors"`
	Agents     map[string]ProductUnitConfig `json:"agents,omitempty"`
	Units      map[string]ProductUnitConfig `json:"units,omitempty"`
}

type ServiceDBConfig struct {
	DBType    string            `json:"dbType"`
	DBDSN     string            `json:"dbDsn"`
	ExtraKeys map[string]string `json:"extraKeys"`
}

type ProductUnitConfig struct {
	Module    string `json:"module,omitempty"`
	RawConfig string `json:"rawConfig"`
}

type ConnectorConfig = ProductUnitConfig

type RuntimeConfig struct {
	DebugTargets     []string            `json:"debugTargets"`
	DebugTargetFlags map[string][]string `json:"debugTargetFlags,omitempty"`
	OpenConsole      bool                `json:"openConsole"`
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
	Name      string `json:"name"`
	Product   string `json:"product"`
	Path      string `json:"path"`
	GroupName string `json:"groupName,omitempty"`
}

type MaquetteGroup struct {
	Name string `json:"name"`
}

type groupRegistry struct {
	Groups []MaquetteGroup `json:"groups"`
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

func DBTemplates() []DBTemplate {
	return []DBTemplate{
		{Type: "sqlite", Template: ""},
		{Type: "mysql", Template: ""},
		{Type: "postgres", Template: "user= password= dbname= sslmode=disable"},
		{Type: "mssql", Template: "server=;instance=;database=;port=;user id=;password="},
		{Type: "oracle", Template: "/@:/"},
	}
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
	return RegisteredMaquette{Name: config.Name, Product: config.Product, Path: targetPath, GroupName: config.GroupName}, nil
}

func SaveRegisteredConfigReplacing(oldName string, config Config) (RegisteredMaquette, error) {
	ApplyDefaults(&config)
	NormalizeConfigForSave(&config)
	if err := ValidateConfig(config); err != nil {
		return RegisteredMaquette{}, err
	}
	oldSafe := safeDirName(oldName)
	newSafe := safeDirName(config.Name)
	oldDir := filepath.Join(MaquettesDir(), oldSafe)
	newDir := filepath.Join(MaquettesDir(), newSafe)
	if oldSafe != newSafe {
		if _, err := os.Stat(oldDir); err != nil {
			return RegisteredMaquette{}, err
		}
		if _, err := os.Stat(newDir); err == nil {
			return RegisteredMaquette{}, fmt.Errorf("maquette deja enregistree: %s", config.Name)
		} else if !os.IsNotExist(err) {
			return RegisteredMaquette{}, err
		}
		oldConfig, err := LoadConfig(filepath.Join(oldDir, "maquette.json"))
		if err != nil {
			return RegisteredMaquette{}, err
		}
		oldDefaultTarget := DefaultMaquetteTargetPath(oldConfig)
		newDefaultTarget := DefaultMaquetteTargetPath(config)
		targetPath := strings.TrimSpace(config.Maquette.TargetPath)
		shouldRenameTarget := targetPath == "" || sameCleanPath(targetPath, oldDefaultTarget)
		if shouldRenameTarget {
			config.Maquette.TargetPath = newDefaultTarget
			if _, err := os.Stat(newDefaultTarget); err == nil {
				return RegisteredMaquette{}, fmt.Errorf("dossier Gedix cible deja existant: %s", newDefaultTarget)
			} else if !os.IsNotExist(err) {
				return RegisteredMaquette{}, err
			}
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			return RegisteredMaquette{}, err
		}
		targetRenamed := false
		if shouldRenameTarget {
			if _, err := os.Stat(oldDefaultTarget); err == nil {
				if err := os.MkdirAll(filepath.Dir(newDefaultTarget), 0755); err != nil {
					_ = os.Rename(newDir, oldDir)
					return RegisteredMaquette{}, err
				}
				if err := os.Rename(oldDefaultTarget, newDefaultTarget); err != nil {
					_ = os.Rename(newDir, oldDir)
					return RegisteredMaquette{}, err
				}
				targetRenamed = true
			} else if !os.IsNotExist(err) {
				_ = os.Rename(newDir, oldDir)
				return RegisteredMaquette{}, err
			}
		}
		item, err := saveRegisteredConfigIntoDir(config, newDir)
		if err != nil {
			if targetRenamed {
				_ = os.Rename(newDefaultTarget, oldDefaultTarget)
			}
			_ = os.Rename(newDir, oldDir)
			return RegisteredMaquette{}, err
		}
		return item, nil
	}
	return saveRegisteredConfigIntoDir(config, newDir)
}

func DeleteRegisteredConfig(name string) error {
	if err := DeleteAPIToken(name); err != nil {
		return err
	}
	path := filepath.Join(MaquettesDir(), safeDirName(name), "maquette.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func saveRegisteredConfigIntoDir(config Config, targetDir string) (RegisteredMaquette, error) {
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
	return RegisteredMaquette{Name: config.Name, Product: config.Product, Path: targetPath, GroupName: config.GroupName}, nil
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
	config.Product = NormalizeProductID(config.Product)
	product, _ := ProductDefinitionByID(config.Product)
	if strings.TrimSpace(config.Maquette.AppName) == "" {
		config.Maquette.AppName = firstNonEmpty(product.DefaultAppName, "prod")
	}
	if config.GedixConfig.Services == nil {
		config.GedixConfig.Services = map[string]ServiceDBConfig{}
	}
	if config.GedixConfig.Connectors == nil {
		config.GedixConfig.Connectors = map[string]ProductUnitConfig{}
	}
	if config.GedixConfig.Agents == nil {
		config.GedixConfig.Agents = map[string]ProductUnitConfig{}
	}
	if config.GedixConfig.Units == nil {
		config.GedixConfig.Units = map[string]ProductUnitConfig{}
	}
	if config.Runtime.DebugTargetFlags == nil {
		config.Runtime.DebugTargetFlags = map[string][]string{}
	}
}

func ProductUnits(config Config) map[string]ProductUnitConfig {
	ApplyDefaults(&config)
	product, _ := ProductDefinitionByID(config.Product)
	units := map[string]ProductUnitConfig{}
	if !product.HasUnits() {
		return units
	}
	for name, unit := range config.GedixConfig.Units {
		units[name] = unit
	}
	switch product.UnitKind {
	case UnitKindAgent:
		for name, unit := range config.GedixConfig.Agents {
			units[name] = unit
		}
	default:
		for name, unit := range config.GedixConfig.Connectors {
			units[name] = unit
		}
	}
	return units
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
	return RegisteredMaquette{Name: config.Name, Product: config.Product, Path: targetPath, GroupName: config.GroupName}, nil
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
		items = append(items, RegisteredMaquette{Name: config.Name, Product: config.Product, Path: path, GroupName: config.GroupName})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func ListMaquetteGroups() ([]MaquetteGroup, error) {
	registry, err := loadGroupRegistry()
	if err != nil {
		return nil, err
	}
	sortGroups(registry.Groups)
	return registry.Groups, nil
}

func CreateMaquetteGroup(name string) (MaquetteGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return MaquetteGroup{}, fmt.Errorf("nom de groupe requis")
	}
	if safeDirName(name) == "sans-nom" {
		return MaquetteGroup{}, fmt.Errorf("nom de groupe invalide")
	}
	registry, err := loadGroupRegistry()
	if err != nil {
		return MaquetteGroup{}, err
	}
	for _, group := range registry.Groups {
		if strings.EqualFold(group.Name, name) {
			return MaquetteGroup{}, fmt.Errorf("groupe deja existant: %s", name)
		}
	}
	group := MaquetteGroup{Name: name}
	registry.Groups = append(registry.Groups, group)
	sortGroups(registry.Groups)
	return group, saveGroupRegistry(registry)
}

func RenameMaquetteGroup(oldName string, newName string) (MaquetteGroup, error) {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return MaquetteGroup{}, fmt.Errorf("nom de groupe requis")
	}
	registry, err := loadGroupRegistry()
	if err != nil {
		return MaquetteGroup{}, err
	}
	found := false
	for index, group := range registry.Groups {
		if strings.EqualFold(group.Name, newName) && !strings.EqualFold(group.Name, oldName) {
			return MaquetteGroup{}, fmt.Errorf("groupe deja existant: %s", newName)
		}
		if strings.EqualFold(group.Name, oldName) {
			registry.Groups[index].Name = newName
			found = true
		}
	}
	if !found {
		return MaquetteGroup{}, os.ErrNotExist
	}
	maquettes, err := ListMaquettes()
	if err != nil {
		return MaquetteGroup{}, err
	}
	for _, item := range maquettes {
		config, _, err := LoadRegisteredConfig(item.Name)
		if err != nil {
			return MaquetteGroup{}, err
		}
		if strings.EqualFold(config.GroupName, oldName) {
			config.GroupName = newName
			if _, err := SaveRegisteredConfig(config); err != nil {
				return MaquetteGroup{}, err
			}
		}
	}
	sortGroups(registry.Groups)
	return MaquetteGroup{Name: newName}, saveGroupRegistry(registry)
}

func DeleteMaquetteGroup(name string) error {
	name = strings.TrimSpace(name)
	maquettes, err := ListMaquettes()
	if err != nil {
		return err
	}
	for _, item := range maquettes {
		if strings.EqualFold(item.GroupName, name) {
			return fmt.Errorf("groupe non vide: retirez ou deplacez les maquettes avant suppression")
		}
	}
	registry, err := loadGroupRegistry()
	if err != nil {
		return err
	}
	next := registry.Groups[:0]
	found := false
	for _, group := range registry.Groups {
		if strings.EqualFold(group.Name, name) {
			found = true
			continue
		}
		next = append(next, group)
	}
	if !found {
		return os.ErrNotExist
	}
	registry.Groups = next
	return saveGroupRegistry(registry)
}

func GroupRegistryPath() string {
	return filepath.Join(MaquettesDir(), "groups.json")
}

func loadGroupRegistry() (groupRegistry, error) {
	path := GroupRegistryPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return groupRegistry{Groups: []MaquetteGroup{}}, nil
	}
	if err != nil {
		return groupRegistry{}, err
	}
	var registry groupRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return groupRegistry{}, err
	}
	if registry.Groups == nil {
		registry.Groups = []MaquetteGroup{}
	}
	return registry, nil
}

func saveGroupRegistry(registry groupRegistry) error {
	if err := os.MkdirAll(MaquettesDir(), 0755); err != nil {
		return err
	}
	sortGroups(registry.Groups)
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GroupRegistryPath(), append(payload, '\n'), 0644)
}

func sortGroups(groups []MaquetteGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
	})
}

func safeDirName(value string) string {
	name := strings.TrimSpace(value)
	name = strings.NewReplacer("\\", "-", "/", "-", ":", "", "*", "", "?", "", `"`, "", "<", "", ">", "", "|", "").Replace(name)
	name = strings.Join(strings.Fields(name), "_")
	name = strings.TrimRight(name, ". ")
	name = strings.TrimSpace(name)
	if name == "" {
		return "sans-nom"
	}
	return name
}

func sameCleanPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(filepath.Clean(left))
	rightAbs, rightErr := filepath.Abs(filepath.Clean(right))
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	return strings.EqualFold(left, right)
}

func stringParam(params map[string]any, key string) string {
	value, _ := params[key].(string)
	return value
}

func boolParam(params map[string]any, key string) bool {
	value, _ := params[key].(bool)
	return value
}

func numberParam(params map[string]any, key string) any {
	switch value := params[key].(type) {
	case int:
		return value
	case int64:
		return value
	case float64:
		if value == float64(int64(value)) {
			return int64(value)
		}
		return value
	case json.Number:
		if integer, err := value.Int64(); err == nil {
			return integer
		}
		if decimal, err := value.Float64(); err == nil {
			return decimal
		}
	}
	return params[key]
}
