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
	TouchPlan(int64) error
	UpdatePlan(int64, model.PlanInput) (model.TestPlan, error)
	DeletePlan(int64) error
	PermanentDeletePlan(int64) error
	RestorePlan(int64) (model.TestPlan, error)
	CreateSheet(int64, model.SheetInput) (model.TestSheet, error)
	ListSheets(int64) ([]model.TestSheet, error)
	GetSheet(int64) (model.TestSheet, error)
	UpdateSheet(int64, model.SheetInput) (model.TestSheet, error)
	DeleteSheet(int64) error
	CreateStep(int64, model.StepInput) (model.TestSheetStep, error)
	ListSteps(int64) ([]model.TestSheetStep, error)
	GetStep(int64) (model.TestSheetStep, error)
	UpdateStep(int64, model.StepInput) (model.TestSheetStep, error)
	DeleteStep(int64) error
	DuplicateStep(int64) (model.TestSheetStep, error)
	ReorderSteps(int64, []int64) error
	ReorderSheets(int64, []int64) error
	CreateRunWithSnapshot(int64) (model.TestRun, error)
	GetRun(int64) (model.TestRun, error)
	ListPlanRuns(int64) ([]model.TestRunSummary, error)
	ListRunSummaries() ([]model.TestRunSummary, error)
	ListPlanSummaries(bool) ([]model.TestPlanSummary, error)
	ReplayRun(int64) (model.TestRun, error)
	ArchiveRun(int64) (model.TestRun, error)
	CancelRun(int64) (model.TestRun, error)
	UpdateRunSheet(int64, int64, model.RunSheetResultInput) (model.RunSheet, error)
	UpdateRunStep(int64, int64, model.RunStepResultInput) (model.RunStep, error)
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
	plan, err := s.repo.UpdatePlan(id, input)
	if err != nil {
		return model.TestPlan{}, err
	}
	if err := s.markPlanChanged(id); err != nil {
		return model.TestPlan{}, err
	}
	return s.repo.GetPlan(plan.ID)
}

func (s *Service) DeletePlan(id int64) error {
	if err := s.repo.DeletePlan(id); err != nil {
		return err
	}
	return s.cancelRunningRunsForPlan(id)
}

func (s *Service) PermanentDeletePlan(id int64) error {
	return s.repo.PermanentDeletePlan(id)
}

