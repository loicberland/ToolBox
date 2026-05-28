package lab

import (
	"fmt"
	"net/http"
)

const createMachiningJobPath = "/entreprise/api/v1/machining_jobs/actions/create_new"

func ExecuteCreateMachiningJob() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
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
