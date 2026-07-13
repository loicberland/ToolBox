package lab

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	watchStatusValueCategoryPath = "/m2m/api/v1/statusvalue_categories"
	watchStatusValuePath         = "/m2m/api/v1/statusvalues"
	watchVariablesPath           = "/m2m/api/v1/variables"
	watchScriptsPath             = "/m2m/api/v1/scripts"
	watchTargetsPath             = "/m2m/api/v1/targets"
	watchAgentsPath              = "/m2m/api/v1/agents"
	watchPlantsPath              = "/entreprise/api/v1/plants"
	watchWorkshopsPath           = "/entreprise/api/v1/workshops"
	watchMachineGroupsPath       = "/entreprise/api/v1/machine_groups"
	watchMachinesPath            = "/entreprise/api/v1/machines"
)

func watchAPIActions() []Action {
	actions := []Action{
		{
			ID:          "create-statusvalue-category",
			Label:       "Créer une catégorie états machines",
			Description: "Crée une catégorie d'états machines.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "is_stop", Label: "Arrêt", Type: "bool", Default: false, HiddenWhen: map[string]any{"is_production": true}},
				{Name: "is_production", Label: "Production", Type: "bool", Default: false, HiddenWhen: map[string]any{"is_stop": true}},
				{Name: "color", Label: "Couleur d'affichage", Type: "color", Required: true, Default: "#D0021B"},
			},
			Execute: executeWatchPost("Créer une catégorie états machines", watchStatusValueCategoryPath, watchStatusValueCategoryPayload),
		},
		{
			ID:          "create-statusvalue",
			Label:       "Créer un état machine",
			Description: "Crée un état machine.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "statusvalue_category_id", Label: "Catégorie", Type: "number", Required: true, Default: 1},
				{Name: "is_color_overriden", Label: "Surcharger la couleur", Type: "bool", Default: false},
				{Name: "color", Label: "Couleur d'affichage", Type: "color", Default: "#D0021B", HiddenWhen: map[string]any{"is_color_overriden": false}},
			},
			Execute: executeWatchPost("Créer un état machine", watchStatusValuePath, watchStatusValuePayload),
		},
		{
			ID:          "create-acquisition-variable",
			Label:       "Créer une variable d'acquisition",
			Description: "Crée une variable d'acquisition.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "protocol", Label: "Type", Type: "string", Required: true, Options: watchAcquisitionVariableProtocolOptions()},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				watchAcquisitionAddressField(),
				{Name: "virtual_random", Label: "Sélection aléatoire", Type: "bool", Default: false, HiddenUnless: map[string]any{"protocol": "virtual"}},
				{Name: "virtual_values", Label: "Valeurs prises par la variable", Type: "string", Default: "", HiddenUnless: map[string]any{"protocol": "virtual"}},
			},
			Execute: executeWatchPost("Créer une variable d'acquisition", watchVariablesPath, watchAcquisitionVariablePayload),
		},
		{
			ID:          "create-calculated-variable",
			Label:       "Créer une variable calculée",
			Description: "Crée une variable calculée.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "protocol", Label: "Type", Type: "string", Required: true, Options: watchAcquisitionVariableProtocolOptions()},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
			},
			Execute: executeWatchPost("Créer une variable calculée", watchVariablesPath, watchCalculatedVariablePayload),
		},
		// {
		// 	ID:          "create-script",
		// 	Label:       "Créer un script",
		// 	Description: "Crée un script.",
		// 	Kind:        KindAPI,
		// 	Products:    []string{GedixWatchV10},
		// 	Fields: []ActionField{
		// 		{Name: "entity_name", Label: "Nom du script", Type: "string", Required: true},
		// 		{Name: "protocol", Label: "Protocole", Type: "string", Required: true, Options: watchProtocolOptions()},
		// 		{Name: "type", Label: "Type", Type: "string", Required: true, Default: "create", Options: []ActionOption{
		// 			{Label: "Script de transformation", Value: "transform"},
		// 			{Label: "Script de création d'une nouvelle variable", Value: "create"},
		// 		}},
		// 		{Name: "out_variable_id", Label: "Variable de sortie", Type: "number", Required: true, Min: 1},
		// 		{Name: "body", Label: "Script", Type: "text", Required: true, Default: ""},
		// 	},
		// 	Execute: executeWatchPost("Créer un script", watchScriptsPath, watchScriptPayload),
		// },
		{
			ID:          "create-target",
			Label:       "Créer une cible",
			Description: "Crée une cible.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "agent_name", Label: "Agent", Type: "string", Required: true, OptionsSource: "agents", Description: "Sélectionnez un agent défini dans l'onglet Agents."},
				{Name: "tunnel_steps", Label: "Etapes réseau", Type: "object[]", Default: []any{}, ItemFields: []ActionField{
					{Name: "entity_name", Label: "Nom relais", Type: "string"},
					{Name: "rank", Label: "Ordre", Type: "number"},
				}},
			},
			Execute: executeWatchPost("Créer une cible", watchTargetsPath, watchTargetPayload),
		},
		{
			ID:          "create-agent",
			Label:       "Créer un agent",
			Description: "Crée un agent.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields:      watchAgentFields(),
			Execute:     executeWatchPost("Créer un agent", watchAgentsPath, watchAgentPayload),
		},
		{
			ID:          "create-plant",
			Label:       "Créer une usine",
			Description: "Crée une usine.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "licensed_plant_id", Label: "Usine déclarée dans la licence", Type: "string", Required: true},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
				{Name: "session_opening_mode", Label: "Mode d'ouverture des sessions de travail", Type: "string", Required: true, Default: "workstation", Options: watchSessionOpeningModeOptions()},
			},
			Execute: executeWatchPost("Créer une usine", watchPlantsPath, watchPlantPayload),
		},
		{
			ID:          "create-workshop",
			Label:       "Créer un atelier",
			Description: "Crée un atelier.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "plant_id", Label: "Usine", Type: "number", Required: true, Default: 1},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
			},
			Execute: executeWatchPost("Créer un atelier", watchWorkshopsPath, watchWorkshopPayload),
		},
		{
			ID:          "create-machine-group",
			Label:       "Créer un groupe de machines",
			Description: "Crée un groupe de machines.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "workshop_id", Label: "Atelier", Type: "number", Required: true, Default: 1},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
			},
			Execute: executeWatchPost("Créer un groupe de machines", watchMachineGroupsPath, watchMachineGroupPayload),
		},
		{
			ID:          "create-machine",
			Label:       "Créer une machine",
			Description: "Crée une machine.",
			Kind:        KindAPI,
			Products:    []string{GedixWatchV10},
			Fields: []ActionField{
				{Name: "entity_name", Label: "Nom", Type: "string", Required: true},
				{Name: "machine_group_id", Label: "Groupe de machines", Type: "number", Required: true, Default: 1},
				{Name: "agent_id", Label: "Agent", Type: "number"},
				{Name: "description", Label: "Description", Type: "string", Default: ""},
			},
			Execute: executeWatchPost("Créer une machine", watchMachinesPath, watchMachinePayload),
		},
	}
	return append(actions, legacyWatchAPIActions(actions)...)
}

