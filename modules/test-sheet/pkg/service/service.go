package service

import (
	"fmt"
	"strings"

	"toolBox/modules/test-sheet/pkg/model"
	"toolBox/modules/test-sheet/pkg/repository"
)

type Repository interface {
	CreatePlan(model.PlanInput) (model.TestPlan, error)
	ListPlans() ([]model.TestPlan, error)
	GetPlan(int64) (model.TestPlan, error)
	UpdatePlan(int64, model.PlanInput) (model.TestPlan, error)
	DeletePlan(int64) error
	CreateSheet(int64, model.SheetInput) (model.TestSheet, error)
	ListSheets(int64) ([]model.TestSheet, error)
	GetSheet(int64) (model.TestSheet, error)
	UpdateSheet(int64, model.SheetInput) (model.TestSheet, error)
	DeleteSheet(int64) error
	ReorderSheets(int64, []int64) error
	CreateRunWithSnapshot(int64) (model.TestRun, error)
	GetRun(int64) (model.TestRun, error)
	UpdateRunSheet(int64, int64, model.RunSheetResultInput) (model.RunSheet, error)
	FinishRun(int64) (model.TestRun, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreatePlan(input model.PlanInput) (model.TestPlan, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestPlan{}, fmt.Errorf("plan name is required")
	}
	return s.repo.CreatePlan(input)
}

func (s *Service) ListPlans() ([]model.TestPlan, error) {
	return s.repo.ListPlans()
}

func (s *Service) GetPlan(id int64) (model.TestPlan, error) {
	return s.repo.GetPlan(id)
}

func (s *Service) UpdatePlan(id int64, input model.PlanInput) (model.TestPlan, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestPlan{}, fmt.Errorf("plan name is required")
	}
	return s.repo.UpdatePlan(id, input)
}

func (s *Service) DeletePlan(id int64) error {
	return s.repo.DeletePlan(id)
}

func (s *Service) DuplicatePlan(id int64) (model.TestPlan, error) {
	plan, err := s.repo.GetPlan(id)
	if err != nil {
		return model.TestPlan{}, err
	}
	sheets, err := s.repo.ListSheets(id)
	if err != nil {
		return model.TestPlan{}, err
	}
	copyPlan, err := s.repo.CreatePlan(model.PlanInput{
		Name:           plan.Name + " (copie)",
		Description:    plan.Description,
		MockupSettings: plan.MockupSettings,
	})
	if err != nil {
		return model.TestPlan{}, err
	}
	for _, sheet := range sheets {
		_, err := s.repo.CreateSheet(copyPlan.ID, model.SheetInput{
			Name:           sheet.Name,
			Description:    sheet.Description,
			Prerequisites:  sheet.Prerequisites,
			Action:         sheet.Action,
			ExpectedResult: sheet.ExpectedResult,
			ExecutionOrder: sheet.ExecutionOrder,
			MockupSettings: sheet.MockupSettings,
		})
		if err != nil {
			return model.TestPlan{}, err
		}
	}
	return copyPlan, nil
}

func (s *Service) CreateSheet(planID int64, input model.SheetInput) (model.TestSheet, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestSheet{}, fmt.Errorf("sheet name is required")
	}
	if _, err := s.repo.GetPlan(planID); err != nil {
		return model.TestSheet{}, err
	}
	return s.repo.CreateSheet(planID, input)
}

func (s *Service) ListSheets(planID int64) ([]model.TestSheet, error) {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return nil, err
	}
	return s.repo.ListSheets(planID)
}

func (s *Service) UpdateSheet(id int64, input model.SheetInput) (model.TestSheet, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestSheet{}, fmt.Errorf("sheet name is required")
	}
	return s.repo.UpdateSheet(id, input)
}

func (s *Service) DeleteSheet(id int64) error {
	return s.repo.DeleteSheet(id)
}

