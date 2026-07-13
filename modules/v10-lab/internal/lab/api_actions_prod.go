package lab

import (
	"fmt"
	"net/http"
	"strings"
)

func prodAPIActions() []Action {
	return []Action{
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
				{Name: "is_file_comparison_allowed", Label: "Autoriser la comparaison des fichier", Type: "bool", Default: true, HiddenWhenAny: []map[string]any{{"dnc_port_type": "serial"}, {"is_root_browsing_allowed": false}}},
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

const createPlantPath = "/entreprise/api/v1/plants"

type CreatePlantPayload struct {
	EntityName        string `json:"entity_name"`
	LicensedPlantID   string `json:"licensed_plant_id"`
	Description       string `json:"description"`
	AddressName       string `json:"address_name"`
	AddressStreet     string `json:"address_street"`
	AddressPostalCode string `json:"address_postalcode"`
	AddressTown       string `json:"address_town"`
	AddressCountry    string `json:"address_country"`
	CreatedBy         any    `json:"created_by"`
}

func ExecuteCreatePlant() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createPlantPayload(params)
		if err := client.CreatePlant(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Usine créée avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func (c *GedixAPIClient) CreatePlant(payload CreatePlantPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer une usine",
		Method:           http.MethodPost,
		Path:             createPlantPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createPlantPayload(params map[string]any) CreatePlantPayload {
	return CreatePlantPayload{
		EntityName:        stringParam(params, "entity_name"),
		LicensedPlantID:   stringParam(params, "licensed_plant_id"),
		Description:       stringParam(params, "description"),
		AddressName:       stringParam(params, "address_name"),
		AddressStreet:     stringParam(params, "address_street"),
		AddressPostalCode: stringParam(params, "address_postalcode"),
		AddressTown:       stringParam(params, "address_town"),
		AddressCountry:    stringParam(params, "address_country"),
		CreatedBy:         numberParam(params, "created_by"),
	}
}

const createWorkshopPath = "/entreprise/api/v1/workshops"

type CreateWorkshopPayload struct {
	EntityName            string `json:"entity_name"`
	Description           string `json:"description"`
	PlantID               any    `json:"plant_id"`
	IsUnloadFormMandatory bool   `json:"is_unload_form_mandatory"`
	UnloadFormID          any    `json:"unload_form_id"`
	CreatedBy             any    `json:"created_by"`
}

func ExecuteCreateWorkshop() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createWorkshopPayload(params)
		if err := client.CreateWorkshop(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Atelier créé avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func (c *GedixAPIClient) CreateWorkshop(payload CreateWorkshopPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer un atelier",
		Method:           http.MethodPost,
		Path:             createWorkshopPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createWorkshopPayload(params map[string]any) CreateWorkshopPayload {
	return CreateWorkshopPayload{
		EntityName:            stringParam(params, "entity_name"),
		Description:           stringParam(params, "description"),
		PlantID:               numberParam(params, "plant_id"),
		IsUnloadFormMandatory: boolParam(params, "is_unload_form_mandatory"),
		UnloadFormID:          numberParam(params, "unload_form_id"),
		CreatedBy:             numberParam(params, "created_by"),
	}
}

const createMachineGroupPath = "/entreprise/api/v1/machine_groups"

type CreateMachineGroupPayload struct {
	EntityName                      string `json:"entity_name"`
	Description                     string `json:"description"`
	CharsEOLDefault                 string `json:"chars_eol_default"`
	WorkshopID                      any    `json:"workshop_id"`
	OperatorInstructions            string `json:"operator_instructions"`
	MachineGroupsFiles              []any  `json:"machine_groups_files"`
	IsAutoLoading                   bool   `json:"is_auto_loading"`
	TargetNameAutoLoad              string `json:"target_name_auto_load"`
	IsJobNameAuto                   bool   `json:"is_job_name_auto"`
	JobNameAutoTemplate             string `json:"job_name_auto_template"`
	JobNameAutoNextNumber           any    `json:"job_name_auto_next_number"`
	IsOperatorInstructionsDisplayed bool   `json:"is_operator_instructions_displayed"`
	CreatedBy                       any    `json:"created_by"`
}

func ExecuteCreateMachineGroup() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createMachineGroupPayload(params)
		if err := client.CreateMachineGroup(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Groupe de machine créé avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func (c *GedixAPIClient) CreateMachineGroup(payload CreateMachineGroupPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer groupe de machine",
		Method:           http.MethodPost,
		Path:             createMachineGroupPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createMachineGroupPayload(params map[string]any) CreateMachineGroupPayload {
	return CreateMachineGroupPayload{
		EntityName:                      stringParam(params, "entity_name"),
		Description:                     stringParam(params, "description"),
		CharsEOLDefault:                 stringParam(params, "chars_eol_default"),
		WorkshopID:                      numberParam(params, "workshop_id"),
		OperatorInstructions:            stringParam(params, "operator_instructions"),
		MachineGroupsFiles:              []any{},
		IsAutoLoading:                   boolParam(params, "is_auto_loading"),
		TargetNameAutoLoad:              stringParam(params, "target_name_auto_load"),
		IsJobNameAuto:                   boolParam(params, "is_job_name_auto"),
		JobNameAutoTemplate:             stringParam(params, "job_name_auto_template"),
		JobNameAutoNextNumber:           numberParam(params, "job_name_auto_next_number"),
		IsOperatorInstructionsDisplayed: boolParam(params, "is_operator_instructions_displayed"),
		CreatedBy:                       numberParam(params, "created_by"),
	}
}

const createTargetPath = "/dnc/api/v1/targets"

type CreateTargetPayload struct {
	EntityName    string             `json:"entity_name"`
	Description   string             `json:"description"`
	ConnectorName string             `json:"connector_name"`
	Configs       []TargetConfig     `json:"configs"`
	TunnelSteps   []TargetTunnelStep `json:"tunnel_steps"`
	CreatedBy     any                `json:"created_by"`
}

type TargetConfig struct {
	ModuleValue string `json:"module_value"`
	ModuleKey   string `json:"module_key"`
}

type TargetTunnelStep struct {
	EntityName string `json:"entity_name"`
	Rank       any    `json:"rank,omitempty"`
}

func ExecuteCreateTarget() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		if err := validateTargetConfigs(params); err != nil {
			return err
		}
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createTargetPayload(params)
		if err := client.CreateTarget(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Cible créée avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func validateTargetConfigs(params map[string]any) error {
	allowed := map[string]bool{
		"remote-filepath":     true,
		"subprogram-filepath": true,
	}
	seen := map[string]bool{}
	for _, row := range objectArrayParam(params, "configs") {
		moduleKey := strings.TrimSpace(fmt.Sprint(row["module_key"]))
		if moduleKey == "" {
			continue
		}
		if !allowed[moduleKey] {
			return fmt.Errorf("la clé module %q n'est pas autorisée dans configs", moduleKey)
		}
		if seen[moduleKey] {
			return fmt.Errorf("la clé module %q est utilisée plusieurs fois dans configs", moduleKey)
		}
		seen[moduleKey] = true
	}
	return nil
}

func (c *GedixAPIClient) CreateTarget(payload CreateTargetPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer cible",
		Method:           http.MethodPost,
		Path:             createTargetPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createTargetPayload(params map[string]any) CreateTargetPayload {
	return CreateTargetPayload{
		EntityName:    stringParam(params, "entity_name"),
		Description:   stringParam(params, "description"),
		ConnectorName: stringParam(params, "connector_name"),
		Configs:       targetConfigsParam(params, "configs"),
		TunnelSteps:   targetTunnelStepsParam(params, "tunnel_steps"),
		CreatedBy:     numberParam(params, "created_by"),
	}
}

func targetConfigsParam(params map[string]any, key string) []TargetConfig {
	rows := objectArrayParam(params, key)
	items := []TargetConfig{}
	seen := map[string]bool{}
	for _, row := range rows {
		moduleKey := strings.TrimSpace(fmt.Sprint(row["module_key"]))
		if moduleKey == "" || seen[moduleKey] {
			continue
		}
		seen[moduleKey] = true
		items = append(items, TargetConfig{
			ModuleKey:   moduleKey,
			ModuleValue: strings.TrimSpace(fmt.Sprint(row["module_value"])),
		})
	}
	return items
}

func targetTunnelStepsParam(params map[string]any, key string) []TargetTunnelStep {
	rows := objectArrayParam(params, key)
	items := []TargetTunnelStep{}
	for index, row := range rows {
		entityName := strings.TrimSpace(fmt.Sprint(row["entity_name"]))
		if entityName == "" {
			continue
		}
		step := TargetTunnelStep{EntityName: entityName}
		if rank, ok := anyToInt(row["rank"]); ok && rank > 0 {
			step.Rank = rank
		} else {
			step.Rank = index + 1
		}
		items = append(items, step)
	}
	return items
}

const createMachinePath = "/entreprise/api/v1/machines"

type CreateMachinePayload struct {
	EntityName                            string                `json:"entity_name"`
	Description                           string                `json:"description"`
	CharsEOL                              string                `json:"chars_eol"`
	IsFileDeletionAllowed                 bool                  `json:"is_file_deletion_allowed"`
	IsFileViewingAllowed                  bool                  `json:"is_file_viewing_allowed"`
	IsFileComparisonAllowed               bool                  `json:"is_file_comparison_allowed"`
	IsRootBrowsingAllowed                 bool                  `json:"is_root_browsing_allowed"`
	TargetName                            string                `json:"target_name"`
	IsConfirmDeletionBeforeLoadDisabled   bool                  `json:"is_confirm_deletion_before_load_disabled"`
	TargetNameLoad                        string                `json:"target_name_load"`
	TargetNameUnload                      string                `json:"target_name_unload"`
	TargetNameMazakMatrixFileMazak        string                `json:"target_name_mazak_matrix_file_mazak"`
	TargetNameMazakMatrixFileLayout       string                `json:"target_name_mazak_matrix_file_layout"`
	TargetNameMazakMatrixFileSetup        string                `json:"target_name_mazak_matrix_file_setup"`
	TargetNamePresettingProgram           string                `json:"target_name_presetting_program"`
	TargetNameProbeFile                   string                `json:"target_name_probe_file"`
	DNCPortType                           string                `json:"dnc_port_type"`
	TargetNameRoot                        string                `json:"target_name_root"`
	MachineGroupsMachines                 []MachineGroupMachine `json:"machine_groups_machines"`
	OperatorInstructions                  string                `json:"operator_instructions"`
	MachinesFiles                         []any                 `json:"machines_files"`
	IsCommandProgramIgnored               bool                  `json:"is_command_program_ignored"`
	TargetNameCommandProgram              string                `json:"target_name_command_program"`
	CommandProgramErrorTemplateID         any                   `json:"command_program_error_template_id"`
	CommandProgramWaitBeforeLoadSeconds   any                   `json:"command_program_wait_before_load_seconds"`
	HasCommandProgram                     bool                  `json:"has_command_program"`
	CommandProgramName                    string                `json:"command_program_name"`
	WaitBetweenCommandProgramCheckSeconds any                   `json:"wait_between_command_program_check_seconds"`
	CommandProgramRegexp                  string                `json:"command_program_regexp"`
	CommandProgramRegexpLoadValue         string                `json:"command_program_regexp_load_value"`
	CommandProgramRegexpUnloadValue       string                `json:"command_program_regexp_unload_value"`
	IsOperatorInstructionsDisplayed       bool                  `json:"is_operator_instructions_displayed"`
	NumericalControlsParameterID          any                   `json:"numerical_controls_parameter_id"`
	CreatedBy                             any                   `json:"created_by"`
}

type MachineGroupMachine struct {
	MachineGroupID int `json:"machine_group_id"`
}

func ExecuteCreateMachine() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		if err := validateNumberListMin(params, "machine_group_ids", 1); err != nil {
			return err
		}
		payload := createMachinePayload(params)
		if err := client.CreateMachine(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Machine créée avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func (c *GedixAPIClient) CreateMachine(payload CreateMachinePayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer machine",
		Method:           http.MethodPost,
		Path:             createMachinePath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createMachinePayload(params map[string]any) CreateMachinePayload {
	return CreateMachinePayload{
		EntityName:                            stringParam(params, "entity_name"),
		Description:                           stringParam(params, "description"),
		CharsEOL:                              stringParam(params, "chars_eol"),
		IsFileDeletionAllowed:                 boolParam(params, "is_file_deletion_allowed"),
		IsFileViewingAllowed:                  boolParam(params, "is_file_viewing_allowed"),
		IsFileComparisonAllowed:               boolParam(params, "is_file_comparison_allowed"),
		IsRootBrowsingAllowed:                 boolParam(params, "is_root_browsing_allowed"),
		TargetName:                            stringParam(params, "target_name"),
		IsConfirmDeletionBeforeLoadDisabled:   boolParam(params, "is_confirm_deletion_before_load_disabled"),
		TargetNameLoad:                        stringParam(params, "target_name_load"),
		TargetNameUnload:                      stringParam(params, "target_name_unload"),
		TargetNameMazakMatrixFileMazak:        stringParam(params, "target_name_mazak_matrix_file_mazak"),
		TargetNameMazakMatrixFileLayout:       stringParam(params, "target_name_mazak_matrix_file_layout"),
		TargetNameMazakMatrixFileSetup:        stringParam(params, "target_name_mazak_matrix_file_setup"),
		TargetNamePresettingProgram:           stringParam(params, "target_name_presetting_program"),
		TargetNameProbeFile:                   stringParam(params, "target_name_probe_file"),
		DNCPortType:                           stringParam(params, "dnc_port_type"),
		TargetNameRoot:                        stringParam(params, "target_name_root"),
		MachineGroupsMachines:                 machineGroupsMachinesParam(params, "machine_group_ids"),
		OperatorInstructions:                  stringParam(params, "operator_instructions"),
		MachinesFiles:                         []any{},
		IsCommandProgramIgnored:               boolParam(params, "is_command_program_ignored"),
		TargetNameCommandProgram:              stringParam(params, "target_name_command_program"),
		CommandProgramErrorTemplateID:         numberParam(params, "command_program_error_template_id"),
		CommandProgramWaitBeforeLoadSeconds:   numberParam(params, "command_program_wait_before_load_seconds"),
		HasCommandProgram:                     boolParam(params, "has_command_program"),
		CommandProgramName:                    stringParam(params, "command_program_name"),
		WaitBetweenCommandProgramCheckSeconds: numberParam(params, "wait_between_command_program_check_seconds"),
		CommandProgramRegexp:                  stringParam(params, "command_program_regexp"),
		CommandProgramRegexpLoadValue:         stringParam(params, "command_program_regexp_load_value"),
		CommandProgramRegexpUnloadValue:       stringParam(params, "command_program_regexp_unload_value"),
		IsOperatorInstructionsDisplayed:       boolParam(params, "is_operator_instructions_displayed"),
		NumericalControlsParameterID:          numberParam(params, "numerical_controls_parameter_id"),
		CreatedBy:                             numberParam(params, "created_by"),
	}
}

func machineGroupsMachinesParam(params map[string]any, key string) []MachineGroupMachine {
	ids := numberListParam(params, key)
	items := make([]MachineGroupMachine, 0, len(ids))
	for _, id := range ids {
		items = append(items, MachineGroupMachine{MachineGroupID: id})
	}
	return items
}

const (
	createMachiningJobDefaultStatesPath      = "/entreprise/api/v1/machining_job_states/actions/create_default_states"
	createPresettingProgramDefaultStatesPath = "/entreprise/api/v1/machining_job_presetting_program_states/actions/create_default_presetting_states"
	createDocumentDefaultStatesPath          = "/entreprise/api/v1/document_states/actions/create_default_states"
)

func ExecuteCreateMachiningJobDefaultStates() ActionExecute {
	return executeCreateDefaultStates("Créer cycle de vie Dossier CN", createMachiningJobDefaultStatesPath, "Cycle de vie Dossier CN créé avec succès")
}

func ExecuteCreatePresettingProgramDefaultStates() ActionExecute {
	return executeCreateDefaultStates("Créer cycle de vie préréglage", createPresettingProgramDefaultStatesPath, "Cycle de vie préréglage créé avec succès")
}

func ExecuteCreateDocumentDefaultStates() ActionExecute {
	return executeCreateDefaultStates("Créer cycle de vie documents", createDocumentDefaultStatesPath, "Cycle de vie documents créé avec succès")
}

func executeCreateDefaultStates(name string, apiPath string, success string) ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		if err := client.CreateDefaultStates(name, apiPath, defaultStatesQuery(params)); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] %s.\n", success)
		return nil
	}
}

func (c *GedixAPIClient) CreateDefaultStates(name string, apiPath string, query map[string]string) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             name,
		Method:           http.MethodPost,
		Path:             apiPath,
		Query:            query,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func defaultStatesQuery(params map[string]any) map[string]string {
	return map[string]string{
		"lang":    stringParam(params, "lang"),
		"user_id": queryNumberParam(params, "user_id"),
	}
}

const createMachiningJobPath = "/entreprise/api/v1/machining_jobs/actions/create_new"

func ExecuteCreateMachiningJob() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		if err := validateNumberListMin(params, "machine_group_ids", 1); err != nil {
			return err
		}
		query := createMachiningJobQuery(params)
		if err := client.CreateMachiningJob(query); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Dossier CN créé avec succès : %s\n", stringParam(params, "entity_name"))
		return nil
	}
}

func (c *GedixAPIClient) CreateMachiningJob(query map[string]string) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer dossier CN",
		Method:           http.MethodPost,
		Path:             createMachiningJobPath,
		Query:            query,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createMachiningJobQuery(params map[string]any) map[string]string {
	return map[string]string{
		"overrideChecked":   "false",
		"user_id":           queryNumberParam(params, "user_id"),
		"machine_group_ids": numberListJSONParam(params, "machine_group_ids"),
		"version":           queryNumberParam(params, "version"),
		"description":       stringParam(params, "description"),
		"entity_name":       stringParam(params, "entity_name"),
	}
}