func legacyWatchAPIActions(actions []Action) []Action {
	aliases := map[string]string{
		"create-statusvalue-category": "create-watch-statusvalue-category",
		"create-statusvalue":          "create-watch-statusvalue",
		"create-acquisition-variable": "create-watch-acquisition-variable",
		"create-calculated-variable":  "create-watch-calculated-variable",
		"create-script":               "create-watch-script",
		"create-target":               "create-watch-target",
		"create-agent":                "create-watch-agent",
		"create-plant":                "create-watch-plant",
		"create-workshop":             "create-watch-workshop",
		"create-machine-group":        "create-watch-machine-group",
		"create-machine":              "create-watch-machine",
	}
	items := []Action{}
	for _, action := range actions {
		aliasID, ok := aliases[action.ID]
		if !ok {
			continue
		}
		alias := action
		alias.ID = aliasID
		alias.Hidden = true
		items = append(items, alias)
	}
	return items
}

func executeWatchPost(name string, path string, payload func(map[string]any) any) ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		body := payload(params)
		if err := client.DoJSON(GedixAPIRequest{Name: name, Method: http.MethodPost, Path: path, Body: body}); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] %s créée avec succès.\n", name)
		return nil
	}
}

func watchStatusValueCategoryPayload(params map[string]any) any {
	isStop := boolParam(params, "is_stop")
	isProduction := boolParam(params, "is_production")
	if isStop {
		isProduction = false
	}
	return map[string]any{
		"entity_name":   stringParam(params, "entity_name"),
		"description":   stringParam(params, "description"),
		"is_stop":       isStop,
		"is_production": isProduction,
		"color":         stringParam(params, "color"),
	}
}

