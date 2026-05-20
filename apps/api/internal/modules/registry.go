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
			ID:          "v10-lab",
			Name:        "V10 Lab",
			Description: "Generateur de maquettes V10",
			Actions: []modulecontract.ModuleAction{
				{ID: "products", Name: "Produits", Description: "Liste les produits supportes"},
				{ID: "actions", Name: "Actions", Description: "Liste les actions disponibles"},
				{ID: "validate", Name: "Valider", Description: "Valide une configuration JSON"},
				{ID: "run", Name: "Executer", Description: "Execute fictivement un pipeline"},
				{ID: "register", Name: "Enregistrer", Description: "Enregistre une maquette localement"},
				{ID: "list", Name: "Lister", Description: "Liste les maquettes enregistrees"},
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
