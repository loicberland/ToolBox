package lab

import (
	"fmt"
	"sort"
	"strings"
)

const (
	GedixProdV10         = "gedix-prod-v10"
	GedixToolStockV10    = "gedix-tool-stock-v10"
	GedixWatchV10        = "gedix-watch-v10"
	GedixViewer          = "gedix-viewer"
	GedixViewerSamson826 = "gedix-viewer-samson826"
	GedixAcspcV10        = "gedix-acspc-v10"
	GedixAcassV10        = "gedix-acass-v10"
	GedixLegacySecure    = "gedix-legacy-secure"
)

type UnitKind string

const (
	UnitKindConnector UnitKind = "connector"
	UnitKindAgent     UnitKind = "agent"
	UnitKindAdaptor   UnitKind = "adaptor"
)

type ProductServiceDefinition struct {
	Name              string `json:"name"`
	Label             string `json:"label"`
	HasDatabase       bool   `json:"hasDatabase"`
	SupportsExtraKeys bool   `json:"supportsExtraKeys"`
}

type ProductUnitDefinition struct {
	Kind                     UnitKind `json:"kind"`
	SingularLabel            string   `json:"singularLabel"`
	PluralLabel              string   `json:"pluralLabel"`
	CfgSectionName           string   `json:"cfgSectionName"`
	FolderPrefix             string   `json:"folderPrefix"`
	RuntimeExecutablePattern string   `json:"runtimeExecutablePattern"`
	ModuleExecutablePattern  string   `json:"moduleExecutablePattern"`
}

type ProductDefinition struct {
	ID                           string                     `json:"id"`
	Name                         string                     `json:"name"`
	Label                        string                     `json:"label"`
	Description                  string                     `json:"description"`
	DefaultAppName               string                     `json:"defaultAppName"`
	Services                     []ProductServiceDefinition `json:"services"`
	UnitKind                     UnitKind                   `json:"unitKind"`
	UnitSingularLabel            string                     `json:"unitSingularLabel"`
	UnitPluralLabel              string                     `json:"unitPluralLabel"`
	UnitCfgSectionName           string                     `json:"unitCfgSectionName"`
	UnitFolderPrefix             string                     `json:"unitFolderPrefix"`
	UnitRuntimeExecutablePattern string                     `json:"unitRuntimeExecutablePattern"`
	UnitModuleExecutablePattern  string                     `json:"unitModuleExecutablePattern"`
	UnitDefinitions              []ProductUnitDefinition    `json:"unitDefinitions,omitempty"`
}

type Product = ProductDefinition