func watchStatusValuePayload(params map[string]any) any {
	return map[string]any{
		"entity_name":             stringParam(params, "entity_name"),
		"description":             stringParam(params, "description"),
		"statusvalue_category_id": intParam(params, "statusvalue_category_id"),
		"is_color_overriden":      boolParam(params, "is_color_overriden"),
		"color":                   firstNonEmpty(stringParam(params, "color"), "#D0021B"),
	}
}

func watchAcquisitionVariablePayload(params map[string]any) any {
	protocol := stringParam(params, "protocol")
	address := stringParam(params, "address")
	if protocol == "sincom_alarm" {
		address = "alarm"
	} else if protocol == "virtual" {
		address = ""
	}
	return map[string]any{
		"entity_name":    stringParam(params, "entity_name"),
		"protocol":       protocol,
		"address":        address,
		"description":    stringParam(params, "description"),
		"is_processed":   false,
		"virtual_random": boolParam(params, "virtual_random"),
		"virtual_values": stringParam(params, "virtual_values"),
	}
}

func watchCalculatedVariablePayload(params map[string]any) any {
	return map[string]any{
		"entity_name":  stringParam(params, "entity_name"),
		"protocol":     stringParam(params, "protocol"),
		"address":      "",
		"description":  stringParam(params, "description"),
		"is_processed": true,
	}
}

func watchScriptPayload(params map[string]any) any {
	payload := map[string]any{
		"entity_name":       stringParam(params, "entity_name"),
		"type":              stringParam(params, "type"),
		"protocol":          stringParam(params, "protocol"),
		"scripts_variables": []any{},
		"body":              stringParam(params, "body"),
		"tests":             "[]",
	}
	if stringParam(params, "type") != "transform" {
		payload["out_variable_id"] = intParam(params, "out_variable_id")
	}
	return payload
}

func watchTargetPayload(params map[string]any) any {
	return map[string]any{
		"entity_name":  stringParam(params, "entity_name"),
		"description":  stringParam(params, "description"),
		"agent_name":   stringParam(params, "agent_name"),
		"tunnel_steps": watchTunnelStepsParam(params, "tunnel_steps"),
	}
}

func watchAgentPayload(params map[string]any) any {
	protocol := stringParam(params, "protocol")
	payload := map[string]any{
		"entity_name":                  stringParam(params, "entity_name"),
		"protocol":                     protocol,
		"cnc_model":                    stringParam(params, "cnc_model"),
		"external_id":                  stringParam(params, "external_id"),
		"host_id":                      stringParam(params, "host_id"),
		"host_port":                    defaultIntParam(params, "host_port", 3010),
		"ip":                           stringParam(params, "ip"),
		"port":                         watchAgentPort(params, protocol),
		"polling_delay":                defaultIntParam(params, "polling_delay", 2),
		"turret":                       defaultIntParam(params, "turret", 1),
		"status_file_path":             stringParam(params, "status_file_path"),
		"version":                      firstNonEmpty(stringParam(params, "version"), "6250"),
		"connection_name":              stringParam(params, "connection_name"),
		"is_auth_needed":               boolParam(params, "is_auth_needed"),
		"remote_auth_username":         stringParam(params, "remote_auth_username"),
		"remote_auth_password":         stringParam(params, "remote_auth_password"),
		"is_tls_needed":                boolParam(params, "is_tls_needed"),
		"is_monitoring_enabled":        defaultBoolParam(params, "is_monitoring_enabled", true),
		"tls_certificate_bytes":        stringParam(params, "tls_certificate_bytes"),
		"tls_private_key_bytes":        stringParam(params, "tls_private_key_bytes"),
		"agents_variables":             watchAgentVariablesParam(params),
		"security_mode":                firstNonEmpty(stringParam(params, "security_mode"), "None"),
		"security_policy":              firstNonEmpty(stringParam(params, "security_policy"), "None"),
		"agents_scripts":               watchAgentScriptsParam(params),
		"waiting_response_delay":       defaultIntParam(params, "waiting_response_delay", 10),
		"opcua_server_id":              intParam(params, "opcua_server_id"),
		"target_id":                    nil,
		"is_amnesic":                   boolParam(params, "is_amnesic"),
		"is_forcing_time_reason_error": boolParam(params, "is_forcing_time_reason_error"),
		"read_error_statusvalue_id":    intParam(params, "read_error_statusvalue_id"),
		"is_keeping_status_error":      defaultBoolParam(params, "is_keeping_status_error", true),
	}
	if targetID, ok := anyToInt(params["target_id"]); ok && targetID > 0 {
		payload["target_id"] = targetID
	}
	return payload
}

