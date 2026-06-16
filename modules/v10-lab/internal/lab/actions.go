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
				{Name: "licensed_plant_id", Label: "Usine déclarée dans la licence", Type: "string", Required: true, Default: "Usine1"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "address_name", Label: "Nom adresse", Type: "string", Default: ""},
				{Name: "address_street", Label: "Rue", Type: "string", Default: ""},
				{Name: "address_postalcode", Label: "Code postal", Type: "string", Default: ""},
				{Name: "address_town", Label: "Ville", Type: "string", Default: ""},
				{Name: "address_country", Label: "Pays", Type: "string", Default: ""},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1, Min: 1},
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
				{Name: "plant_id", Label: "ID de l'usine", Type: "number", Required: true, Default: 1},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "is_unload_form_mandatory", Label: "Activer le formulaire de déchargement", Type: "bool", Default: false},
				{Name: "unload_form_id", Label: "ID du formulaire de déchargement", Type: "number", Default: 1, HiddenWhen: map[string]any{"is_unload_form_mandatory": false}},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1, Min: 1},
			},
			Execute: ExecuteCreateWorkshop(),
		},
		{
			ID:          "create-machine-group",
			Label:       "Créer groupe de machine",
			Description: "Crée un groupe de machine Gedix via l'API entreprise.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom du groupe", Type: "string", Required: true, Default: "Groupe1"},
				{Name: "workshop_id", Label: "ID de l'atelier", Type: "number", Required: true, Default: 1, Min: 1},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "chars_eol_default", Label: "Charactère EOL/Défaut", Type: "string", Default: "13,10"},
				{Name: "is_job_name_auto", Label: "Nommage automatique des dossiers CN", Type: "bool", Default: false},
				{Name: "job_name_auto_template", Label: "Template de nommage automatique", Type: "string", Default: "", HiddenWhen: map[string]any{"is_job_name_auto": false}},
				{Name: "job_name_auto_next_number", Label: "Prochain valeur à venir", Type: "number", Default: 0, HiddenWhen: map[string]any{"is_job_name_auto": false}},
				{Name: "is_auto_loading", Label: "Chargement automatique", Type: "bool", Default: false},
				{Name: "target_name_auto_load", Label: "Cible CN (chargement auto)", Type: "string", Default: "", HiddenWhen: map[string]any{"is_auto_loading": false}},
				{Name: "is_operator_instructions_displayed", Label: "Afficher instructions opérateur", Type: "bool", Default: false},
				{Name: "operator_instructions", Label: "Instructions opérateur", Type: "text", Default: "", HiddenWhen: map[string]any{"is_operator_instructions_displayed": false}},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1, Min: 1},
			},
			Execute: ExecuteCreateMachineGroup(),
		},
		{
			ID:          "create-target",
			Label:       "Créer cible",
			Description: "Crée une cible DNC Gedix via l'API DNC.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom de la cible", Type: "string", Required: true, Default: "cible2"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "connector_name", Label: "Connector", Type: "string", Required: true, OptionsSource: "connectors"},
				{Name: "tunnel_steps", Label: "Etapes réseau", Type: "object[]", Default: []any{}, ItemFields: []ActionField{
					{Name: "entity_name", Label: "Nom relais", Type: "string"},
					{Name: "rank", Label: "Ordre", Type: "number"},
				}},
				{Name: "configs", Label: "Cibles config", Type: "object[]", Default: []any{}, UniqueItemField: "module_key", ItemFields: []ActionField{
					{Name: "module_key", Label: "Clef", Type: "string", Options: []ActionOption{
						{Label: "remote-filepath", Value: "remote-filepath"},
						{Label: "subprogram-filepath", Value: "subprogram-filepath"},
					}},
					{Name: "module_value", Label: "Valeur", Type: "string"},
				}},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1, Min: 1},
			},
			Execute: ExecuteCreateTarget(),
		},
		{
			ID:          "create-machine",
			Label:       "Créer machine",
			Description: "Crée une machine Gedix via l'API entreprise.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom de la machine", Type: "string", Required: true, Default: "Machine"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "target_name", Label: "Cible CN", Type: "string", Default: ""},
				{Name: "is_confirm_deletion_before_load_disabled", Label: "Désactiver la confirmation lors du chargement", Type: "bool", Default: false},
				{Name: "target_name_load", Label: "Cible CN (chargement programme usinage)", Type: "string", Default: ""},
				{Name: "target_name_unload", Label: "Cible CN (déchargement programme usinage)", Type: "string", Default: ""},
				{Name: "target_name_mazak_matrix_file_mazak", Label: "Cible Mazak Matrix (fichier .maz)", Type: "string", Default: ""},
				{Name: "target_name_mazak_matrix_file_layout", Label: "Cible Mazak Matrix (fichier .lay)", Type: "string", Default: ""},
				{Name: "target_name_mazak_matrix_file_setup", Label: "Cible Mazak Matrix (fichier .stp)", Type: "string", Default: ""},
				{Name: "target_name_presetting_program", Label: "Cible préréglage", Type: "string", Default: ""},
				{Name: "target_name_probe_file", Label: "Cible relevés de cotes", Type: "string", Default: ""},
				{Name: "chars_eol", Label: "Charactères EOL", Type: "string", Default: "13,10"},
				{Name: "has_command_program", Label: "Possède un programme de commande", Type: "bool", Default: false},
				{Name: "wait_between_command_program_check_seconds", Label: "Intervalle de vérification (s)", Type: "number", Default: 30, HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_regexp", Label: "Expression régulière de la commande", Type: "string", Default: "", HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_regexp_load_value", Label: "Commande de chargement", Type: "string", Default: "", HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_regexp_unload_value", Label: "Commande de déchargement", Type: "string", Default: "", HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_name", Label: "Nom du programme de commande", Type: "string", Default: "", HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "target_name_command_program", Label: "Cible CN (programme de commande)", Type: "string", Default: "", HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_error_template_id", Label: "ID du rapport d'erreur", Type: "number", Default: 0, HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "is_command_program_ignored", Label: "Désactiver temporairement", Type: "bool", Default: false, HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "command_program_wait_before_load_seconds", Label: "Délai avant chargement (s)", Type: "number", Default: 0, HiddenWhen: map[string]any{"has_command_program": false}},
				{Name: "dnc_port_type", Label: "Type de port DNC", Type: "string", Default: "ethernet", Options: []ActionOption{{Label: "ethernet", Value: "ethernet"}, {Label: "serial", Value: "serial"}}},
				{Name: "is_root_browsing_allowed", Label: "Activer liste mémoire CN", Type: "bool", Default: true, HiddenWhen: map[string]any{"dnc_port_type": "serial"}},
				{Name: "target_name_root", Label: "Cible CN (racine)", Type: "string", Default: "", HiddenWhenAny: []map[string]any{{"dnc_port_type": "serial"}, {"is_root_browsing_allowed": false}}},
				{Name: "is_file_deletion_allowed", Label: "Autoriser la suppression des fichiers", Type: "bool", Default: true, HiddenWhenAny: []map[string]any{{"dnc_port_type": "serial"}, {"is_root_browsing_allowed": false}}},
				{Name: "is_file_viewing_allowed", Label: "Autoriser la visualisation des fichier", Type: "bool", Default: true, HiddenWhenAny: []map[string]any{{"dnc_port_type": "serial"}, {"is_root_browsing_allowed": false}}},
				{Name: "machine_group_ids", Label: "IDs des groupes de machines", Type: "number[]", Default: []any{}, ItemMin: 1},
				{Name: "is_operator_instructions_displayed", Label: "Afficher les instructions opérateur", Type: "bool", Default: false},
				{Name: "operator_instructions", Label: "Instructions opérateur", Type: "text", Default: "", HiddenWhen: map[string]any{"is_operator_instructions_displayed": false}},
				{Name: "numerical_controls_parameter_id", Label: "ID du paramètres CN", Type: "number", Default: 0},
				{Name: "created_by", Label: "Créé par", Type: "number", Required: true, Default: 1, Min: 1},
			},
			Execute: ExecuteCreateMachine(),
		},
		{
			ID:          "create-machining-job-default-states",
			Label:       "Créer cycle de vie Dossier CN",
			Description: "Crée les états par défaut du cycle de vie Dossier CN.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields:      []ActionField{{Name: "lang", Label: "Langue", Type: "string", Default: "fr"}, {Name: "user_id", Label: "ID de l'utilisateur", Type: "number", Required: true, Default: 1, Min: 1}},
			Execute:     ExecuteCreateMachiningJobDefaultStates(),
		},
		{
			ID:          "create-presetting-program-default-states",
			Label:       "Créer cycle de vie préréglage",
			Description: "Crée les états par défaut du cycle de vie préréglage.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields:      []ActionField{{Name: "lang", Label: "Langue", Type: "string", Default: "fr"}, {Name: "user_id", Label: "ID de l'utilisateur", Type: "number", Required: true, Default: 1, Min: 1}},
			Execute:     ExecuteCreatePresettingProgramDefaultStates(),
		},
		{
			ID:          "create-document-default-states",
			Label:       "Créer cycle de vie documents",
			Description: "Crée les états par défaut du cycle de vie documents.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields:      []ActionField{{Name: "lang", Label: "Langue", Type: "string", Default: "fr"}, {Name: "user_id", Label: "ID de l'utilisateur", Type: "number", Required: true, Default: 1, Min: 1}},
			Execute:     ExecuteCreateDocumentDefaultStates(),
		},
		{
			ID:          "create-machining-job",
			Label:       "Créer dossier CN",
			Description: "Crée un dossier CN via l'API entreprise.",
			Kind:        KindAPI,
			Products:    []string{GedixProdV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom du dossier CN", Type: "string", Required: true},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "machine_group_ids", Label: "IDs des groupes de machines", Type: "number[]", Default: []any{}, ItemMin: 1},
				{Name: "version", Label: "Version", Type: "number", Default: 0},
				{Name: "user_id", Label: "ID de l'utilisateur", Type: "number", Required: true, Default: 1, Min: 1},
			},
			Execute: ExecuteCreateMachiningJob(),
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
