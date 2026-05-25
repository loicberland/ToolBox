package lab

import (
	"fmt"
	"sort"
	"strings"
)

const (
	GedixProdV10      = "gedix-prod-v10"
	GedixToolStockV10 = "gedix-tool-stock-v10"
	GedixWatchV10     = "gedix-watch-v10"
	LegacyGedixV10    = "gedix-v10"
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
		Name:           "Tool Stock V10",
		Label:          "Tool Stock V10",
		Description:    "Produit Tool Stock V10",
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
		ID:             GedixWatchV10,
		Name:           "Watch V10",
		Label:          "Watch V10",
		Description:    "Produit Watch V10",
		DefaultAppName: "watch",
		Services: []ProductServiceDefinition{
			{Name: "webserver", Label: "webserver", HasDatabase: false, SupportsExtraKeys: true},
			{Name: "datawarehouse", Label: "datawarehouse", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "auth", Label: "auth", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "filestore", Label: "filestore", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "m2m", Label: "m2m", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "entreprise", Label: "entreprise", HasDatabase: true, SupportsExtraKeys: true},
			{Name: "etl", Label: "etl", HasDatabase: true, SupportsExtraKeys: true}},
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
	if productID == LegacyGedixV10 {
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

func (p ProductDefinition) UnitModuleExecutableName(unitName string) string {
	pattern := strings.TrimSpace(p.UnitModuleExecutablePattern)
	if pattern == "" {
		pattern = "gx-module-<unitName>.exe"
	}
	return strings.ReplaceAll(pattern, "<unitName>", unitName)
}