func watchPlantPayload(params map[string]any) any {
	return map[string]any{
		"entity_name":          stringParam(params, "entity_name"),
		"session_opening_mode": firstNonEmpty(stringParam(params, "session_opening_mode"), "workstation"),
		"description":          stringParam(params, "description"),
		"licensed_plant_id":    stringParam(params, "licensed_plant_id"),
	}
}

func watchWorkshopPayload(params map[string]any) any {
	return map[string]any{
		"entity_name": stringParam(params, "entity_name"),
		"description": stringParam(params, "description"),
		"plant_id":    defaultIntParam(params, "plant_id", 1),
	}
}

func watchMachineGroupPayload(params map[string]any) any {
	return map[string]any{
		"entity_name": stringParam(params, "entity_name"),
		"description": stringParam(params, "description"),
		"workshop_id": defaultIntParam(params, "workshop_id", 1),
	}
}

func watchMachinePayload(params map[string]any) any {
	return map[string]any{
		"entity_name":      stringParam(params, "entity_name"),
		"description":      stringParam(params, "description"),
		"machine_group_id": defaultIntParam(params, "machine_group_id", 1),
		"agent_id":         intParam(params, "agent_id"),
		"file_id":          defaultIntParam(params, "file_id", 0),
		"file_uploaded":    defaultBoolParam(params, "file_uploaded", false),
		"filetype":         firstNonEmpty(stringParam(params, "filetype"), "image/png"),
		"filename":         stringParam(params, "filename"),
		"show_picture":     defaultBoolParam(params, "show_picture", false),
	}
}

func watchTunnelStepsParam(params map[string]any, key string) []map[string]any {
	rows := objectArrayParam(params, key)
	items := []map[string]any{}
	for index, row := range rows {
		entityName := strings.TrimSpace(fmt.Sprint(row["entity_name"]))
		if entityName == "" {
			continue
		}
		rank := index + 1
		if value, ok := anyToInt(row["rank"]); ok && value > 0 {
			rank = value
		}
		items = append(items, map[string]any{"entity_name": entityName, "rank": rank})
	}
	return items
}

func watchAgentVariablesParam(params map[string]any) []map[string]any {
	statusID := intParam(params, "machine_status_variable_id")
	partCounterID := intParam(params, "part_counter_variable_id")
	mainProgramID := intParam(params, "main_program_variable_id")
	items := []map[string]any{}
	for _, row := range objectArrayParam(params, "agents_variables") {
		variableID, ok := anyToInt(row["variable_id"])
		if !ok || variableID <= 0 {
			continue
		}
		items = append(items, map[string]any{
			"variable_id":       variableID,
			"value":             variableID,
			"updated":           1,
			"is_machine_status": variableID == statusID,
			"is_part_counter":   variableID == partCounterID,
			"is_main_program":   variableID == mainProgramID,
		})
	}
	return items
}