func (s *Service) RestorePlan(id int64) (model.TestPlan, error) {
	return s.repo.RestorePlan(id)
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
		copySheet, err := s.repo.CreateSheet(copyPlan.ID, model.SheetInput{
			Name:           sheet.Name,
			Description:    sheet.Description,
			Prerequisites:  sheet.Prerequisites,
			Config:         sheet.Config,
			Command:        sheet.Command,
			Notes:          sheet.Notes,
			Action:         sheet.Action,
			ExpectedResult: sheet.ExpectedResult,
			ExecutionOrder: sheet.ExecutionOrder,
			MockupSettings: sheet.MockupSettings,
		})
		if err != nil {
			return model.TestPlan{}, err
		}
		for _, step := range sheet.Steps {
			_, err := s.repo.CreateStep(copySheet.ID, model.StepInput{
				Action:         step.Action,
				Field:          step.Field,
				ExpectedResult: step.ExpectedResult,
				ExecutionOrder: step.ExecutionOrder,
			})
			if err != nil {
				return model.TestPlan{}, err
			}
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
	sheet, err := s.repo.CreateSheet(planID, input)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markPlanChanged(planID); err != nil {
		return model.TestSheet{}, err
	}
	return sheet, nil
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
	sheet, err := s.repo.GetSheet(id)
	if err != nil {
		return model.TestSheet{}, err
	}
	updated, err := s.repo.UpdateSheet(id, input)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markPlanChanged(sheet.PlanID); err != nil {
		return model.TestSheet{}, err
	}
	return updated, nil
}

func (s *Service) DeleteSheet(id int64) error {
	sheet, err := s.repo.GetSheet(id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteSheet(id); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) DuplicateSheet(id int64) (model.TestSheet, error) {
	sheet, err := s.repo.GetSheet(id)
	if err != nil {
		return model.TestSheet{}, err
	}
	copySheet, err := s.repo.CreateSheet(sheet.PlanID, model.SheetInput{
		Name:           sheet.Name + " (copie)",
		Description:    sheet.Description,
		Prerequisites:  sheet.Prerequisites,
		Config:         sheet.Config,
		Command:        sheet.Command,
		Notes:          sheet.Notes,
		Action:         sheet.Action,
		ExpectedResult: sheet.ExpectedResult,
		MockupSettings: sheet.MockupSettings,
	})
	if err != nil {
		return model.TestSheet{}, err
	}
	for _, step := range sheet.Steps {
		_, err := s.repo.CreateStep(copySheet.ID, model.StepInput{
			Action:         step.Action,
			Field:          step.Field,
			ExpectedResult: step.ExpectedResult,
			ExecutionOrder: step.ExecutionOrder,
		})
		if err != nil {
			return model.TestSheet{}, err
		}
	}
	if err := s.markPlanChanged(sheet.PlanID); err != nil {
		return model.TestSheet{}, err
	}
	return s.repo.GetSheet(copySheet.ID)
}

func (s *Service) CreateStep(sheetID int64, input model.StepInput) (model.TestSheetStep, error) {
	if strings.TrimSpace(input.Action) == "" && strings.TrimSpace(input.ExpectedResult) == "" {
		return model.TestSheetStep{}, fmt.Errorf("step action or expected result is required")
	}
	sheet, err := s.repo.GetSheet(sheetID)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	step, err := s.repo.CreateStep(sheetID, input)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	if err := s.markPlanChanged(sheet.PlanID); err != nil {
		return model.TestSheetStep{}, err
	}
	return step, nil
}

func (s *Service) ListSteps(sheetID int64) ([]model.TestSheetStep, error) {
	if _, err := s.repo.GetSheet(sheetID); err != nil {
		return nil, err
	}
	return s.repo.ListSteps(sheetID)
}

func (s *Service) UpdateStep(id int64, input model.StepInput) (model.TestSheetStep, error) {
	if strings.TrimSpace(input.Action) == "" && strings.TrimSpace(input.ExpectedResult) == "" {
		return model.TestSheetStep{}, fmt.Errorf("step action or expected result is required")
	}
	step, err := s.repo.GetStep(id)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	sheet, err := s.repo.GetSheet(step.SheetID)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	updated, err := s.repo.UpdateStep(id, input)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	if err := s.markPlanChanged(sheet.PlanID); err != nil {
		return model.TestSheetStep{}, err
	}
	return updated, nil
}

func (s *Service) DeleteStep(id int64) error {
	step, err := s.repo.GetStep(id)
	if err != nil {
		return err
	}
	sheet, err := s.repo.GetSheet(step.SheetID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteStep(id); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) DuplicateStep(id int64) (model.TestSheetStep, error) {
	step, err := s.repo.GetStep(id)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	sheet, err := s.repo.GetSheet(step.SheetID)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	duplicated, err := s.repo.DuplicateStep(id)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	if err := s.markPlanChanged(sheet.PlanID); err != nil {
		return model.TestSheetStep{}, err
	}
	return duplicated, nil
}

func (s *Service) ReorderSteps(sheetID int64, stepIDs []int64) error {
	sheet, err := s.repo.GetSheet(sheetID)
	if err != nil {
		return err
	}
	if err := s.repo.ReorderSteps(sheetID, stepIDs); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) ReorderSheets(planID int64, sheetIDs []int64) error {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return err
	}
	if err := s.repo.ReorderSheets(planID, sheetIDs); err != nil {
		return err
	}
	return s.markPlanChanged(planID)
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

func (s *Service) ListPlanRuns(planID int64) ([]model.TestRunSummary, error) {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return nil, err
	}
	return s.repo.ListPlanRuns(planID)
}

func (s *Service) ListRunSummaries() ([]model.TestRunSummary, error) {
	return s.repo.ListRunSummaries()
}

func (s *Service) ListPlanSummaries(includeDeleted bool) ([]model.TestPlanSummary, error) {
	return s.repo.ListPlanSummaries(includeDeleted)
}

func (s *Service) ReplayRun(runID int64) (model.TestRun, error) {
	return s.repo.ReplayRun(runID)
}

func (s *Service) ArchiveRun(runID int64) (model.TestRun, error) {
	return s.repo.ArchiveRun(runID)
}

func (s *Service) CancelRun(runID int64) (model.TestRun, error) {
	run, err := s.repo.GetRun(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	if run.Status != model.TestRunStatusRunning {
		return model.TestRun{}, fmt.Errorf("Cette execution est terminee et ne peut plus etre modifiee.")
	}
	return s.repo.CancelRun(runID)
}

func (s *Service) UpdateRunSheet(runID, runSheetID int64, input model.RunSheetResultInput) (model.RunSheet, error) {
	if err := s.ensureRunEditable(runID); err != nil {
		return model.RunSheet{}, err
	}
	if !isAllowedStatus(input.Status) {
		return model.RunSheet{}, fmt.Errorf("invalid run sheet status")
	}
	return s.repo.UpdateRunSheet(runID, runSheetID, input)
}

func (s *Service) UpdateRunStep(runID, runStepID int64, input model.RunStepResultInput) (model.RunStep, error) {
	if err := s.ensureRunEditable(runID); err != nil {
		return model.RunStep{}, err
	}
	if !isAllowedStatus(input.Status) {
		return model.RunStep{}, fmt.Errorf("invalid run step status")
	}
	return s.repo.UpdateRunStep(runID, runStepID, input)
}

func (s *Service) FinishRun(runID int64) (model.TestRun, error) {
	if err := s.ensureRunEditable(runID); err != nil {
		return model.TestRun{}, err
	}
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
		for _, step := range sheet.Steps {
			counts[step.Status]++
		}
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
		writeReportLine(&builder, "Configuration", sheet.Config)
		writeReportLine(&builder, "Commande", sheet.Command)
		writeReportLine(&builder, "Notes", sheet.Notes)
		if len(sheet.Steps) > 0 {
			builder.WriteString("\n| # | Champ | Action | Resultat attendu | Statut | Resultat obtenu | Commentaire |\n")
			builder.WriteString("|---|---|---|---|---|---|---|\n")
			for _, step := range sheet.Steps {
				fmt.Fprintf(&builder, "| %d | %s | %s | %s | %s | %s | %s |\n",
					step.ExecutionOrder,
					tableCell(step.Field),
					tableCell(step.Action),
					tableCell(step.ExpectedResult),
					tableCell(step.Status),
					tableCell(step.ActualResult),
					tableCell(step.Comment),
				)
			}
		}
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func writeReportLine(builder *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(builder, "#### %s\n\n%s\n\n", label, value)
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r\n", "<br>")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

func isAllowedStatus(status string) bool {
	switch status {
	case model.RunSheetStatusPending, model.RunSheetStatusPassed, model.RunSheetStatusFailed, model.RunSheetStatusBlocked, model.RunSheetStatusSkipped:
		return true
	default:
		return false
	}
}

func (s *Service) ensureRunEditable(runID int64) error {
	run, err := s.repo.GetRun(runID)
	if err != nil {
		return err
	}
	if run.Status == model.TestRunStatusRunning {
		return nil
	}
	return fmt.Errorf("Cette execution est terminee et ne peut plus etre modifiee.")
}

func (s *Service) markPlanChanged(planID int64) error {
	if err := s.repo.TouchPlan(planID); err != nil {
		return err
	}
	return s.cancelRunningRunsForPlan(planID)
}

func (s *Service) cancelRunningRunsForPlan(planID int64) error {
	runs, err := s.repo.ListPlanRuns(planID)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.Status == model.TestRunStatusRunning {
			if _, err := s.repo.CancelRun(run.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func IsNotFound(err error) bool {
	return repository.IsNotFound(err)
}
