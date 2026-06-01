package lab

import (
	"fmt"
	"net/http"
	"strings"
)

const createTargetPath = "/dnc/api/v1/targets"

type CreateTargetPayload struct {
	EntityName    string             `json:"entity_name"`
	Description   string             `json:"description"`
	ConnectorName string             `json:"connector_name"`
	Configs       []TargetConfig     `json:"configs"`
	TunnelSteps   []TargetTunnelStep `json:"tunnel_steps"`
	CreatedBy     any                `json:"created_by"`
}

type TargetConfig struct {
	ModuleValue string `json:"module_value"`
	ModuleKey   string `json:"module_key"`
}

type TargetTunnelStep struct {
	EntityName string `json:"entity_name"`
	Rank       any    `json:"rank,omitempty"`
}

func ExecuteCreateTarget() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		if err := validateTargetConfigs(params); err != nil {
			return err
		}
		client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
		if err != nil {
			return err
		}
		payload := createTargetPayload(params)
		if err := client.CreateTarget(payload); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Cible créée avec succès : %s\n", payload.EntityName)
		return nil
	}
}

func validateTargetConfigs(params map[string]any) error {
	allowed := map[string]bool{
		"remote-filepath":     true,
		"subprogram-filepath": true,
	}
	seen := map[string]bool{}
	for _, row := range objectArrayParam(params, "configs") {
		moduleKey := strings.TrimSpace(fmt.Sprint(row["module_key"]))
		if moduleKey == "" {
			continue
		}
		if !allowed[moduleKey] {
			return fmt.Errorf("la clé module %q n'est pas autorisée dans configs", moduleKey)
		}
		if seen[moduleKey] {
			return fmt.Errorf("la clé module %q est utilisée plusieurs fois dans configs", moduleKey)
		}
		seen[moduleKey] = true
	}
	return nil
}

func (c *GedixAPIClient) CreateTarget(payload CreateTargetPayload) error {
	return c.DoJSON(GedixAPIRequest{
		Name:             "Créer cible",
		Method:           http.MethodPost,
		Path:             createTargetPath,
		Body:             payload,
		ExpectedStatuses: []int{http.StatusOK},
	})
}

func createTargetPayload(params map[string]any) CreateTargetPayload {
	return CreateTargetPayload{
		EntityName:    stringParam(params, "entity_name"),
		Description:   stringParam(params, "description"),
		ConnectorName: stringParam(params, "connector_name"),
		Configs:       targetConfigsParam(params, "configs"),
		TunnelSteps:   targetTunnelStepsParam(params, "tunnel_steps"),
		CreatedBy:     numberParam(params, "created_by"),
	}
}

func targetConfigsParam(params map[string]any, key string) []TargetConfig {
	rows := objectArrayParam(params, key)
	items := []TargetConfig{}
	seen := map[string]bool{}
	for _, row := range rows {
		moduleKey := strings.TrimSpace(fmt.Sprint(row["module_key"]))
		if moduleKey == "" || seen[moduleKey] {
			continue
		}
		seen[moduleKey] = true
		items = append(items, TargetConfig{
			ModuleKey:   moduleKey,
			ModuleValue: strings.TrimSpace(fmt.Sprint(row["module_value"])),
		})
	}
	return items
}

func targetTunnelStepsParam(params map[string]any, key string) []TargetTunnelStep {
	rows := objectArrayParam(params, key)
	items := []TargetTunnelStep{}
	for index, row := range rows {
		entityName := strings.TrimSpace(fmt.Sprint(row["entity_name"]))
		if entityName == "" {
			continue
		}
		step := TargetTunnelStep{EntityName: entityName}
		if rank, ok := anyToInt(row["rank"]); ok && rank > 0 {
			step.Rank = rank
		} else {
			step.Rank = index + 1
		}
		items = append(items, step)
	}
	return items
}
