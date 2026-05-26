package lab

import "fmt"

func Actions() []Action {
	return []Action{
		{
			ID:          "create-env",
			Label:       "Créer maquette",
			Description: "Crée une maquette Gedix V10 depuis un ZIP de release.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "zipPath", Label: "ZIP release", Type: "string", Description: "Optionnel si release.zipPath est renseigné"},
				{Name: "workDir", Label: "Dossier de travail", Type: "string"},
				{Name: "overwrite", Label: "Écraser", Type: "bool", Default: false},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				return CreateEnv(ctx, params)
			},
		},
		{
			ID:          "configure-gedix-cfg",
			Label:       "Configurer gedix.cfg",
			Description: "Modifie les paramètres principaux du fichier gedix.cfg de la maquette.",
			Kind:        KindSystem,
			Products:    []string{},
			Execute: func(ctx ActionContext, params map[string]any) error {
				return ConfigureGedixCfg(ctx.Config, ctx.Writer)
			},
		},
		{
			ID:          "start-maquette",
			Label:       "Démarrer maquette",
			Description: "Démarre gx-front, gx-app et les cibles debug configurées.",
			Kind:        KindSystem,
			Products:    []string{},
			Execute: func(ctx ActionContext, params map[string]any) error {
				return StartMaquette(ctx.Config, ctx.Writer)
			},
		},
		{
			ID:          "kill-gx-processes",
			Label:       "Couper les services GX",
			Description: "Coupe manuellement tous les processus GX en cours sur cette machine.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "force", Label: "Forcer", Type: "bool", Default: false},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				return KillGXProcesses(ctx.Writer, boolParam(params, "force"), false)
			},
		},
		{
			ID:          "update-env",
			Label:       "Mettre à jour maquette",
			Description: "Met à jour les fichiers applicatifs de la maquette depuis une nouvelle release, sans écraser la configuration ni les logs.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "zipPath", Label: "ZIP release", Type: "string"},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				return UpdateEnv(ctx, params)
			},
		},
		{
			ID:          "start-services",
			Label:       "Démarrer services",
			Description: "Alias compatibilité: utilisez start-maquette.",
			Kind:        KindSystem,
			Products:    []string{},
			Fields: []ActionField{
				{Name: "debugServices", Label: "Services debug", Type: "string[]"},
				{Name: "services", Label: "Services", Type: "string[]"},
			},
			Execute: func(ctx ActionContext, params map[string]any) error {
				fmt.Fprintln(ctx.Writer, "[ALIAS] start-services redirige vers start-maquette.")
				return StartMaquette(ctx.Config, ctx.Writer)
			},
		},
		{
			ID:          "create-plant",
			Label:       "Créer une usine",
			Description: "Crée une usine Gedix via l'API entreprise.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom de l'usine", Type: "string", Required: true, Default: "Usine"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "address_name", Label: "Nom adresse", Type: "string", Default: ""},
				{Name: "address_street", Label: "Rue", Type: "string", Default: ""},
				{Name: "address_postalcode", Label: "Code postal", Type: "string", Default: ""},
				{Name: "address_town", Label: "Ville", Type: "string", Default: ""},
				{Name: "address_country", Label: "Pays", Type: "string", Default: ""},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1},
			},
			Execute: ExecuteCreatePlant(),
		},
		{
			ID:          "create-workshop",
			Label:       "Créer un atelier",
			Description: "Crée un atelier Gedix via l'API entreprise.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom de l'atelier", Type: "string", Required: true, Default: "Atelier"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "plant_id", Label: "Usine ID", Type: "number", Required: true, Default: 1},
				{Name: "is_unload_form_mandatory", Label: "Activer le formulaire de déchargement", Type: "bool", Default: false},
				{Name: "unload_form_id", Label: "ID du formulaire de déchargement", Type: "number", Default: 1},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1},
			},
			Execute: ExecuteCreateWorkshop(),
		},
	}
}

func ActionsForProduct(productID string) []Action {
	productID = NormalizeProductID(productID)
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
	productID = NormalizeProductID(productID)
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
