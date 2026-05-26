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
		// {
		// 	ID:          "stop-maquette",
		// 	Label:       "Arrêter maquette",
		// 	Description: "Indique comment arrêter manuellement la maquette.",
		// 	Kind:        KindSystem,
		// 	Products:    []string{},
		// 	Execute: func(ctx ActionContext, params map[string]any) error {
		// 		fmt.Fprintln(ctx.Writer, "Fermez les fenêtres gx-front/gx-app ouvertes pour arrêter la maquette.")
		// 		fmt.Fprintln(ctx.Writer, "Pour tuer radicalement les processus, utilisez v10-lab kill-gx-processes --force.")
		// 		return nil
		// 	},
		// },
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
		// {
		// 	ID:          "stop-services",
		// 	Label:       "Arrêter services",
		// 	Description: "Alias compatibilité: utilisez stop-maquette.",
		// 	Kind:        KindSystem,
		// 	Products:    []string{},
		// 	Fields: []ActionField{
		// 		{Name: "taskkill", Label: "Forcer la coupure GX", Type: "bool", Default: false},
		// 	},
		// 	Execute: func(ctx ActionContext, params map[string]any) error {
		// 		fmt.Fprintln(ctx.Writer, "[ALIAS] stop-services redirige vers stop-maquette.")
		// 		fmt.Fprintln(ctx.Writer, "Fermez les fenêtres gx-front/gx-app ouvertes pour arrêter la maquette.")
		// 		return nil
		// 	},
		// },
		{
			ID:          "gedix-api-test",
			Label:       "Test API Gedix",
			Description: "Exécute une requête HTTP de test vers l'API Gedix.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "method", Label: "Méthode", Type: "string", Default: "GET"},
				{Name: "path", Label: "Chemin API", Type: "string", Required: true, Default: "/api/health"},
				{Name: "bodyJson", Label: "Body JSON", Type: "text", Description: "Optionnel, utilisé pour POST/PUT/PATCH"},
				{Name: "printResponseBody", Label: "Afficher la réponse", Type: "bool", Default: true},
			},
			Execute: ExecuteGedixAPITest(),
		},
		// {
		// 	ID:          "create-machine-group",
		// 	Label:       "Créer groupe machine",
		// 	Description: "Crée fictivement un groupe machine Gedix V10.",
		// 	Kind:        KindAPI,
		// 	Products:    []string{GedixProdV10},
		// 	Hidden:      true,
		// 	Fields: []ActionField{
		// 		{Name: "code", Label: "Code", Type: "string", Required: true},
		// 		{Name: "name", Label: "Nom", Type: "string", Required: true},
		// 	},
		// 	Execute: func(ctx ActionContext, params map[string]any) error {
		// 		fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer groupe machine %s / %s\n", stringParam(params, "code"), stringParam(params, "name"))
		// 		return nil
		// 	},
		// },
		// {
		// 	ID:          "create-machine",
		// 	Label:       "Créer machine",
		// 	Description: "Crée fictivement une machine Gedix V10.",
		// 	Kind:        KindAPI,
		// 	Products:    []string{GedixProdV10},
		// 	Hidden:      true,
		// 	Fields: []ActionField{
		// 		{Name: "code", Label: "Code", Type: "string", Required: true},
		// 		{Name: "name", Label: "Nom", Type: "string", Required: true},
		// 		{Name: "groupCode", Label: "Groupe", Type: "string"},
		// 	},
		// 	Execute: func(ctx ActionContext, params map[string]any) error {
		// 		fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer machine %s / %s\n", stringParam(params, "code"), stringParam(params, "name"))
		// 		return nil
		// 	},
		// },
		// {
		// 	ID:          "create-cnc-folder",
		// 	Label:       "Créer dossier CN",
		// 	Description: "Crée fictivement un dossier CN Gedix V10.",
		// 	Kind:        KindAPI,
		// 	Products:    []string{GedixProdV10},
		// 	Hidden:      true,
		// 	Fields: []ActionField{
		// 		{Name: "machineGroupCode", Label: "Groupe machine", Type: "string", Required: true},
		// 		{Name: "programCode", Label: "Programme", Type: "string", Required: true},
		// 		{Name: "programIndex", Label: "Indice", Type: "string", Default: "A"},
		// 	},
		// 	Execute: func(ctx ActionContext, params map[string]any) error {
		// 		fmt.Fprintf(ctx.Writer, "[DRY-RUN] Créer dossier CN %s indice %s pour groupe %s\n", stringParam(params, "programCode"), stringParam(params, "programIndex"), stringParam(params, "machineGroupCode"))
		// 		return nil
		// 	},
		// },
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
