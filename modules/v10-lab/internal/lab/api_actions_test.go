package lab

import (
	"net/url"
	"reflect"
	"testing"
)

func TestCreateMachineGroupPayload(t *testing.T) {
	payload := createMachineGroupPayload(paramsWithDefaults(mustFindAction(t, "create-machine-group"), map[string]any{
		"entity_name": "Groupe1",
		"workshop_id": 2,
	}))

	if payload.EntityName != "Groupe1" || payload.CharsEOLDefault != "13,10" || payload.WorkshopID != 2 {
		t.Fatalf("unexpected machine group payload: %#v", payload)
	}
	if payload.MachineGroupsFiles == nil || len(payload.MachineGroupsFiles) != 0 {
		t.Fatalf("machine_groups_files must be an empty array: %#v", payload.MachineGroupsFiles)
	}
}

func TestCreateMachineGroupCatalogDefaults(t *testing.T) {
	action := mustFindAction(t, "create-machine-group")
	workshopID := mustFindField(t, action, "workshop_id")
	createdBy := mustFindField(t, action, "created_by")
	if !workshopID.Required || workshopID.Default != 1 || workshopID.Min != 1 {
		t.Fatalf("unexpected workshop_id metadata: %#v", workshopID)
	}
	if !createdBy.Required || createdBy.Default != 1 || createdBy.Min != 1 {
		t.Fatalf("unexpected created_by metadata: %#v", createdBy)
	}
	payload := createMachineGroupPayload(paramsWithDefaults(action, map[string]any{"entity_name": "Groupe1"}))
	if payload.WorkshopID != 1 || payload.CreatedBy != 1 {
		t.Fatalf("expected catalog defaults in payload: %#v", payload)
	}
}

func TestCreateTargetPayloadEmptyArrays(t *testing.T) {
	payload := createTargetPayload(paramsWithDefaults(mustFindAction(t, "create-target"), map[string]any{
		"entity_name":    "cible2",
		"connector_name": "connector-focas-01",
	}))

	if payload.EntityName != "cible2" || payload.ConnectorName != "connector-focas-01" {
		t.Fatalf("unexpected target payload: %#v", payload)
	}
	if len(payload.Configs) != 0 || len(payload.TunnelSteps) != 0 {
		t.Fatalf("expected empty arrays, got %#v", payload)
	}
}

func TestCreateTargetPayloadWithConfigsAndTunnelSteps(t *testing.T) {
	payload := createTargetPayload(paramsWithDefaults(mustFindAction(t, "create-target"), map[string]any{
		"entity_name":    "cible2",
		"connector_name": "connector-focas-01",
		"configs": []any{
			map[string]any{"module_key": "remote-filepath", "module_value": "remote"},
			map[string]any{"module_key": "subprogram-filepath", "module_value": "sub"},
		},
		"tunnel_steps": []any{
			map[string]any{"entity_name": "relay-dnc-01", "rank": float64(1)},
			map[string]any{"entity_name": "relay-dnc-02", "rank": float64(2)},
		},
	}))

	if !reflect.DeepEqual(payload.Configs, []TargetConfig{{ModuleKey: "remote-filepath", ModuleValue: "remote"}, {ModuleKey: "subprogram-filepath", ModuleValue: "sub"}}) {
		t.Fatalf("unexpected configs: %#v", payload.Configs)
	}
	if len(payload.TunnelSteps) != 2 || payload.TunnelSteps[0].EntityName != "relay-dnc-01" || payload.TunnelSteps[1].Rank != 2 {
		t.Fatalf("unexpected tunnel steps: %#v", payload.TunnelSteps)
	}
}

func TestCreateTargetCatalogCreatedByRequired(t *testing.T) {
	action := mustFindAction(t, "create-target")
	createdBy := mustFindField(t, action, "created_by")
	if !createdBy.Required || createdBy.Default != 1 || createdBy.Min != 1 {
		t.Fatalf("unexpected created_by metadata: %#v", createdBy)
	}
	payload := createTargetPayload(paramsWithDefaults(action, map[string]any{
		"entity_name":    "cible2",
		"connector_name": "connector-focas-01",
	}))
	if payload.CreatedBy != 1 {
		t.Fatalf("expected created_by default in payload: %#v", payload)
	}
}