func (s *Service) DuplicateSheet(id int64) (model.TestSheet, error) {
	sheet, err := s.repo.GetSheet(id)
	if err != nil {
		return model.TestSheet{}, err
	}
	return s.repo.CreateSheet(sheet.PlanID, model.SheetInput{
		Name:           sheet.Name + " (copie)",
		Description:    sheet.Description,
		Prerequisites:  sheet.Prerequisites,
		Action:         sheet.Action,
		ExpectedResult: sheet.ExpectedResult,
		MockupSettings: sheet.MockupSettings,
	})
}

func (s *Service) ReorderSheets(planID int64, sheetIDs []int64) error {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return err
	}
	return s.repo.ReorderSheets(planID, sheetIDs)
}

func (s *Service) CreateRun(planID int64) (model.TestRun, error) {
	sheets, err := s.repo.ListSheets(planID)
	if err != nil {
		return model.TestRun{}, err
	}
	if len(sheets) == 0 {
		return model.TestRun{}, fmt.Errorf("cannot start a run without sheets")
	}
	return s.repo.CreateRunWithSnapshot(planID)
}

func (s *Service) GetRun(runID int64) (model.TestRun, error) {
	return s.repo.GetRun(runID)
}

func (s *Service) UpdateRunSheet(runID, runSheetID int64, input model.RunSheetResultInput) (model.RunSheet, error) {
	if !isAllowedStatus(input.Status) {
		return model.RunSheet{}, fmt.Errorf("invalid run sheet status")
	}
	return s.repo.UpdateRunSheet(runID, runSheetID, input)
}

func (s *Service) FinishRun(runID int64) (model.TestRun, error) {
	return s.repo.FinishRun(runID)
}

func (s *Service) GenerateMarkdownReport(runID int64) (string, error) {
	run, err := s.repo.GetRun(runID)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Rapport de test - %s\n\n", run.PlanName)
	fmt.Fprintf(&builder, "- Execution: #%d\n", run.ID)
	fmt.Fprintf(&builder, "- Statut: %s\n", run.Status)
	fmt.Fprintf(&builder, "- Demarree le: %s\n", run.StartedAt.Format("2006-01-02 15:04"))
	if run.FinishedAt != nil {
		fmt.Fprintf(&builder, "- Terminee le: %s\n", run.FinishedAt.Format("2006-01-02 15:04"))
	}
	builder.WriteString("\n## Synthese\n\n")
	counts := map[string]int{}
	for _, sheet := range run.Sheets {
		counts[sheet.Status]++
	}
	for _, status := range []string{model.RunSheetStatusPending, model.RunSheetStatusPassed, model.RunSheetStatusFailed, model.RunSheetStatusBlocked, model.RunSheetStatusSkipped} {
		fmt.Fprintf(&builder, "- %s: %d\n", status, counts[status])
	}
	builder.WriteString("\n## Fiches executees\n\n")
	for _, sheet := range run.Sheets {
		fmt.Fprintf(&builder, "### %d. %s\n\n", sheet.ExecutionOrder, sheet.Name)
		fmt.Fprintf(&builder, "- Statut: %s\n", sheet.Status)
		writeReportLine(&builder, "Description", sheet.Description)
		writeReportLine(&builder, "Prerequis", sheet.Prerequisites)
		writeReportLine(&builder, "Action", sheet.Action)
		writeReportLine(&builder, "Resultat attendu", sheet.ExpectedResult)
		writeReportLine(&builder, "Resultat reel", sheet.ActualResult)
		writeReportLine(&builder, "Commentaire", sheet.Comment)
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func writeReportLine(builder *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(builder, "- %s: %s\n", label, strings.ReplaceAll(value, "\n", " "))
}

func isAllowedStatus(status string) bool {
	switch status {
	case model.RunSheetStatusPending, model.RunSheetStatusPassed, model.RunSheetStatusFailed, model.RunSheetStatusBlocked, model.RunSheetStatusSkipped:
		return true
	default:
		return false
	}
}

func IsNotFound(err error) bool {
	return repository.IsNotFound(err)
}