var productRegistry = []ProductDefinition{
	{
		ID:             GedixAcassV10,
		Name:           "Gedix Acass V10",
		Label:          "Gedix Acass V10",
		Description:    "Produit Gedix Acass V10",
		DefaultAppName: "acass",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "printer", Label: "printer", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     UnitKindConnector,
		UnitSingularLabel:            "connector",
		UnitPluralLabel:              "connectors",
		UnitCfgSectionName:           "connectors",
		UnitFolderPrefix:             "connector-",
		UnitRuntimeExecutablePattern: "gx-connector.exe",
		UnitModuleExecutablePattern:  "",
	},
	{
		ID:             GedixAcspcV10,
		Name:           "Gedix Acspc V10",
		Label:          "Gedix Acspc V10",
		Description:    "Produit Gedix Acspc V10",
		DefaultAppName: "acspc",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "printer", Label: "printer", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "etl", Label: "etl", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     UnitKindConnector,
		UnitSingularLabel:            "connector",
		UnitPluralLabel:              "connectors",
		UnitCfgSectionName:           "connectors",
		UnitFolderPrefix:             "connector-",
		UnitRuntimeExecutablePattern: "gx-connector.exe",
		UnitModuleExecutablePattern:  "",
	},
	{
		ID:             GedixProdV10,
		Name:           "Gedix Prod V10",
		Label:          "Gedix Prod V10",
		Description:    "Produit Gedix Prod V10",
		DefaultAppName: "prod",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "etl", Label: "etl", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "dnc", Label: "dnc", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "reactor", Label: "reactor", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "config", Label: "config", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     UnitKindConnector,
		UnitSingularLabel:            "connector",
		UnitPluralLabel:              "connectors",
		UnitCfgSectionName:           "connectors",
		UnitFolderPrefix:             "connector-",
		UnitRuntimeExecutablePattern: "gx-connector.exe",
		UnitModuleExecutablePattern:  "gx-module-<moduleName>.exe",
	},
	{
		ID:                           GedixLegacySecure,
		Name:                         "Gedix Legacy Secure",
		Label:                        "Gedix Legacy Secure",
		Description:                  "Produit Gedix Legacy Secure",
		DefaultAppName:               "legacy_secure",
		Services:                     []ProductServiceDefinition{},
		UnitKind:                     UnitKindConnector,
		UnitSingularLabel:            "connector",
		UnitPluralLabel:              "connectors",
		UnitCfgSectionName:           "connectors",
		UnitFolderPrefix:             "connector-",
		UnitRuntimeExecutablePattern: "gx-connector.exe",
		UnitModuleExecutablePattern:  "gx-module-<moduleName>.exe",
		UnitDefinitions: []ProductUnitDefinition{
			{
				Kind:                     UnitKindConnector,
				SingularLabel:            "connector",
				PluralLabel:              "connectors",
				CfgSectionName:           "connectors",
				FolderPrefix:             "connector-",
				RuntimeExecutablePattern: "gx-connector.exe",
				ModuleExecutablePattern:  "gx-module-<moduleName>.exe",
			},
			{
				Kind:                     UnitKindAdaptor,
				SingularLabel:            "adaptor",
				PluralLabel:              "adaptors",
				CfgSectionName:           "adaptors",
				FolderPrefix:             "adaptor-",
				RuntimeExecutablePattern: "gx-adaptor-<moduleName>.exe",
				ModuleExecutablePattern:  "",
			},
		},
	},
	{
		ID:             GedixToolStockV10,
		Name:           "Gedix Tool Stock V10",
		Label:          "Gedix Tool Stock V10",
		Description:    "Produit Gedix Tool Stock V10",
		DefaultAppName: "tool_stock",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "etl", Label: "etl", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "config", Label: "config", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     UnitKindConnector,
		UnitSingularLabel:            "connector",
		UnitPluralLabel:              "connectors",
		UnitCfgSectionName:           "connectors",
		UnitFolderPrefix:             "connector-",
		UnitRuntimeExecutablePattern: "<moduleName>.exe",
		UnitModuleExecutablePattern:  "<moduleName>.exe",
	},
	{
		ID:             GedixViewer,
		Name:           "Gedix Viewer",
		Label:          "Gedix Viewer",
		Description:    "Produit Gedix Viewer",
		DefaultAppName: "viewer",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "legacy", Label: "legacy", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     "",
		UnitSingularLabel:            "",
		UnitPluralLabel:              "",
		UnitCfgSectionName:           "",
		UnitFolderPrefix:             "",
		UnitRuntimeExecutablePattern: "",
		UnitModuleExecutablePattern:  "",
	},
	{
		ID:             GedixViewerSamson826,
		Name:           "Gedix Viewer Samson826",
		Label:          "Gedix Viewer Samson826",
		Description:    "Produit Gedix Viewer Samson826",
		DefaultAppName: "viewerSamson826",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "legacy", Label: "legacy", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     "",
		UnitSingularLabel:            "",
		UnitPluralLabel:              "",
		UnitCfgSectionName:           "",
		UnitFolderPrefix:             "",
		UnitRuntimeExecutablePattern: "",
		UnitModuleExecutablePattern:  "",
	},
	{
		ID:             GedixWatchV10,
		Name:           "Gedix Watch V10",
		Label:          "Gedix Watch V10",
		Description:    "Produit Gedix Watch V10",
		DefaultAppName: "watch",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "datawarehouse", Label: "datawarehouse", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "m2m", Label: "m2m", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "etl", Label: "etl", HasDatabase: true, SupportsExtraKeys: true},
		},
		UnitKind:                     UnitKindAgent,
		UnitSingularLabel:            "agent",
		UnitPluralLabel:              "agents",
		UnitCfgSectionName:           "agents",
		UnitFolderPrefix:             "agent-",
		UnitRuntimeExecutablePattern: "gx-agent.exe",
		UnitModuleExecutablePattern:  "gx-module-<moduleName>.exe",
	},
}

