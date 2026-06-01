package lab

import (
	"fmt"
	"net/http"
)

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
