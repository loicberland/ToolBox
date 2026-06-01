package lab

import (
	"fmt"
	"net/http"
)

const createPlantPath = "/entreprise/api/v1/plants"

type CreatePlantPayload struct {
	EntityName        string `json:"entity_name"`
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
		Description:       stringParam(params, "description"),
		AddressName:       stringParam(params, "address_name"),
		AddressStreet:     stringParam(params, "address_street"),
		AddressPostalCode: stringParam(params, "address_postalcode"),
		AddressTown:       stringParam(params, "address_town"),
		AddressCountry:    stringParam(params, "address_country"),
		CreatedBy:         numberParam(params, "created_by"),
	}
}
