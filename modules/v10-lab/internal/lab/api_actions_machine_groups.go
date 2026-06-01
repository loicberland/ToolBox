package lab

import (
	"fmt"
	"net/http"
)

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