func Products() []Product {
	items := append([]ProductDefinition{}, productRegistry...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func ProductDefinitionByID(productID string) (ProductDefinition, error) {
	productID = NormalizeProductID(productID)
	for _, product := range productRegistry {
		if product.ID == productID {
			return product, nil
		}
	}
	return ProductDefinition{}, fmt.Errorf("produit inconnu %q", productID)
}

func ProductExists(productID string) bool {
	_, err := ProductDefinitionByID(productID)
	return err == nil
}

func NormalizeProductID(productID string) string {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return GedixProdV10
	}
	return productID
}

func (p ProductDefinition) Service(name string) (ProductServiceDefinition, bool) {
	for _, service := range p.Services {
		if strings.EqualFold(service.Name, name) {
			return service, true
		}
	}
	return ProductServiceDefinition{}, false
}

func (p ProductDefinition) HasUnits() bool {
	return len(p.UnitDefinitionsForProduct()) > 0
}

func (p ProductDefinition) SupportsModuleCommand() bool {
	for _, definition := range p.UnitDefinitionsForProduct() {
		if strings.TrimSpace(definition.ModuleExecutablePattern) != "" {
			return true
		}
	}
	return false
}

func unitArticle(product ProductDefinition) string {
	label := strings.TrimSpace(product.PrimaryUnitDefinition().SingularLabel)
	if label == "" {
		label = "connector"
	}
	first := strings.ToLower(label[:1])
	if strings.ContainsAny(first, "aeiouh") {
		return "l'" + label
	}
	return "le " + label
}

func unitDefinitionArticle(definition ProductUnitDefinition) string {
	label := strings.TrimSpace(definition.SingularLabel)
	if label == "" {
		label = "unite"
	}
	first := strings.ToLower(label[:1])
	if strings.ContainsAny(first, "aeiouh") {
		return "l'" + label
	}
	return "le " + label
}

func (p ProductDefinition) UnitDefinitionsForProduct() []ProductUnitDefinition {
	if len(p.UnitDefinitions) > 0 {
		return append([]ProductUnitDefinition{}, p.UnitDefinitions...)
	}
	if (p.UnitKind == UnitKindConnector || p.UnitKind == UnitKindAgent || p.UnitKind == UnitKindAdaptor) && strings.TrimSpace(p.UnitCfgSectionName) != "" {
		return []ProductUnitDefinition{p.PrimaryUnitDefinition()}
	}
	return []ProductUnitDefinition{}
}

func (p ProductDefinition) PrimaryUnitDefinition() ProductUnitDefinition {
	return ProductUnitDefinition{
		Kind:                     p.UnitKind,
		SingularLabel:            p.UnitSingularLabel,
		PluralLabel:              p.UnitPluralLabel,
		CfgSectionName:           p.UnitCfgSectionName,
		FolderPrefix:             p.UnitFolderPrefix,
		RuntimeExecutablePattern: p.UnitRuntimeExecutablePattern,
		ModuleExecutablePattern:  p.UnitModuleExecutablePattern,
	}
}

func (p ProductDefinition) UnitDefinition(kind UnitKind) (ProductUnitDefinition, bool) {
	for _, definition := range p.UnitDefinitionsForProduct() {
		if definition.Kind == kind {
			return definition, true
		}
	}
	return ProductUnitDefinition{}, false
}

func (p ProductDefinition) UnitModuleExecutableName(moduleName string) string {
	return ResolveUnitModuleExecutable(p, p.PrimaryUnitDefinition(), "", ProductUnitConfig{Module: moduleName})
}

func (p ProductDefinition) UnitRuntimeExecutableName(unitName string, moduleName string) string {
	return ResolveUnitRuntimeExecutable(p, p.PrimaryUnitDefinition(), unitName, ProductUnitConfig{Module: moduleName})
}

func ResolveUnitRuntimeExecutable(_ ProductDefinition, definition ProductUnitDefinition, unitName string, unitConfig ProductUnitConfig) string {
	pattern := strings.TrimSpace(definition.RuntimeExecutablePattern)
	if pattern == "" {
		pattern = "gx-connector.exe"
	}
	return renderUnitExecutablePattern(pattern, unitName, unitConfig.Module)
}

func ResolveUnitModuleExecutable(_ ProductDefinition, definition ProductUnitDefinition, unitName string, unitConfig ProductUnitConfig) string {
	moduleName := NormalizeModuleType(unitConfig.Module)
	if moduleName == "digi-legacy" && strings.TrimSpace(definition.ModuleExecutablePattern) == "gx-module-<moduleName>.exe" {
		return "gx-connector.exe"
	}
	pattern := strings.TrimSpace(definition.ModuleExecutablePattern)
	if pattern == "" {
		pattern = "gx-module-<unitName>.exe"
	}
	return renderUnitExecutablePattern(pattern, unitName, unitConfig.Module)
}

func renderUnitExecutablePattern(pattern string, unitName string, moduleRaw string) string {
	moduleRaw = trimModuleRaw(moduleRaw)
	moduleName := NormalizeModuleType(moduleRaw)
	replacer := strings.NewReplacer(
		"<unitName>", strings.TrimSpace(unitName),
		"<moduleName>", moduleName,
		"<moduleRaw>", moduleRaw,
	)
	return replacer.Replace(strings.TrimSpace(pattern))
}

func patternRequiresModuleName(pattern string) bool {
	return strings.Contains(strings.TrimSpace(pattern), "<moduleName>")
}

func trimModuleRaw(rawType string) string {
	value := strings.TrimSpace(rawType)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	return strings.TrimSpace(value)
}

func NormalizeModuleType(rawType string) string {
	value := trimModuleRaw(rawType)
	if strings.HasPrefix(strings.ToLower(value), "module-") {
		value = value[len("module-"):]
	}
	return value
}
