package modules

import "toolBox/pkg/modulecontract"

type Registry struct {
	modules map[string]modulecontract.ModuleInfo
}

func NewRegistry() *Registry {
	items := []modulecontract.ModuleInfo{
		{
			ID:          "test-sheet",
			Name:        "Fiches de test",
			Description: "Creation et traitement de fiches de test",
			Actions: []modulecontract.ModuleAction{
				{ID: "init-db", Name: "Initialiser la base", Description: "Prepare la base SQLite du module"},
				{ID: "list", Name: "Lister", Description: "Retourne les fiches disponibles"},
			},
		},
		{
			ID:          "test-env",
			Name:        "Maquettes de test",
			Description: "Installation et configuration de maquettes de test",
			Actions: []modulecontract.ModuleAction{
				{ID: "init-config", Name: "Initialiser la configuration", Description: "Cree un fichier de configuration local"},
				{ID: "validate", Name: "Valider", Description: "Verifie la configuration courante"},
			},
		},
	}

	registry := &Registry{modules: make(map[string]modulecontract.ModuleInfo, len(items))}
	for _, item := range items {
		registry.modules[item.ID] = item
	}
	return registry
}

func (r *Registry) List() []modulecontract.ModuleInfo {
	items := make([]modulecontract.ModuleInfo, 0, len(r.modules))
	for _, item := range r.modules {
		items = append(items, item)
	}
	return items
}

func (r *Registry) Get(id string) (modulecontract.ModuleInfo, bool) {
	item, ok := r.modules[id]
	return item, ok
}
