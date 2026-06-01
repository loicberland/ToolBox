package lab

import (
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
