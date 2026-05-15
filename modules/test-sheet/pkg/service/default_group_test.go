package service

import (
	"testing"

	"toolBox/modules/test-sheet/pkg/model"
)

func TestCreatePlanCreatesOneDefaultGroup(t *testing.T) {
	svc := newTestService(t)

	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}

	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected one default group, got %+v", groups)
	}
	assertDefaultGroup(t, groups[0])
}

func TestCreateGroupAfterCreatePlanAddsOnlyRequestedGroup(t *testing.T) {
	svc := newTestService(t)

	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := svc.CreateGroup(plan.ID, model.GroupInput{Name: "Sous-plan 2"})
	if err != nil {
		t.Fatal(err)
	}

	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected exactly two groups, got %+v", groups)
	}
	assertDefaultGroup(t, groups[0])
	if groups[1].ID != group.ID || groups[1].Name != "Sous-plan 2" {
		t.Fatalf("manual group was not the only additional group: created=%+v groups=%+v", group, groups)
	}
}

func TestCreateGroupDoesNotDuplicateDefaultGroup(t *testing.T) {
	svc := newTestService(t)

	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateGroup(plan.ID, model.GroupInput{Name: "Sous-plan 2"}); err != nil {
		t.Fatal(err)
	}

	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	defaultCount := 0
	for _, group := range groups {
		if group.Name == defaultGroupName {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		t.Fatalf("expected one default group, got %d in %+v", defaultCount, groups)
	}
}

func TestEnsureDefaultGroupIsIdempotent(t *testing.T) {
	svc := newTestService(t)

	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan A"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ensureDefaultGroup(plan.ID); err != nil {
		t.Fatal(err)
	}

	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected idempotent default group creation, got %+v", groups)
	}
	assertDefaultGroup(t, groups[0])
}

func assertDefaultGroup(t *testing.T, group model.TestGroup) {
	t.Helper()
	if group.Name != defaultGroupName {
		t.Fatalf("unexpected default group name %q", group.Name)
	}
	if group.Description != "" {
		t.Fatalf("unexpected default group description %q", group.Description)
	}
	if group.ExecutionOrder != 1 {
		t.Fatalf("unexpected default group execution order %d", group.ExecutionOrder)
	}
}
