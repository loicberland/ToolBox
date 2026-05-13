package service

import (
	"errors"
	"testing"

	"toolBox/modules/test-sheet/pkg/model"
)

func TestPlanNameMustBeUniqueIncludingHiddenPlans(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "T5353"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePlan(model.PlanInput{Name: " t5353 "}); !isNameConflict(err, "plan") {
		t.Fatalf("expected plan name conflict, got %v", err)
	}
	if err := svc.DeletePlan(plan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePlan(model.PlanInput{Name: "T5353"}); !isHiddenNameConflict(err, "plan") {
		t.Fatalf("expected hidden plan name conflict, got %v", err)
	}
}

func TestUpdatePlanRejectsExistingName(t *testing.T) {
	svc := newTestService(t)
	first, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreatePlan(model.PlanInput{Name: "Plan B"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdatePlan(second.ID, model.PlanInput{Name: " plan   a "}); !isNameConflict(err, "plan") {
		t.Fatalf("expected plan name conflict with %s, got %v", first.Name, err)
	}
}

func TestImportPlanRejectsExistingName(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan source"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := svc.ExportPlan(plan.ID, DefaultExportOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportPlanZip(payload, " plan source "); !isNameConflict(err, "plan") {
		t.Fatalf("expected import plan name conflict, got %v", err)
	}
	result, err := svc.ImportPlanZip(payload, "Plan source client A")
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Plan source client A" {
		t.Fatalf("unexpected imported name %q", result.Name)
	}
}

func TestGroupNameMustBeUniqueInsidePlanOnly(t *testing.T) {
	svc := newTestService(t)
	planA, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}
	planB, err := svc.CreatePlan(model.PlanInput{Name: "Plan B"})
	if err != nil {
		t.Fatal(err)
	}
	groupA, err := svc.CreateGroup(planA.ID, model.GroupInput{Name: "Installation"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateGroup(planA.ID, model.GroupInput{Name: " installation "}); !isNameConflict(err, "group") {
		t.Fatalf("expected group name conflict, got %v", err)
	}
	if _, err := svc.CreateGroup(planB.ID, model.GroupInput{Name: "Installation"}); err != nil {
		t.Fatalf("same group name in another plan should be allowed: %v", err)
	}
	groups, err := svc.ListGroups(planA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateGroup(groupA.ID, model.GroupInput{Name: groups[0].Name}); !isNameConflict(err, "group") {
		t.Fatalf("expected group update conflict, got %v", err)
	}
}

func TestSheetNameMustBeUniqueInsideGroupOnly(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan"})
	if err != nil {
		t.Fatal(err)
	}
	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	groupB, err := svc.CreateGroup(plan.ID, model.GroupInput{Name: "Desinstallation"})
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := svc.CreateSheetInGroup(groups[0].ID, model.SheetInput{Name: "Test connexion"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSheetInGroup(groups[0].ID, model.SheetInput{Name: " test   connexion "}); !isNameConflict(err, "sheet") {
		t.Fatalf("expected sheet name conflict, got %v", err)
	}
	if _, err := svc.CreateSheetInGroup(groupB.ID, model.SheetInput{Name: "Test connexion"}); err != nil {
		t.Fatalf("same sheet name in another group should be allowed: %v", err)
	}
	other, err := svc.CreateSheetInGroup(groups[0].ID, model.SheetInput{Name: "Autre fiche"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateSheet(other.ID, model.SheetInput{Name: sheet.Name}); !isNameConflict(err, "sheet") {
		t.Fatalf("expected sheet update conflict, got %v", err)
	}
}

func TestDuplicatePlanUsesNextAvailableCopyName(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "T5353"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.DuplicatePlan(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "T5353 (copie)" {
		t.Fatalf("unexpected first copy name %q", first.Name)
	}
	second, err := svc.DuplicatePlan(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Name != "T5353 (copie 2)" {
		t.Fatalf("unexpected second copy name %q", second.Name)
	}
}

func TestDuplicateGroupAndSheetUseNextAvailableCopyName(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan"})
	if err != nil {
		t.Fatal(err)
	}
	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	groupCopy, err := svc.DuplicateGroup(groups[0].ID, model.DuplicateGroupInput{})
	if err != nil {
		t.Fatal(err)
	}
	if groupCopy.Name != "Sous-plan principal (copie)" {
		t.Fatalf("unexpected group copy name %q", groupCopy.Name)
	}
	groupCopy2, err := svc.DuplicateGroup(groups[0].ID, model.DuplicateGroupInput{})
	if err != nil {
		t.Fatal(err)
	}
	if groupCopy2.Name != "Sous-plan principal (copie 2)" {
		t.Fatalf("unexpected second group copy name %q", groupCopy2.Name)
	}

	sheet, err := svc.CreateSheetInGroup(groups[0].ID, model.SheetInput{Name: "Fiche"})
	if err != nil {
		t.Fatal(err)
	}
	sheetCopy, err := svc.DuplicateSheet(sheet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sheetCopy.Name != "Fiche (copie)" {
		t.Fatalf("unexpected sheet copy name %q", sheetCopy.Name)
	}
	sheetCopy2, err := svc.DuplicateSheet(sheet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sheetCopy2.Name != "Fiche (copie 2)" {
		t.Fatalf("unexpected second sheet copy name %q", sheetCopy2.Name)
	}
}

func isNameConflict(err error, conflictType string) bool {
	var conflict NameConflictError
	return errors.As(err, &conflict) && conflict.ConflictType == conflictType && !conflict.Hidden
}

func isHiddenNameConflict(err error, conflictType string) bool {
	var conflict NameConflictError
	return errors.As(err, &conflict) && conflict.ConflictType == conflictType && conflict.Hidden
}