func watchAgentScriptsParam(params map[string]any) []map[string]any {
	items := []map[string]any{}
	for _, row := range objectArrayParam(params, "agents_scripts") {
		scriptID, ok := anyToInt(row["script_id"])
		if ok && scriptID > 0 {
			items = append(items, map[string]any{"script_id": scriptID})
		}
	}
	return items
}

func watchAgentPort(params map[string]any, protocol string) int {
	if port, ok := anyToInt(params["port"]); ok && port > 0 {
		return port
	}
	switch protocol {
	case "ezsocket-module":
		return 683
	case "modbus", "modbus-module", "advantech-modbus-dotnet":
		return 502
	case "mqtt-module", "jfmx-mqtt":
		return 1883
	case "mtconnect-module":
		return 5000
	case "opcua-module":
		return 4841
	case "siemens-s7-module":
		return 102
	default:
		return 0
	}
}

func defaultIntParam(params map[string]any, key string, defaultValue int) int {
	if value, ok := anyToInt(params[key]); ok {
		return value
	}
	return defaultValue
}

func defaultBoolParam(params map[string]any, key string, defaultValue bool) bool {
	value, exists := params[key]
	if !exists {
		return defaultValue
	}
	typed, ok := value.(bool)
	if !ok {
		return defaultValue
	}
	return typed
}

func watchSessionOpeningModeOptions() []ActionOption {
	return []ActionOption{
		{Label: "Scan poste de travail uniquement", Value: "workstation"},
		{Label: "Scan OF", Value: "workorder"},
		{Label: "Scan Poste de travail et pièce", Value: "part"},
	}
}

func watchProtocolOptions() []ActionOption {
	return []ActionOption{
		{Value: "adam_var", Label: "Adam - Agent V1"},
		{Value: "focas", Label: "Focas - Agent V1"},
		{Value: "focas-module", Label: "Focas - Agent V3"},
		{Value: "heidenhain_var", Label: "Heidenhain - Agent V1"},
		{Value: "heidenhain-module", Label: "Heidenhain - Agent V3"},
		{Value: "ezsocket-module", Label: "Mitsubishi EzSocket - Agent V3"},
		{Value: "jfmx-mqtt", Label: "JFMX-MQTT - Agent V1"},
		{Value: "jfmx_var", Label: "JFMX-XML - Agent V1"},
		{Value: "modbus", Label: "Modbus - Agent V2"},
		{Value: "modbus-module", Label: "Modbus - Agent V3"},
		{Value: "advantech-modbus-dotnet", Label: "Advantech Modbus - Agent V3"},
		{Value: "mqtt-module", Label: "MQTT - Agent V3"},
		{Value: "mtconnect-module", Label: "Mtconnect - Agent V3"},
		{Value: "okuma-module", Label: "Okuma - Agent V3"},
		{Value: "okuma-module-xp32b", Label: "Okuma - Agent V3 32b"},
		{Value: "opcua-module", Label: "OPC-UA - Agent V3"},
		{Value: "opc", Label: "OPC-UA - Agent V2"},
		{Value: "sincom", Label: "Sincom - Agent V1"},
		{Value: "sincom-module", Label: "Sincom - Agent V3"},
		{Value: "virtual", Label: "Virtuel - Agent V2"},
		{Value: "virtual-module", Label: "Virtuel - Agent V3"},
		{Value: "brother-module", Label: "Brother - Agent V3"},
		{Value: "siemens-dde-module", Label: "Siemens/dde - Agent V3"},
		{Value: "siemens-s7-module", Label: "Siemens S7 - Agent V3"},
	}
}

