package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"toolBox/pkg/toolboxruntime"
)

type SavedActionPlan struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ProductID   string         `json:"productId,omitempty"`
	Description string         `json:"description,omitempty"`
	Actions     []PipelineStep `json:"actions"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

type SaveActionPlanInput struct {
	Name        string         `json:"name"`
	ProductID   string         `json:"productId,omitempty"`
	Description string         `json:"description,omitempty"`
	Actions     []PipelineStep `json:"actions"`
	Overwrite   bool           `json:"overwrite,omitempty"`
}

type actionPlanRegistry struct {
	Plans []SavedActionPlan `json:"plans"`
}

func SavedActionPlansPath() string {
	layout, err := toolboxruntime.ForModule(ModuleID)
	if err != nil {
		return filepath.Join("data", ModuleID, "action_plans.json")
	}
	return filepath.Join(layout.DataDir, "action_plans.json")
}

func ListSavedActionPlans(productID string) ([]SavedActionPlan, error) {
	registry, err := loadActionPlanRegistry()
	if err != nil {
		return nil, err
	}
	productID = strings.TrimSpace(productID)
	items := []SavedActionPlan{}
	for _, plan := range registry.Plans {
		if productID != "" && NormalizeProductID(plan.ProductID) != NormalizeProductID(productID) {
			continue
		}
		items = append(items, plan)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func SaveActionPlan(input SaveActionPlanInput) (SavedActionPlan, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return SavedActionPlan{}, fmt.Errorf("nom de plan d'actions requis")
	}
	productID := NormalizeProductID(input.ProductID)
	registry, err := loadActionPlanRegistry()
	if err != nil {
		return SavedActionPlan{}, err
	}
	now := time.Now().Format(time.RFC3339)
	next := SavedActionPlan{
		ID:          actionPlanBaseID(name, productID),
		Name:        name,
		ProductID:   productID,
		Description: strings.TrimSpace(input.Description),
		Actions:     normalizePipelineStepsForSave(input.Actions),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	for index, plan := range registry.Plans {
		if sameActionPlanIdentity(plan, name, productID) {
			if !input.Overwrite {
				return SavedActionPlan{}, fmt.Errorf("un plan d'actions avec ce nom existe deja")
			}
			next.ID = plan.ID
			next.CreatedAt = firstNonEmpty(plan.CreatedAt, now)
			registry.Plans[index] = next
			return next, saveActionPlanRegistry(registry)
		}
	}
	next.ID = uniqueActionPlanID(registry.Plans, next.ID)
	registry.Plans = append(registry.Plans, next)
	return next, saveActionPlanRegistry(registry)
}

func DeleteSavedActionPlan(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id de plan d'actions requis")
	}
	registry, err := loadActionPlanRegistry()
	if err != nil {
		return err
	}
	next := registry.Plans[:0]
	found := false
	for _, plan := range registry.Plans {
		if plan.ID == id {
			found = true
			continue
		}
		next = append(next, plan)
	}
	if !found {
		return os.ErrNotExist
	}
	registry.Plans = next
	return saveActionPlanRegistry(registry)
}

func loadActionPlanRegistry() (actionPlanRegistry, error) {
	path := SavedActionPlansPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return actionPlanRegistry{Plans: []SavedActionPlan{}}, nil
	}
	if err != nil {
		return actionPlanRegistry{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return actionPlanRegistry{Plans: []SavedActionPlan{}}, nil
	}
	var registry actionPlanRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return actionPlanRegistry{}, err
	}
	if registry.Plans == nil {
		registry.Plans = []SavedActionPlan{}
	}
	for index := range registry.Plans {
		registry.Plans[index].Actions = normalizePipelineStepsForSave(registry.Plans[index].Actions)
	}
	return registry, nil
}

func saveActionPlanRegistry(registry actionPlanRegistry) error {
	path := SavedActionPlansPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	sort.Slice(registry.Plans, func(i, j int) bool {
		return strings.ToLower(registry.Plans[i].Name) < strings.ToLower(registry.Plans[j].Name)
	})
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "action_plans-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	if _, err := temp.Write(append(payload, '\n')); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	return nil
}

func sameActionPlanIdentity(plan SavedActionPlan, name string, productID string) bool {
	return strings.EqualFold(strings.TrimSpace(plan.Name), name) && NormalizeProductID(plan.ProductID) == productID
}

func actionPlanBaseID(name string, productID string) string {
	base := strings.ToLower(safeDirName(strings.TrimSpace(productID + "-" + name)))
	return base
}

func uniqueActionPlanID(plans []SavedActionPlan, base string) string {
	used := map[string]bool{}
	for _, plan := range plans {
		used[plan.ID] = true
	}
	if !used[base] {
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", base, index)
		if !used[candidate] {
			return candidate
		}
	}
}

func clonePipelineSteps(steps []PipelineStep) []PipelineStep {
	if steps == nil {
		return []PipelineStep{}
	}
	payload, err := json.Marshal(steps)
	if err != nil {
		return append([]PipelineStep{}, steps...)
	}
	var cloned []PipelineStep
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return append([]PipelineStep{}, steps...)
	}
	if cloned == nil {
		return []PipelineStep{}
	}
	return cloned
}
