package lab

import (
	"fmt"
	"net/http"
)

const createMachinePath = "/entreprise/api/v1/machines"

type CreateMachinePayload struct {
	EntityName                            string                `json:"entity_name"`
	Description                           string                `json:"description"`
	CharsEOL                              string                `json:"chars_eol"`
	IsFileDeletionAllowed                 bool                  `json:"is_file_deletion_allowed"`
	IsFileViewingAllowed                  bool                  `json:"is_file_viewing_allowed"`
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