func watchAcquisitionVariableProtocolOptions() []ActionOption {
	options := append([]ActionOption{}, watchProtocolOptions()...)
	options = append(options,
		ActionOption{Value: "focas_var", Label: "Focas/Fonction - Agent V1"},
		ActionOption{Value: "focas_register", Label: "Focas/Registre - Agent V1"},
		ActionOption{Value: "focas_parameter", Label: "Focas/Paramètre - Agent V1"},
		ActionOption{Value: "focas_axis", Label: "Focas/Axe - Agent V1"},
		ActionOption{Value: "focas_macro", Label: "Focas/Macro - Agent V1"},
		ActionOption{Value: "modbus_var", Label: "Modbus/Booléen - Agent V2"},
		ActionOption{Value: "modbus_register", Label: "Modbus/Nombre - Agent V2"},
		ActionOption{Value: "opc_var", Label: "PC-UA/Booléen - Agent V2"},
		ActionOption{Value: "opc_register", Label: "OPC-UA/Nombre - Agent V2"},
		ActionOption{Value: "schneider_var", Label: "Schneider - Agent V1"},
		ActionOption{Value: "sincom_var", Label: "Sincom/Variable - Agent V1"},
		ActionOption{Value: "sincom_alarm", Label: "Sincom/Alarme - Agent V1"},
	)
	return options
}

func watchAcquisitionAddressField() ActionField {
	moduleProtocols := []map[string]any{
		{"protocol": "focas-module"},
		{"protocol": "brother-module"},
		{"protocol": "modbus-module"},
		{"protocol": "advantech-modbus-dotnet"},
		{"protocol": "mqtt-module"},
		{"protocol": "ezsocket-module"},
		{"protocol": "mtconnect-module"},
		{"protocol": "opcua-module"},
	}
	return ActionField{
		Name:     "address",
		Label:    "Variable module",
		Type:     "string",
		Required: true,
		HiddenWhenAny: []map[string]any{
			{"protocol": "sincom_alarm"},
			{"protocol": "virtual"},
		},
		LabelWhen: []ConditionalText{
			{When: map[string]any{"protocol": "modbus_var"}, Text: "Numéro du channel"},
			{When: map[string]any{"protocol": "modbus_register"}, Text: "Numéro du channel"},
			{When: map[string]any{"protocol": "opc_var"}, Text: "Identifiant du noeud associé"},
			{When: map[string]any{"protocol": "opc_register"}, Text: "Identifiant du noeud associé"},
			{When: map[string]any{"protocol": "virtual-module"}, Text: "Règle de génération"},
			{When: map[string]any{"protocol": "schneider_var"}, Text: "Identifiant du noeud associé"},
			{When: map[string]any{"protocol": "sincom-module"}, Text: "Identifiant du noeud associé"},
			{When: map[string]any{"protocol": "siemens-dde-module"}, Text: "Identifiant du noeud associé"},
			{When: map[string]any{"protocol": "sincom_var"}, Text: "Identifiant du noeud associé"},
		},
		DescriptionWhen: append([]ConditionalText{
			{When: map[string]any{"protocol": "jfmx-mqtt"}, Text: "Topic de la variable. Il s'ajoute après la chaîne de connexion de l'agent"},
			{When: map[string]any{"protocol": "adam_var"}, Text: "Bit (Digital input) entre 1-n selon le modèle du boitier"},
			{When: map[string]any{"protocol": "schneider_var"}, Text: "Bit (Digital input) entre 1-n selon le modèle de l'automate"},
			{When: map[string]any{"protocol": "focas_parameter"}, Text: "Exemple: 6711"},
			{When: map[string]any{"protocol": "focas_register"}, Text: "Exemple: 100"},
			{When: map[string]any{"protocol": "jfmx_var"}, Text: "Nom du noeud dans le fichier xml"},
			{When: map[string]any{"protocol": "modbus_var"}, Text: "Numéro du channel de la variable à récupérer (entre 0 et 7)"},
			{When: map[string]any{"protocol": "modbus_register"}, Text: "Numéro du channel de la variable à récupérer (entre 0 et 7)"},
			{When: map[string]any{"protocol": "opc_var"}, Text: "Exemple pour récupérer un compteur : ns=3;s=Counter"},
			{When: map[string]any{"protocol": "opc_register"}, Text: "Exemple pour récupérer un compteur : ns=3;s=Counter"},
			{When: map[string]any{"protocol": "virtual-module"}, Text: `Avec la fonction 'generate' et les paramètres 'values' (séparées par ':') et 'strategy' ('serie' ou 'random'). Exemples : "generate|values=a:b:c;strategy=serie", "generate|values=a:b#10:c#5;strategy=serie", "generate|values=a:b:c;strategy=random", "generate|values=a#10:b:c;strategy=random". Pour déclencher une erreur lors de la lecture, la valeur doit être "read_error". Exemple : "generate|values=a:read_error:c;strategy=serie"`},
		}, descriptionWhenForProtocols(moduleProtocols, "Voir help CLI module")...),
	}
}

