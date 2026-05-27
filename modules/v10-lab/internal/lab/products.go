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
)

type UnitKind string

const (
	UnitKindConnector UnitKind = "connector"
	UnitKindAgent     UnitKind = "agent"
)

type ProductServiceDefinition struct {
	Name              string `json:"name"`
	Label             string `json:"label"`
	HasDatabase       bool   `json:"hasDatabase"`
	SupportsExtraKeys bool   `json:"supportsExtraKeys"`
}

type ProductDefinition struct {
	ID                          string                     `json:"id"`
	Name                        string                     `json:"name"`
	Label                       string                     `json:"label"`
	Description                 string                     `json:"description"`
	DefaultAppName              string                     `json:"defaultAppName"`
	Services                    []ProductServiceDefinition `json:"services"`
	UnitKind                    UnitKind                   `json:"unitKind"`
	UnitSingularLabel           string                     `json:"unitSingularLabel"`
	UnitPluralLabel             string                     `json:"unitPluralLabel"`
	UnitCfgSectionName          string                     `json:"unitCfgSectionName"`
	UnitFolderPrefix            string                     `json:"unitFolderPrefix"`
	UnitExecutableName          string                     `json:"unitExecutableName"`
	UnitModuleExecutablePattern string                     `json:"unitModuleExecutablePattern"`
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
		UnitKind:                    UnitKindConnector,
		UnitSingularLabel:           "connecteur",
		UnitPluralLabel:             "connecteurs",
		UnitCfgSectionName:          "connectors",
		UnitFolderPrefix:            "connector-",
		UnitExecutableName:          "gx-connector.exe",
		UnitModuleExecutablePattern: "",
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
		UnitKind:                    UnitKindConnector,
		UnitSingularLabel:           "connecteur",
		UnitPluralLabel:             "connecteurs",
		UnitCfgSectionName:          "connectors",
		UnitFolderPrefix:            "connector-",
		UnitExecutableName:          "gx-connector.exe",
		UnitModuleExecutablePattern: "",
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
		UnitKind:                    UnitKindConnector,
		UnitSingularLabel:           "connecteur",
		UnitPluralLabel:             "connecteurs",
		UnitCfgSectionName:          "connectors",
		UnitFolderPrefix:            "connector-",
		UnitExecutableName:          "gx-connector.exe",
		UnitModuleExecutablePattern: "gx-module-<unitName>.exe",
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
		UnitKind:                    UnitKindConnector,
		UnitSingularLabel:           "connecteur",
		UnitPluralLabel:             "connecteurs",
		UnitCfgSectionName:          "connectors",
		UnitFolderPrefix:            "connector-",
		UnitExecutableName:          "gx-connector.exe",
		UnitModuleExecutablePattern: "gx-module-<unitName>.exe",
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
		UnitKind:                    "",
		UnitSingularLabel:           "",
		UnitPluralLabel:             "",
		UnitCfgSectionName:          "",
		UnitFolderPrefix:            "",
		UnitExecutableName:          "",
		UnitModuleExecutablePattern: "",
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
		UnitKind:                    "",
		UnitSingularLabel:           "",
		UnitPluralLabel:             "",
		UnitCfgSectionName:          "",
		UnitFolderPrefix:            "",
		UnitExecutableName:          "",
		UnitModuleExecutablePattern: "",
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
		UnitKind:                    UnitKindAgent,
		UnitSingularLabel:           "agent",
		UnitPluralLabel:             "agents",
		UnitCfgSectionName:          "agents",
		UnitFolderPrefix:            "agent-",
		UnitExecutableName:          "gx-agent.exe",
		UnitModuleExecutablePattern: "gx-module-<unitName>.exe",
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

func (p ProductDefinition) UnitModuleExecutableName(moduleName string) string {
	pattern := strings.TrimSpace(p.UnitModuleExecutablePattern)
	if pattern == "" {
		pattern = "gx-module-<unitName>.exe"
	}
	return strings.ReplaceAll(pattern, "<unitName>", NormalizeModuleType(moduleName))
}

func NormalizeModuleType(rawType string) string {
	value := strings.TrimSpace(rawType)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "module-") {
		value = value[len("module-"):]
	}
	return value
}
