package lab

import "fmt"

func Actions() []Action {
	actions := []Action{
		{
			ID:          "install-env",
			Label:       "Installer maquette",
			Description: "Installe une maquette Gedix V10 depuis un ZIP de release.",
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
	}
	actions = append(actions, prodAPIActions()...)
	actions = append(actions, watchAPIActions()...)
	return actions
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

func FindActionForProduct(actionID string, productID string) (Action, bool) {
	productID = NormalizeProductID(productID)
	if productID != "" {
		for _, action := range Actions() {
			if action.ID == actionID && action.SupportsProduct(productID) {
				return action, true
			}
		}
	}
	return FindAction(actionID)
}

func canonicalActionID(actionID string) string {
	switch actionID {
	case "create-watch-statusvalue-category":
		return "create-statusvalue-category"
	case "create-watch-statusvalue":
		return "create-statusvalue"
	case "create-watch-acquisition-variable":
		return "create-acquisition-variable"
	case "create-watch-calculated-variable":
		return "create-calculated-variable"
	case "create-watch-script":
		return "create-script"
	case "create-watch-target":
		return "create-target"
	case "create-watch-agent":
		return "create-agent"
	default:
		return actionID
	}
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
	pruneHiddenActionFieldValues(action.Fields, next, nil)
	for _, field := range action.Fields {
		if _, exists := next[field.Name]; !exists && field.Default != nil && !actionFieldHidden(field, actionFieldVisibilityParams(action.Fields, next, nil)) {
			next[field.Name] = field.Default
		}
	}
	return next
}

func normalizeActionParamsForSave(action Action, params map[string]any) map[string]any {
	next := map[string]any{}
	for key, value := range params {
		next[key] = value
	}
	pruneHiddenActionFieldValues(action.Fields, next, nil)
	return next
}

func normalizePipelineStepsForSave(steps []PipelineStep) []PipelineStep {
	return normalizePipelineStepsForProductSave(steps, "")
}

func normalizePipelineStepsForProductSave(steps []PipelineStep, productID string) []PipelineStep {
	next := clonePipelineSteps(steps)
	for index := range next {
		next[index].Action = canonicalActionID(next[index].Action)
		action, ok := FindActionForProduct(next[index].Action, productID)
		if !ok {
			continue
		}
		next[index].Params = normalizeActionParamsForSave(action, next[index].Params)
	}
	return next
}

func normalizeConfigPipelineForSave(config *Config) {
	config.Pipeline = normalizePipelineStepsForProductSave(config.Pipeline, config.Product)
}

func pruneHiddenActionFieldValues(fields []ActionField, params map[string]any, parentParams map[string]any) {
	if params == nil {
		return
	}
	visibilityParams := actionFieldVisibilityParams(fields, params, parentParams)
	for _, field := range fields {
		if actionFieldHidden(field, visibilityParams) {
			delete(params, field.Name)
			continue
		}
		if field.Type == "object[]" && len(field.ItemFields) > 0 {
			if _, exists := params[field.Name]; !exists {
				continue
			}
			params[field.Name] = normalizeObjectArrayFieldValue(params[field.Name], field.ItemFields, visibilityParams)
		}
	}
}

func normalizeObjectArrayFieldValue(value any, itemFields []ActionField, parentParams map[string]any) any {
	switch rows := value.(type) {
	case []map[string]any:
		next := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			next = append(next, normalizeObjectArrayRow(row, itemFields, parentParams))
		}
		return next
	case []any:
		next := make([]any, 0, len(rows))
		for _, row := range rows {
			if typed, ok := row.(map[string]any); ok {
				next = append(next, normalizeObjectArrayRow(typed, itemFields, parentParams))
				continue
			}
			next = append(next, row)
		}
		return next
	default:
		return value
	}
}

func normalizeObjectArrayRow(row map[string]any, itemFields []ActionField, parentParams map[string]any) map[string]any {
	next := map[string]any{}
	for key, value := range row {
		next[key] = value
	}
	pruneHiddenActionFieldValues(itemFields, next, parentParams)
	return next
}

func actionFieldVisibilityParams(fields []ActionField, params map[string]any, parentParams map[string]any) map[string]any {
	next := map[string]any{}
	for key, value := range parentParams {
		next[key] = value
	}
	for key, value := range params {
		next[key] = value
	}
	for _, field := range fields {
		if _, exists := next[field.Name]; !exists && field.Default != nil {
			next[field.Name] = field.Default
		}
	}
	return next
}
