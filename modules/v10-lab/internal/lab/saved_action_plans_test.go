package lab

import (
	"encoding/json"
	"os"
	"testing"

	"toolBox/pkg/toolboxruntime"
)

func TestListSavedActionPlansEmptyWhenFileMissing(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	items, err := ListSavedActionPlans("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %#v", items)
	}
}

func TestSaveActionPlanKeepsVisibleConditionalFields(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	plan, err := SaveActionPlan(SaveActionPlanInput{
		Name:      "Commande active",
		ProductID: GedixProdV10,
		Actions: []PipelineStep{{Action: "create-machine", Label: "Machine", Params: map[string]any{
			"entity_name":                                "Machine",
			"has_command_program":                        true,
			"command_program_name":                       "CMD",
			"command_program_regexp":                     "LOAD",
			"wait_between_command_program_check_seconds": float64(45),
			"description":                                "visible stays",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	params := plan.Actions[0].Params
	if params["command_program_name"] != "CMD" || params["command_program_regexp"] != "LOAD" || params["wait_between_command_program_check_seconds"] != float64(45) {
		t.Fatalf("visible conditional fields were not saved: %#v", params)
	}
	if params["description"] != "visible stays" || params["has_command_program"] != true {
		t.Fatalf("unrelated fields changed: %#v", params)
	}
}

func TestSaveActionPlanRemovesHiddenConditionalFields(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	plan, err := SaveActionPlan(SaveActionPlanInput{
		Name:      "Commande inactive",
		ProductID: GedixProdV10,
		Actions: []PipelineStep{{Action: "create-machine", Label: "Machine", Params: map[string]any{
			"entity_name":                                "Machine",
			"has_command_program":                        false,
			"command_program_name":                       "old",
			"command_program_regexp":                     "old-regexp",
			"wait_between_command_program_check_seconds": float64(45),
			"description":                                "kept",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	params := plan.Actions[0].Params
	for _, key := range []string{"command_program_name", "command_program_regexp", "wait_between_command_program_check_seconds"} {
		if _, exists := params[key]; exists {
			t.Fatalf("hidden field %s should be removed: %#v", key, params)
		}
	}
	if params["description"] != "kept" || params["entity_name"] != "Machine" || params["has_command_program"] != false {
		t.Fatalf("visible fields should stay unchanged: %#v", params)
	}

	raw := readActionPlansJSON(t)
	savedParams := raw["plans"].([]any)[0].(map[string]any)["actions"].([]any)[0].(map[string]any)["params"].(map[string]any)
	if _, exists := savedParams["command_program_name"]; exists {
		t.Fatalf("hidden field should not be written to JSON: %#v", savedParams)
	}
}

func TestListSavedActionPlansNormalizesLegacyHiddenFields(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())
	mustWrite(t, SavedActionPlansPath(), `{
  "plans": [
    {
      "id": "legacy",
      "name": "Legacy",
      "productId": "gedix-prod-v10",
      "actions": [
        {
          "action": "create-machine",
          "label": "Machine",
          "params": {
            "entity_name": "Machine",
            "has_command_program": false,
            "command_program_name": "old"
          }
        }
      ],
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-01T00:00:00Z"
    }
  ]
}`)

	items, err := ListSavedActionPlans(GedixProdV10)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := items[0].Actions[0].Params["command_program_name"]; exists {
		t.Fatalf("legacy hidden value should be removed on load: %#v", items[0].Actions[0].Params)
	}
}

func TestSaveActionPlanAcceptsAbsentHiddenConditionalField(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	plan, err := SaveActionPlan(SaveActionPlanInput{
		Name:      "Absent hidden",
		ProductID: GedixProdV10,
		Actions: []PipelineStep{{Action: "create-machine", Params: map[string]any{
			"entity_name":         "Machine",
			"has_command_program": false,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := plan.Actions[0].Params["command_program_name"]; exists {
		t.Fatalf("absent hidden field should stay absent: %#v", plan.Actions[0].Params)
	}
}

func readActionPlansJSON(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(SavedActionPlansPath())
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestSaveActionPlanAndList(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	plan, err := SaveActionPlan(SaveActionPlanInput{
		Name:      "Initialisation Prod V10",
		ProductID: GedixProdV10,
		Actions:   []PipelineStep{{Action: "create-workshop", Label: "Atelier", Params: map[string]any{"entity_name": "Atelier"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID == "" || plan.Name != "Initialisation Prod V10" || len(plan.Actions) != 1 {
		t.Fatalf("unexpected saved plan: %#v", plan)
	}

	items, err := ListSavedActionPlans(GedixProdV10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != plan.ID || items[0].Actions[0].Action != "create-workshop" {
		t.Fatalf("unexpected plans: %#v", items)
	}
	if _, err := os.Stat(SavedActionPlansPath()); err != nil {
		t.Fatalf("expected registry file: %v", err)
	}
}

func TestSaveActionPlanDuplicateAndOverwrite(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	first, err := SaveActionPlan(SaveActionPlanInput{Name: "Cycle", ProductID: GedixProdV10, Actions: []PipelineStep{{Action: "create-target"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveActionPlan(SaveActionPlanInput{Name: "cycle", ProductID: GedixProdV10}); err == nil {
		t.Fatal("expected duplicate error")
	}
	second, err := SaveActionPlan(SaveActionPlanInput{Name: "Cycle", ProductID: GedixProdV10, Actions: []PipelineStep{{Action: "create-machine"}}, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.CreatedAt != first.CreatedAt {
		t.Fatalf("overwrite should preserve id and createdAt: first=%#v second=%#v", first, second)
	}
	items, err := ListSavedActionPlans(GedixProdV10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Actions) != 1 || items[0].Actions[0].Action != "create-machine" {
		t.Fatalf("unexpected overwritten plan: %#v", items)
	}
}

func TestDeleteSavedActionPlan(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	plan, err := SaveActionPlan(SaveActionPlanInput{Name: "A supprimer", ProductID: GedixProdV10})
	if err != nil {
		t.Fatal(err)
	}
	if err := DeleteSavedActionPlan(plan.ID); err != nil {
		t.Fatal(err)
	}
	items, err := ListSavedActionPlans(GedixProdV10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected deleted plan, got %#v", items)
	}
}

func TestListSavedActionPlansFiltersProduct(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())

	if _, err := SaveActionPlan(SaveActionPlanInput{Name: "Prod", ProductID: GedixProdV10}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveActionPlan(SaveActionPlanInput{Name: "Stock", ProductID: GedixToolStockV10}); err != nil {
		t.Fatal(err)
	}
	items, err := ListSavedActionPlans(GedixProdV10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "Prod" {
		t.Fatalf("unexpected filtered plans: %#v", items)
	}
}
