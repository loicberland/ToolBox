package lab

import "net/http"

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

func (c *GedixAPIClient) CreatePlant(payload CreatePlantPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer une usine",
		Method:           http.MethodPost,
		Path:             createPlantPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}
