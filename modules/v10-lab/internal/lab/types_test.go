package lab

import "testing"

func TestValidateConfigAcceptsExampleShape(t *testing.T) {
	config := Config{
		Name:    "ticket-T5808",
		Product: ProductGedixV10,
		Pipeline: []PipelineStep{
			{
				Action: "create-env",
				Params: map[string]any{
					"releasePath": "D:/release",
					"targetPath":  "D:/target",
				},
			},
			{
				Action: "create-machine-group",
				Params: map[string]any{
					"code": "FRAISAGE",
					"name": "Groupe Fraisage",
				},
			},
		},
	}

	if err := ValidateConfig(config); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConfigReportsUnknownActionAndMissingField(t *testing.T) {
	config := Config{
		Name:    "ticket-T5808",
		Product: ProductGedixV10,
		Pipeline: []PipelineStep{
			{Action: "create-foo"},
			{Action: "create-machine", Params: map[string]any{"name": "FANUC"}},
		},
	}

	err := ValidateConfig(config)
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T %v", err, err)
	}
	if len(validationErr.Items) != 2 {
		t.Fatalf("expected 2 validation items, got %#v", validationErr.Items)
	}
}

func TestActionsForProductIncludesSystemAndGedixActions(t *testing.T) {
	actions := ActionsForProduct(ProductGedixV10)
	byID := map[string]bool{}
	for _, action := range actions {
		byID[action.ID] = true
	}

	for _, id := range []string{"create-env", "start-services", "stop-services", "create-machine-group", "create-machine", "create-cnc-folder"} {
		if !byID[id] {
			t.Fatalf("expected action %s in product actions", id)
		}
	}
}
