package lab

import (
	"fmt"
	"net/http"
)

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