func TestCreatedByCatalogMin(t *testing.T) {
	for _, actionID := range []string{"create-plant", "create-workshop", "create-machine-group", "create-target", "create-machine"} {
		action := mustFindAction(t, actionID)
		createdBy := mustFindField(t, action, "created_by")
		if !createdBy.Required || createdBy.Default != 1 || createdBy.Min != 1 {
			t.Fatalf("unexpected created_by metadata for %s: %#v", actionID, createdBy)
		}
	}
}

func TestCreateMachinePayloadEthernetWithGroup(t *testing.T) {
	payload := createMachinePayload(paramsWithDefaults(mustFindAction(t, "create-machine"), map[string]any{
		"entity_name":       "Machine",
		"dnc_port_type":     "ethernet",
		"machine_group_ids": []any{float64(2)},
		"target_name_root":  "cible2",
	}))

	if payload.EntityName != "Machine" || payload.CharsEOL != "13,10" || payload.DNCPortType != "ethernet" {
		t.Fatalf("unexpected machine payload: %#v", payload)
	}
	if !payload.IsFileDeletionAllowed || !payload.IsFileViewingAllowed || !payload.IsRootBrowsingAllowed {
		t.Fatalf("expected ethernet defaults to allow file/root options: %#v", payload)
	}
	if !reflect.DeepEqual(payload.MachineGroupsMachines, []MachineGroupMachine{{MachineGroupID: 2}}) {
		t.Fatalf("unexpected machine groups: %#v", payload.MachineGroupsMachines)
	}
}

func TestCreateMachineMachineGroupIDsValidation(t *testing.T) {
	for _, params := range []map[string]any{
		{"machine_group_ids": []any{}},
		{"machine_group_ids": []any{float64(1), float64(2)}},
	} {
		if err := validateNumberListMin(params, "machine_group_ids", 1); err != nil {
			t.Fatalf("expected valid machine_group_ids %#v: %v", params, err)
		}
	}
	for _, params := range []map[string]any{
		{"machine_group_ids": []any{float64(0)}},
		{"machine_group_ids": []any{float64(-1)}},
	} {
		if err := validateNumberListMin(params, "machine_group_ids", 1); err == nil {
			t.Fatalf("expected invalid machine_group_ids %#v", params)
		}
	}
}

func TestCreateMachinePayloadSerial(t *testing.T) {
	payload := createMachinePayload(paramsWithDefaults(mustFindAction(t, "create-machine"), map[string]any{
		"entity_name":   "Machine",
		"dnc_port_type": "serial",
	}))

	if payload.DNCPortType != "serial" || payload.TargetNameRoot != "cible2" {
		t.Fatalf("hidden fields must keep backend defaults: %#v", payload)
	}
}

func TestCreateMachinePayloadWithoutCommandProgram(t *testing.T) {
	payload := createMachinePayload(paramsWithDefaults(mustFindAction(t, "create-machine"), map[string]any{
		"entity_name":          "Machine",
		"has_command_program":  false,
		"command_program_name": "ignored-by-ui-but-kept",
	}))

	if payload.HasCommandProgram {
		t.Fatalf("expected command program disabled: %#v", payload)
	}
	if payload.CommandProgramName != "ignored-by-ui-but-kept" {
		t.Fatalf("backend builder should not drop provided hidden values: %#v", payload)
	}
}

func TestCreateMachinePayloadWithCommandProgram(t *testing.T) {
	payload := createMachinePayload(paramsWithDefaults(mustFindAction(t, "create-machine"), map[string]any{
		"entity_name":                                "Machine",
		"has_command_program":                        true,
		"command_program_name":                       "CMD",
		"wait_between_command_program_check_seconds": float64(45),
		"command_program_regexp":                     "LOAD",
		"command_program_regexp_load_value":          "L",
		"command_program_regexp_unload_value":        "U",
		"command_program_wait_before_load_seconds":   float64(5),
		"command_program_error_template_id":          float64(7),
		"target_name_command_program":                "cmd-target",
		"is_command_program_ignored":                 true,
	}))

	if !payload.HasCommandProgram || payload.CommandProgramName != "CMD" || payload.CommandProgramErrorTemplateID != int64(7) || !payload.IsCommandProgramIgnored {
		t.Fatalf("unexpected command program payload: %#v", payload)
	}
}