func descriptionWhenForProtocols(groups []map[string]any, text string) []ConditionalText {
	items := make([]ConditionalText, 0, len(groups))
	for _, group := range groups {
		items = append(items, ConditionalText{When: group, Text: text})
	}
	return items
}

func watchAgentFields() []ActionField {
	return []ActionField{
		{Name: "entity_name", Label: "Nom de l'agent", Type: "string", Required: true},
		{Name: "target_id", Label: "Cible", Type: "number"},
		{Name: "protocol", Label: "Protocole", Type: "string", Required: true, Options: watchProtocolOptions()},
		{Name: "ip", Label: "IP de la machine", Type: "string", Required: true, HiddenWhenAny: hiddenForProtocols("siemens-dde-module", "jfmx_var", "heidenhain_var", "virtual", "virtual-module")},
		{Name: "port", Label: "Port de la machine", Type: "number", Required: true, HiddenWhenAny: hiddenForProtocols("siemens-dde-module", "jfmx_var", "heidenhain_var", "virtual", "virtual-module")},
		{Name: "turret", Label: "Numéro de la tourelle", Type: "number", Required: true, Default: 1, HiddenWhenAny: hiddenForProtocols("siemens-dde-module", "jfmx_var", "adam", "jfmx-mqtt", "heidenhain_var", "ezsocket-module", "modbus", "modbus-module", "advantech-modbus-dotnet", "mqtt-module", "mtconnect-module", "opcua-module", "siemens-s7-module", "virtual", "virtual-module")},
		{Name: "polling_delay", Label: "Durée d'attente en secondes entre deux séries de lecture des variables", Type: "number", Required: true, Default: 2},
		{Name: "is_monitoring_enabled", Label: "Autorise la connexion de l'agent avec la machine", Type: "bool", Default: true},
		{Name: "is_forcing_time_reason_error", Label: "Transformer les erreurs de connexion en état machine", Type: "bool", Default: false},
		{Name: "is_amnesic", Label: "Ne pas historiser les variables acquises", Type: "bool", Default: false},
		{Name: "is_keeping_status_error", Label: "Garder le dernier état acquis", Type: "bool", Default: true, HiddenWhen: map[string]any{"is_forcing_time_reason_error": false}},
		{Name: "read_error_statusvalue_id", Label: "Etat machine forcé quand la machine n'est pas joignable", Type: "number", HiddenUnless: map[string]any{"is_forcing_time_reason_error": true, "is_keeping_status_error": false}},
		{Name: "connection_name", Label: "Chaîne de connexion", Type: "string", HiddenUnlessAny: visibleForProtocols("jfmx-mqtt", "opc", "opcua-module", "siemens-s7-module", "heidenhain_var"), RequiredWhenAny: visibleForProtocols("jfmx-mqtt", "opcua-module", "heidenhain_var")},
		{Name: "waiting_response_delay", Label: "Timeout en seconde pour la connexion à la machine", Type: "number", Required: true, Default: 10, HiddenUnlessAny: visibleForProtocols("jfmx-mqtt", "opc", "opcua-module", "siemens-s7-module", "virtual", "virtual-module", "focas-module", "sincom-module", "heidenhain_var", "heidenhain-module", "ezsocket-module")},
		{Name: "is_auth_needed", Label: "Authentification distante requise", Type: "bool", Default: false, HiddenUnlessAny: visibleForProtocols("jfmx-mqtt", "opc", "opcua-module", "heidenhain_var")},
		{Name: "remote_auth_username", Label: "Identifiant", Type: "string", HiddenUnless: map[string]any{"is_auth_needed": true}},
		{Name: "remote_auth_password", Label: "Mot de passe", Type: "string", HiddenUnlessAny: []map[string]any{{"is_auth_needed": true}, {"protocol": "siemens-s7-module"}}},
		{Name: "is_tls_needed", Label: "Connexion distante sécurisée par TLS requise", Type: "bool", Default: false, HiddenUnlessAny: visibleForProtocols("jfmx-mqtt", "opc", "opcua-module", "heidenhain_var")},
		{Name: "tls_certificate_bytes", Label: "Certificat TLS", Type: "text", HiddenUnless: map[string]any{"is_tls_needed": true}},
		{Name: "tls_private_key_bytes", Label: "Clef privée du certificat", Type: "text", HiddenUnless: map[string]any{"is_tls_needed": true}},
		{Name: "cnc_model", Label: "Modèle CN", Type: "string", Default: "", HiddenUnlessAny: visibleForProtocols("focas-module")},
		{Name: "version", Label: "Version du boîtier adam", Type: "string", Required: true, Default: "6250", HiddenUnlessAny: visibleForProtocols("adam_var")},
		{Name: "status_file_path", Label: "Chemin vers le fichier xml", Type: "string", Required: true, HiddenUnlessAny: visibleForProtocols("jfmx_var")},
		{Name: "security_mode", Label: "Mode de sécurité", Type: "string", Default: "None", Options: []ActionOption{{Label: "Aucun", Value: "None"}, {Label: "Sign", Value: "Sign"}, {Label: "SignAndEncrypt", Value: "SignAndEncrypt"}}, HiddenUnlessAny: visibleForProtocols("opc", "opcua-module")},
		{Name: "security_policy", Label: "Politique de sécurité", Type: "string", Default: "None", Options: []ActionOption{{Label: "Aucun", Value: "None"}, {Label: "Basic128Rsa15", Value: "Basic128Rsa15"}, {Label: "Basic256", Value: "Basic256"}, {Label: "Basic256Sha256", Value: "Basic256Sha256"}, {Label: "Aes128Sha256RsaOaep", Value: "Aes128Sha256RsaOaep"}, {Label: "Aes256Sha256RsaPss", Value: "Aes256Sha256RsaPss"}}, HiddenUnlessAny: visibleForProtocols("opc", "opcua-module")},
		{Name: "external_id", Label: "Identifiant de la machine Sincom", Type: "string", Required: true, HiddenUnlessAny: visibleForProtocols("sincom-module")},
		{Name: "host_id", Label: "Identifiant de l'hôte", Type: "string", Required: true, HiddenUnlessAny: visibleForProtocols("sincom-module")},
		{Name: "host_port", Label: "Port d'écoute de l'hôte", Type: "number", Required: true, Default: 3010, HiddenUnlessAny: visibleForProtocols("sincom-module")},
		{Name: "agents_variables", Label: "Variables d'acquisition", Type: "object[]", Default: []any{}, ItemFields: []ActionField{{Name: "variable_id", Label: "Variable", Type: "number", Required: true}}},
		{Name: "machine_status_variable_id", Label: "Variable contenant l'état machine", Type: "string", OptionsSource: "field:agents_variables.variable_id", HiddenUnlessNonEmpty: "agents_variables"},
		{Name: "part_counter_variable_id", Label: "Variable contenant le compteur pièces", Type: "string", OptionsSource: "field:agents_variables.variable_id", HiddenUnlessNonEmpty: "agents_variables"},
		{Name: "main_program_variable_id", Label: "Variable contenant le programme principal", Type: "string", OptionsSource: "field:agents_variables.variable_id", HiddenUnlessNonEmpty: "agents_variables"},
		{Name: "agents_scripts", Label: "Scripts", Type: "object[]", Default: []any{}, ItemFields: []ActionField{{Name: "script_id", Label: "Script", Type: "number", Required: true}}},
		{Name: "opcua_server_id", Label: "Serveur OPC-UA distant", Type: "number", Default: 0},
	}
}

func hiddenForProtocols(protocols ...string) []map[string]any {
	groups := make([]map[string]any, 0, len(protocols))
	for _, protocol := range protocols {
		groups = append(groups, map[string]any{"protocol": protocol})
	}
	return groups
}

func visibleForProtocols(protocols ...string) []map[string]any {
	return hiddenForProtocols(protocols...)
}
