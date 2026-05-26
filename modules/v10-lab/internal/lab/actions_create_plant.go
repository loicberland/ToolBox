package lab

import (
	"encoding/json"
	"fmt"
)

func ExecuteCreatePlant() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createPlantPayloadFromParams(params)
		if err := client.CreatePlant(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Usine créée avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func createPlantPayloadFromParams(params map[string]any) CreatePlantPayload {
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

func numberParam(params map[string]any, key string) any {
	switch value := params[key].(type) {
	case int:
		return value
	case int64:
		return value
	case float64:
		if value == float64(int64(value)) {
			return int64(value)
		}
		return value
	case json.Number:
		if integer, err := value.Int64(); err == nil {
			return integer
		}
		if decimal, err := value.Float64(); err == nil {
			return decimal
		}
	}
	return params[key]
}