func TestCreateMachiningJobCatalogUserIDRequired(t *testing.T) {
	action := mustFindAction(t, "create-machining-job")
	userID := mustFindField(t, action, "user_id")
	if !userID.Required || userID.Default != 1 || userID.Min != 1 {
		t.Fatalf("unexpected user_id metadata: %#v", userID)
	}
	query := createMachiningJobQuery(paramsWithDefaults(action, map[string]any{
		"entity_name": "dossier",
	}))
	if query["user_id"] != "1" {
		t.Fatalf("expected user_id default in query: %#v", query)
	}
}

func TestLifecycleActionsCatalogUserIDRequired(t *testing.T) {
	for _, actionID := range []string{"create-machining-job-default-states", "create-presetting-program-default-states", "create-document-default-states"} {
		action := mustFindAction(t, actionID)
		userID := mustFindField(t, action, "user_id")
		if !userID.Required || userID.Default != 1 || userID.Min != 1 {
			t.Fatalf("unexpected user_id metadata for %s: %#v", actionID, userID)
		}
	}
}

func TestCreateMachiningJobMachineGroupIDsValidation(t *testing.T) {
	for _, params := range []map[string]any{
		{"machine_group_ids": []any{}},
		{"machine_group_ids": []any{float64(1)}},
	} {
		if err := validateNumberListMin(params, "machine_group_ids", 1); err != nil {
			t.Fatalf("expected valid machine_group_ids %#v: %v", params, err)
		}
	}
	for _, params := range []map[string]any{
		{"machine_group_ids": []any{float64(0)}},
		{"machine_group_ids": []any{float64(-1)}},
	} {
		if err := validateNumberListMin(params, "machine_group_ids", 1); err == nil {
			t.Fatalf("expected invalid machine_group_ids %#v", params)
		}
	}
}

func TestCreateMachiningJobQueryWithMachineGroup(t *testing.T) {
	query := createMachiningJobQuery(paramsWithDefaults(mustFindAction(t, "create-machining-job"), map[string]any{
		"entity_name":       "groupe 1",
		"description":       `jai m'i des accen"`,
		"machine_group_ids": "1",
		"version":           float64(0),
	}))

	if query["machine_group_ids"] != "[1]" || query["overrideChecked"] != "false" {
		t.Fatalf("unexpected machining job query: %#v", query)
	}
	encoded := url.Values{
		"entity_name":       {query["entity_name"]},
		"description":       {query["description"]},
		"machine_group_ids": {query["machine_group_ids"]},
	}.Encode()
	if parsed, err := url.ParseQuery(encoded); err != nil || parsed.Get("entity_name") != "groupe 1" || parsed.Get("description") != `jai m'i des accen"` || parsed.Get("machine_group_ids") != "[1]" {
		t.Fatalf("query must round-trip through url.Values: encoded=%s parsed=%#v err=%v", encoded, parsed, err)
	}
}

func TestCreateMachiningJobQueryWithoutMachineGroup(t *testing.T) {
	query := createMachiningJobQuery(paramsWithDefaults(mustFindAction(t, "create-machining-job"), map[string]any{
		"entity_name":       "dossié 15 être",
		"machine_group_ids": "",
	}))

	if query["machine_group_ids"] != "[]" || query["entity_name"] != "dossié 15 être" {
		t.Fatalf("unexpected empty group query: %#v", query)
	}
}

func mustFindAction(t *testing.T, id string) Action {
	t.Helper()
	action, ok := FindAction(id)
	if !ok {
		t.Fatalf("missing action %s", id)
	}
	return action
}

func mustFindField(t *testing.T, action Action, name string) ActionField {
	t.Helper()
	for _, field := range action.Fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("missing field %s on action %s", name, action.ID)
	return ActionField{}
}
