package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"toolBox/modules/test-sheet/pkg/model"
	"toolBox/modules/test-sheet/pkg/repository"
)

const maxDocumentUploadBytes = 50 << 20

var unsafeFilenameCharacters = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var ErrRunNotEditable = errors.New("Cette execution est terminee et ne peut plus etre modifiee.")

type NameConflictError struct {
	Message      string
	ConflictType string
	Hidden       bool
}

func (err NameConflictError) Error() string {
	return err.Message
}

type Repository interface {
	CreatePlan(model.PlanInput) (model.TestPlan, error)
	ListPlans() ([]model.TestPlan, error)
	GetPlan(int64) (model.TestPlan, error)
	TouchPlan(int64) error
	UpdatePlan(int64, model.PlanInput) (model.TestPlan, error)
	DeletePlan(int64) error
	PermanentDeletePlan(int64) error
	RestorePlan(int64) (model.TestPlan, error)
	CreateGroup(int64, model.GroupInput) (model.TestGroup, error)
	ListGroups(int64) ([]model.TestGroup, error)
	GetGroup(int64) (model.TestGroup, error)
	UpdateGroup(int64, model.GroupInput) (model.TestGroup, error)
	TouchGroup(int64) error
	DeleteGroup(int64) error
	ReorderGroups(int64, []int64) error
	DefaultGroupID(int64) (int64, error)
	CreateSheet(int64, model.SheetInput) (model.TestSheet, error)
	CreateSheetInGroup(int64, model.SheetInput) (model.TestSheet, error)
	ListSheets(int64) ([]model.TestSheet, error)
	ListSheetsByGroup(int64) ([]model.TestSheet, error)
	GetSheet(int64) (model.TestSheet, error)
	UpdateSheet(int64, model.SheetInput) (model.TestSheet, error)
	DeleteSheet(int64) error
	ReindexGroupSheets(int64) error
	CreateStep(int64, model.StepInput) (model.TestSheetStep, error)
	ListSteps(int64) ([]model.TestSheetStep, error)
	GetStep(int64) (model.TestSheetStep, error)
	UpdateStep(int64, model.StepInput) (model.TestSheetStep, error)
	DeleteStep(int64) error
	DuplicateStep(int64) (model.TestSheetStep, error)
	ReindexSheetSteps(int64) error
	ReorderSteps(int64, []int64) error
	ReorderSheets(int64, []int64) error
	ReorderGroupSheets(int64, []int64) error
	ListDocuments(int64) ([]model.TestDocument, error)
	GetDocument(int64) (model.TestDocument, error)
	CreateDocument(model.TestDocument) (model.TestDocument, error)
	UpdateDocumentFile(int64, string, string, string, int64, string) (model.TestDocument, error)
	DeleteDocument(int64) (model.TestDocument, error)
	LinkSheetDocument(int64, int64) error
	UnlinkSheetDocument(int64, int64) error
	LinkStepDocument(int64, int64) error
	UnlinkStepDocument(int64, int64) error
	CreateRunWithSnapshot(int64) (model.TestRun, error)
	CreateRunWithGroupSnapshot(int64) (model.TestRun, error)
	CreateImportedRun(model.TestRun) (model.TestRun, error)
	CreateImportedRunGroup(model.RunGroup) (model.RunGroup, error)
	CreateImportedRunSheet(model.RunSheet) (model.RunSheet, error)
	CreateImportedRunStep(model.RunStep) (model.RunStep, error)
	GetRun(int64) (model.TestRun, error)
	ListPlanRuns(int64) ([]model.TestRunSummary, error)
	ListGroupRuns(int64) ([]model.TestRunSummary, error)
	ListRunSummaries() ([]model.TestRunSummary, error)
	ListPlanSummaries(bool) ([]model.TestPlanSummary, error)
	ReplayRun(int64) (model.TestRun, error)
	ArchiveRun(int64) (model.TestRun, error)
	CancelRun(int64) (model.TestRun, error)
	UpdateRunSheet(int64, int64, model.RunSheetResultInput) (model.RunSheet, error)
	UpdateRunStep(int64, int64, model.RunStepResultInput) (model.RunStep, error)
	FinishRun(int64) (model.TestRun, error)
	GetRunIDForRunSheet(int64) (int64, error)
	GetRunIDForRunStep(int64) (int64, error)
	ListRunSheetEvidences(int64) ([]model.Evidence, error)
	GetEvidence(int64) (model.Evidence, error)
	CreateEvidence(model.Evidence) (model.Evidence, error)
	UpdateEvidenceFile(int64, string, string, int64) (model.Evidence, error)
	DeleteEvidence(int64) (model.Evidence, error)
	ListRunStepEvidences(int64) ([]model.Evidence, error)
	GetStepEvidence(int64) (model.Evidence, error)
	CreateStepEvidence(model.Evidence) (model.Evidence, error)
	UpdateStepEvidenceFile(int64, string, string, int64) (model.Evidence, error)
	DeleteStepEvidence(int64) (model.Evidence, error)
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
	if err := s.ensurePlanNameUnique(input.Name, 0); err != nil {
		return model.TestPlan{}, err
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
	if _, err := s.repo.GetPlan(id); err != nil {
		return model.TestPlan{}, err
	}
	if err := s.ensurePlanNameUnique(input.Name, id); err != nil {
		return model.TestPlan{}, err
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
	documents, err := s.repo.ListDocuments(id)
	if err != nil {
		return err
	}
	if err := s.repo.PermanentDeletePlan(id); err != nil {
		return err
	}
	for _, document := range documents {
		if document.StoragePath != "" {
			_ = os.Remove(document.StoragePath)
		}
	}
	return nil
}

func (s *Service) RestorePlan(id int64) (model.TestPlan, error) {
	plan, err := s.repo.GetPlan(id)
	if err != nil {
		return model.TestPlan{}, err
	}
	if err := s.ensurePlanNameCanBeRestored(plan.Name, id); err != nil {
		return model.TestPlan{}, err
	}
	return s.repo.RestorePlan(id)
}

func (s *Service) CreateGroup(planID int64, input model.GroupInput) (model.TestGroup, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestGroup{}, fmt.Errorf("group name is required")
	}
	if _, err := s.repo.GetPlan(planID); err != nil {
		return model.TestGroup{}, err
	}
	if err := s.ensureGroupNameUnique(planID, input.Name, 0); err != nil {
		return model.TestGroup{}, err
	}
	group, err := s.repo.CreateGroup(planID, input)
	if err != nil {
		return model.TestGroup{}, err
	}
	if err := s.markPlanChanged(planID); err != nil {
		return model.TestGroup{}, err
	}
	return group, nil
}

func (s *Service) ListGroups(planID int64) ([]model.TestGroup, error) {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return nil, err
	}
	return s.repo.ListGroups(planID)
}

func (s *Service) GetGroup(groupID int64) (model.TestGroup, error) {
	return s.repo.GetGroup(groupID)
}

func (s *Service) UpdateGroup(groupID int64, input model.GroupInput) (model.TestGroup, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestGroup{}, fmt.Errorf("group name is required")
	}
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return model.TestGroup{}, err
	}
	if err := s.ensureGroupNameUnique(group.PlanID, input.Name, groupID); err != nil {
		return model.TestGroup{}, err
	}
	updated, err := s.repo.UpdateGroup(groupID, input)
	if err != nil {
		return model.TestGroup{}, err
	}
	if err := s.markGroupChanged(group.ID); err != nil {
		return model.TestGroup{}, err
	}
	return updated, nil
}

func (s *Service) DeleteGroup(groupID int64) error {
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return err
	}
	groups, err := s.repo.ListGroups(group.PlanID)
	if err != nil {
		return err
	}
	if len(groups) <= 1 {
		return fmt.Errorf("cannot delete the last sub-plan")
	}
	if err := s.repo.DeleteGroup(groupID); err != nil {
		return err
	}
	return s.markGroupChanged(group.ID)
}

func (s *Service) ReorderGroups(planID int64, groupIDs []int64) error {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return err
	}
	if err := s.repo.ReorderGroups(planID, groupIDs); err != nil {
		return err
	}
	return s.markPlanChanged(planID)
}

func (s *Service) DuplicateGroup(groupID int64, input model.DuplicateGroupInput) (model.TestGroup, error) {
	source, err := s.repo.GetGroup(groupID)
	if err != nil {
		return model.TestGroup{}, err
	}
	targetPlanID := input.TargetPlanID
	if targetPlanID == 0 {
		targetPlanID = source.PlanID
	}
	if _, err := s.repo.GetPlan(targetPlanID); err != nil {
		return model.TestGroup{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name, err = nextCopyName(source.Name, func(candidate string) (bool, error) {
			return s.groupNameExists(targetPlanID, candidate, 0)
		})
		if err != nil {
			return model.TestGroup{}, err
		}
	} else if err := s.ensureGroupNameUnique(targetPlanID, name, 0); err != nil {
		return model.TestGroup{}, err
	}
	copyGroup, err := s.repo.CreateGroup(targetPlanID, model.GroupInput{
		Name:        name,
		Description: source.Description,
	})
	if err != nil {
		return model.TestGroup{}, err
	}
	documentMap := map[int64]int64{}
	for _, sheet := range source.Sheets {
		copySheet, err := s.repo.CreateSheetInGroup(copyGroup.ID, model.SheetInput{
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
			return model.TestGroup{}, err
		}
		for _, document := range sheet.Documents {
			targetDocumentID, err := s.copyOrReuseDocument(document, targetPlanID, documentMap)
			if err != nil {
				return model.TestGroup{}, err
			}
			if err := s.repo.LinkSheetDocument(copySheet.ID, targetDocumentID); err != nil {
				return model.TestGroup{}, err
			}
		}
		for _, step := range sheet.Steps {
			copyStep, err := s.repo.CreateStep(copySheet.ID, model.StepInput{
				Action:         step.Action,
				Field:          step.Field,
				ExpectedResult: step.ExpectedResult,
				ExecutionOrder: step.ExecutionOrder,
			})
			if err != nil {
				return model.TestGroup{}, err
			}
			for _, document := range step.Documents {
				targetDocumentID, err := s.copyOrReuseDocument(document, targetPlanID, documentMap)
				if err != nil {
					return model.TestGroup{}, err
				}
				if err := s.repo.LinkStepDocument(copyStep.ID, targetDocumentID); err != nil {
					return model.TestGroup{}, err
				}
			}
		}
	}
	if err := s.markPlanChanged(targetPlanID); err != nil {
		return model.TestGroup{}, err
	}
	return s.repo.GetGroup(copyGroup.ID)
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
	name, err := nextCopyName(plan.Name, func(candidate string) (bool, error) {
		return s.planNameExists(candidate, 0)
	})
	if err != nil {
		return model.TestPlan{}, err
	}
	copyPlan, err := s.repo.CreatePlan(model.PlanInput{
		Name:           name,
		Description:    plan.Description,
		MockupSettings: plan.MockupSettings,
	})
	if err != nil {
		return model.TestPlan{}, err
	}
	targetGroupID, err := s.repo.DefaultGroupID(copyPlan.ID)
	if err != nil {
		return model.TestPlan{}, err
	}
	for _, sheet := range sheets {
		sheetName, err := nextCopyName(sheet.Name, func(candidate string) (bool, error) {
			return s.sheetNameExists(targetGroupID, candidate, 0)
		})
		if err != nil {
			return model.TestPlan{}, err
		}
		if exists, err := s.sheetNameExists(targetGroupID, sheet.Name, 0); err != nil {
			return model.TestPlan{}, err
		} else if !exists {
			sheetName = sheet.Name
		}
		copySheet, err := s.repo.CreateSheet(copyPlan.ID, model.SheetInput{
			Name:           sheetName,
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
	groupID, err := s.repo.DefaultGroupID(planID)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.ensureSheetNameUnique(groupID, input.Name, 0); err != nil {
		return model.TestSheet{}, err
	}
	sheet, err := s.repo.CreateSheetInGroup(groupID, input)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markPlanChanged(planID); err != nil {
		return model.TestSheet{}, err
	}
	return sheet, nil
}

func (s *Service) CreateSheetInGroup(groupID int64, input model.SheetInput) (model.TestSheet, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return model.TestSheet{}, fmt.Errorf("sheet name is required")
	}
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.ensureSheetNameUnique(groupID, input.Name, 0); err != nil {
		return model.TestSheet{}, err
	}
	sheet, err := s.repo.CreateSheetInGroup(groupID, input)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markGroupChanged(group.ID); err != nil {
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

func (s *Service) ListSheetsByGroup(groupID int64) ([]model.TestSheet, error) {
	if _, err := s.repo.GetGroup(groupID); err != nil {
		return nil, err
	}
	return s.repo.ListSheetsByGroup(groupID)
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
	if err := s.ensureSheetNameUnique(sheet.GroupID, input.Name, id); err != nil {
		return model.TestSheet{}, err
	}
	updated, err := s.repo.UpdateSheet(id, input)
	if err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markGroupChanged(sheet.GroupID); err != nil {
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
	return s.markGroupChanged(sheet.GroupID)
}

func (s *Service) DuplicateSheet(id int64) (model.TestSheet, error) {
	sheet, err := s.repo.GetSheet(id)
	if err != nil {
		return model.TestSheet{}, err
	}
	name, err := nextCopyName(sheet.Name, func(candidate string) (bool, error) {
		return s.sheetNameExists(sheet.GroupID, candidate, 0)
	})
	if err != nil {
		return model.TestSheet{}, err
	}
	copySheet, err := s.repo.CreateSheetInGroup(sheet.GroupID, model.SheetInput{
		Name:           name,
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
	if err := s.repo.ReindexGroupSheets(sheet.GroupID); err != nil {
		return model.TestSheet{}, err
	}
	if err := s.markGroupChanged(sheet.GroupID); err != nil {
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
	if err := s.markGroupChanged(sheet.GroupID); err != nil {
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
	if err := s.markGroupChanged(sheet.GroupID); err != nil {
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
	return s.markGroupChanged(sheet.GroupID)
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
	if err := s.repo.ReindexSheetSteps(step.SheetID); err != nil {
		return model.TestSheetStep{}, err
	}
	if err := s.markGroupChanged(sheet.GroupID); err != nil {
		return model.TestSheetStep{}, err
	}
	return s.repo.GetStep(duplicated.ID)
}

func (s *Service) ReorderSteps(sheetID int64, stepIDs []int64) error {
	sheet, err := s.repo.GetSheet(sheetID)
	if err != nil {
		return err
	}
	if err := s.repo.ReorderSteps(sheetID, stepIDs); err != nil {
		return err
	}
	return s.markGroupChanged(sheet.GroupID)
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

func (s *Service) ReorderGroupSheets(groupID int64, sheetIDs []int64) error {
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return err
	}
	if err := s.repo.ReorderGroupSheets(groupID, sheetIDs); err != nil {
		return err
	}
	return s.markGroupChanged(group.ID)
}

func (s *Service) ListDocuments(planID int64) ([]model.TestDocument, error) {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return nil, err
	}
	return s.repo.ListDocuments(planID)
}

func (s *Service) GetDocument(documentID int64) (model.TestDocument, error) {
	return s.repo.GetDocument(documentID)
}

func (s *Service) UploadDocument(planID int64, header *multipart.FileHeader, description string) (model.TestDocument, error) {
	if _, err := s.repo.GetPlan(planID); err != nil {
		return model.TestDocument{}, err
	}
	if header == nil {
		return model.TestDocument{}, fmt.Errorf("document file is required")
	}
	if header.Size > maxDocumentUploadBytes {
		return model.TestDocument{}, fmt.Errorf("document is too large")
	}
	source, err := header.Open()
	if err != nil {
		return model.TestDocument{}, err
	}
	defer source.Close()

	originalName := filepath.Base(header.Filename)
	safeName := safeFilename(originalName)
	document, err := s.repo.CreateDocument(model.TestDocument{
		PlanID:       planID,
		OriginalName: originalName,
		Description:  strings.TrimSpace(description),
	})
	if err != nil {
		return model.TestDocument{}, err
	}

	planDirectory := filepath.Join("data", "test-sheet", "documents", fmt.Sprintf("plan-%d", planID))
	if err := os.MkdirAll(planDirectory, 0755); err != nil {
		_, _ = s.repo.DeleteDocument(document.ID)
		return model.TestDocument{}, err
	}
	storedName := fmt.Sprintf("doc-%d-%s", document.ID, safeName)
	storagePath := filepath.Join(planDirectory, storedName)
	destination, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		_, _ = s.repo.DeleteDocument(document.ID)
		return model.TestDocument{}, err
	}
	defer destination.Close()

	hash := sha256.New()
	limited := io.LimitReader(source, maxDocumentUploadBytes+1)
	written, err := io.Copy(io.MultiWriter(destination, hash), limited)
	if err != nil {
		_, _ = s.repo.DeleteDocument(document.ID)
		return model.TestDocument{}, err
	}
	if written > maxDocumentUploadBytes {
		_ = os.Remove(storagePath)
		_, _ = s.repo.DeleteDocument(document.ID)
		return model.TestDocument{}, fmt.Errorf("document is too large")
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectContentType(storagePath)
	}
	document, err = s.repo.UpdateDocumentFile(document.ID, storedName, storagePath, mimeType, written, hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		_ = os.Remove(storagePath)
		return model.TestDocument{}, err
	}
	if err := s.markPlanChanged(planID); err != nil {
		return model.TestDocument{}, err
	}
	return document, nil
}

func (s *Service) DeleteDocument(documentID int64) error {
	document, err := s.repo.DeleteDocument(documentID)
	if err != nil {
		return err
	}
	if document.StoragePath != "" {
		_ = os.Remove(document.StoragePath)
	}
	return s.markPlanChanged(document.PlanID)
}

func (s *Service) copyOrReuseDocument(source model.TestDocument, targetPlanID int64, documentMap map[int64]int64) (int64, error) {
	if source.PlanID == targetPlanID {
		return source.ID, nil
	}
	if mapped, ok := documentMap[source.ID]; ok {
		return mapped, nil
	}
	targetDocuments, err := s.repo.ListDocuments(targetPlanID)
	if err != nil {
		return 0, err
	}
	for _, document := range targetDocuments {
		if document.OriginalName == source.OriginalName && document.SHA256 != "" && document.SHA256 == source.SHA256 {
			documentMap[source.ID] = document.ID
			return document.ID, nil
		}
	}
	created, err := s.repo.CreateDocument(model.TestDocument{
		PlanID:       targetPlanID,
		OriginalName: source.OriginalName,
		Description:  source.Description,
	})
	if err != nil {
		return 0, err
	}
	planDirectory := filepath.Join("data", "test-sheet", "documents", fmt.Sprintf("plan-%d", targetPlanID))
	if err := os.MkdirAll(planDirectory, 0755); err != nil {
		_, _ = s.repo.DeleteDocument(created.ID)
		return 0, err
	}
	storedName := fmt.Sprintf("doc-%d-%s", created.ID, safeFilename(source.OriginalName))
	storagePath := filepath.Join(planDirectory, storedName)
	if err := copyFile(source.StoragePath, storagePath); err != nil {
		_, _ = s.repo.DeleteDocument(created.ID)
		return 0, err
	}
	updated, err := s.repo.UpdateDocumentFile(created.ID, storedName, storagePath, source.MimeType, source.SizeBytes, source.SHA256)
	if err != nil {
		_ = os.Remove(storagePath)
		return 0, err
	}
	documentMap[source.ID] = updated.ID
	return updated.ID, nil
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	return err
}

func (s *Service) LinkSheetDocument(sheetID, documentID int64) error {
	sheet, document, err := s.sheetAndDocument(sheetID, documentID)
	if err != nil {
		return err
	}
	if sheet.PlanID != document.PlanID {
		return fmt.Errorf("document does not belong to this plan")
	}
	if err := s.repo.LinkSheetDocument(sheetID, documentID); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) UnlinkSheetDocument(sheetID, documentID int64) error {
	sheet, _, err := s.sheetAndDocument(sheetID, documentID)
	if err != nil {
		return err
	}
	if err := s.repo.UnlinkSheetDocument(sheetID, documentID); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) LinkStepDocument(stepID, documentID int64) error {
	step, sheet, document, err := s.stepSheetAndDocument(stepID, documentID)
	if err != nil {
		return err
	}
	if sheet.PlanID != document.PlanID {
		return fmt.Errorf("document does not belong to this plan")
	}
	if err := s.repo.LinkStepDocument(step.ID, documentID); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
}

func (s *Service) UnlinkStepDocument(stepID, documentID int64) error {
	step, sheet, _, err := s.stepSheetAndDocument(stepID, documentID)
	if err != nil {
		return err
	}
	if err := s.repo.UnlinkStepDocument(step.ID, documentID); err != nil {
		return err
	}
	return s.markPlanChanged(sheet.PlanID)
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

func (s *Service) CreateGroupRun(groupID int64) (model.TestRun, error) {
	sheets, err := s.repo.ListSheetsByGroup(groupID)
	if err != nil {
		return model.TestRun{}, err
	}
	if len(sheets) == 0 {
		return model.TestRun{}, fmt.Errorf("cannot start a run without sheets")
	}
	return s.repo.CreateRunWithGroupSnapshot(groupID)
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

func (s *Service) ListGroupRuns(groupID int64) ([]model.TestRunSummary, error) {
	if _, err := s.repo.GetGroup(groupID); err != nil {
		return nil, err
	}
	return s.repo.ListGroupRuns(groupID)
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
		return model.TestRun{}, ErrRunNotEditable
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

func (s *Service) ListRunSheetEvidences(runID, runSheetID int64) ([]model.Evidence, error) {
	if err := s.ensureRunSheetBelongsToRun(runID, runSheetID); err != nil {
		return nil, err
	}
	return s.repo.ListRunSheetEvidences(runSheetID)
}

func (s *Service) GetEvidence(evidenceID int64) (model.Evidence, error) {
	return s.repo.GetEvidence(evidenceID)
}

func (s *Service) UploadRunSheetEvidence(runID, runSheetID int64, header *multipart.FileHeader, comment string) (model.Evidence, error) {
	if err := s.ensureRunEditable(runID); err != nil {
		return model.Evidence{}, err
	}
	if err := s.ensureRunSheetBelongsToRun(runID, runSheetID); err != nil {
		return model.Evidence{}, err
	}
	if header == nil {
		return model.Evidence{}, fmt.Errorf("evidence file is required")
	}
	if header.Size > maxDocumentUploadBytes {
		return model.Evidence{}, fmt.Errorf("evidence is too large")
	}
	source, err := header.Open()
	if err != nil {
		return model.Evidence{}, err
	}
	defer source.Close()

	originalName := filepath.Base(header.Filename)
	safeName := safeFilename(originalName)
	evidence, err := s.repo.CreateEvidence(model.Evidence{
		RunSheetID: runSheetID,
		Name:       originalName,
		Comment:    strings.TrimSpace(comment),
	})
	if err != nil {
		return model.Evidence{}, err
	}

	evidenceDirectory := filepath.Join("data", "test-sheet", "runs", fmt.Sprintf("run-%d", runID), "evidences", fmt.Sprintf("sheet-%d", runSheetID))
	if err := os.MkdirAll(evidenceDirectory, 0755); err != nil {
		_, _ = s.repo.DeleteEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	storedName := fmt.Sprintf("evidence-%d-%s", evidence.ID, safeName)
	storagePath := filepath.Join(evidenceDirectory, storedName)
	destination, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		_, _ = s.repo.DeleteEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	defer destination.Close()

	limited := io.LimitReader(source, maxDocumentUploadBytes+1)
	written, err := io.Copy(destination, limited)
	if err != nil {
		_, _ = s.repo.DeleteEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	if written > maxDocumentUploadBytes {
		_ = os.Remove(storagePath)
		_, _ = s.repo.DeleteEvidence(evidence.ID)
		return model.Evidence{}, fmt.Errorf("evidence is too large")
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectContentType(storagePath)
	}
	evidence, err = s.repo.UpdateEvidenceFile(evidence.ID, storagePath, mimeType, written)
	if err != nil {
		_ = os.Remove(storagePath)
		return model.Evidence{}, err
	}
	return evidence, nil
}

func (s *Service) DeleteEvidence(evidenceID int64) error {
	evidence, err := s.repo.GetEvidence(evidenceID)
	if err != nil {
		return err
	}
	runID, err := s.repo.GetRunIDForRunSheet(evidence.RunSheetID)
	if err != nil {
		return err
	}
	if err := s.ensureRunEditable(runID); err != nil {
		return err
	}
	deleted, err := s.repo.DeleteEvidence(evidenceID)
	if err != nil {
		return err
	}
	if deleted.Path != "" {
		_ = os.Remove(deleted.Path)
	}
	return nil
}

func (s *Service) ListRunStepEvidences(runID, runStepID int64) ([]model.Evidence, error) {
	if err := s.ensureRunStepBelongsToRun(runID, runStepID); err != nil {
		return nil, err
	}
	return s.repo.ListRunStepEvidences(runStepID)
}

func (s *Service) GetStepEvidence(evidenceID int64) (model.Evidence, error) {
	return s.repo.GetStepEvidence(evidenceID)
}

func (s *Service) UploadRunStepEvidence(runID, runStepID int64, header *multipart.FileHeader) (model.Evidence, error) {
	if err := s.ensureRunEditable(runID); err != nil {
		return model.Evidence{}, err
	}
	if err := s.ensureRunStepBelongsToRun(runID, runStepID); err != nil {
		return model.Evidence{}, err
	}
	if header == nil {
		return model.Evidence{}, fmt.Errorf("evidence file is required")
	}
	if header.Size > maxDocumentUploadBytes {
		return model.Evidence{}, fmt.Errorf("evidence is too large")
	}
	source, err := header.Open()
	if err != nil {
		return model.Evidence{}, err
	}
	defer source.Close()

	originalName := filepath.Base(header.Filename)
	safeName := safeFilename(originalName)
	evidence, err := s.repo.CreateStepEvidence(model.Evidence{
		RunStepID: runStepID,
		Name:      originalName,
	})
	if err != nil {
		return model.Evidence{}, err
	}

	evidenceDirectory := filepath.Join("data", "test-sheet", "runs", fmt.Sprintf("run-%d", runID), "evidences", fmt.Sprintf("step-%d", runStepID))
	if err := os.MkdirAll(evidenceDirectory, 0755); err != nil {
		_, _ = s.repo.DeleteStepEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	storedName := fmt.Sprintf("evidence-%d-%s", evidence.ID, safeName)
	storagePath := filepath.Join(evidenceDirectory, storedName)
	destination, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		_, _ = s.repo.DeleteStepEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	defer destination.Close()

	limited := io.LimitReader(source, maxDocumentUploadBytes+1)
	written, err := io.Copy(destination, limited)
	if err != nil {
		_, _ = s.repo.DeleteStepEvidence(evidence.ID)
		return model.Evidence{}, err
	}
	if written > maxDocumentUploadBytes {
		_ = os.Remove(storagePath)
		_, _ = s.repo.DeleteStepEvidence(evidence.ID)
		return model.Evidence{}, fmt.Errorf("evidence is too large")
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectContentType(storagePath)
	}
	evidence, err = s.repo.UpdateStepEvidenceFile(evidence.ID, storagePath, mimeType, written)
	if err != nil {
		_ = os.Remove(storagePath)
		return model.Evidence{}, err
	}
	return evidence, nil
}

func (s *Service) DeleteStepEvidence(evidenceID int64) error {
	evidence, err := s.repo.GetStepEvidence(evidenceID)
	if err != nil {
		return err
	}
	runID, err := s.repo.GetRunIDForRunStep(evidence.RunStepID)
	if err != nil {
		return err
	}
	if err := s.ensureRunEditable(runID); err != nil {
		return err
	}
	deleted, err := s.repo.DeleteStepEvidence(evidenceID)
	if err != nil {
		return err
	}
	if deleted.Path != "" {
		_ = os.Remove(deleted.Path)
	}
	return nil
}

func (s *Service) GenerateMarkdownReport(runID int64) (string, error) {
	run, err := s.repo.GetRun(runID)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Rapport de test - %s\n\n", run.PlanName)
	fmt.Fprintf(&builder, "- Execution: #%d\n", run.RunNumber)
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
		fmt.Fprintf(&builder, "- Statut: %s\n", computedRunSheetStatus(sheet))
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
		writeReportLine(&builder, "Commentaire de la fiche", sheet.Comment)
		if len(sheet.Evidences) > 0 {
			builder.WriteString("#### Documents ajoutes\n\n")
			for _, evidence := range sheet.Evidences {
				fmt.Fprintf(&builder, "- %s\n", evidence.Name)
			}
			builder.WriteString("\n")
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

func computedRunSheetStatus(sheet model.RunSheet) string {
	if len(sheet.Steps) == 0 {
		return sheet.Status
	}
	hasBlocked := false
	hasPending := false
	nonSkippedSteps := 0
	for _, step := range sheet.Steps {
		switch step.Status {
		case model.RunSheetStatusFailed:
			return model.RunSheetStatusFailed
		case model.RunSheetStatusBlocked:
			hasBlocked = true
			nonSkippedSteps++
		case model.RunSheetStatusSkipped:
			continue
		case model.RunSheetStatusPending:
			hasPending = true
			nonSkippedSteps++
		case model.RunSheetStatusPassed:
			nonSkippedSteps++
		default:
			nonSkippedSteps++
		}
	}
	if hasBlocked {
		return model.RunSheetStatusBlocked
	}
	if hasPending {
		return model.RunSheetStatusPending
	}
	if nonSkippedSteps == 0 {
		return model.RunSheetStatusSkipped
	}
	return model.RunSheetStatusPassed
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
	return ErrRunNotEditable
}

func (s *Service) ensureRunSheetBelongsToRun(runID, runSheetID int64) error {
	actualRunID, err := s.repo.GetRunIDForRunSheet(runSheetID)
	if err != nil {
		return err
	}
	if actualRunID != runID {
		return fmt.Errorf("run sheet does not belong to this run")
	}
	return nil
}

func (s *Service) ensureRunStepBelongsToRun(runID, runStepID int64) error {
	actualRunID, err := s.repo.GetRunIDForRunStep(runStepID)
	if err != nil {
		return err
	}
	if actualRunID != runID {
		return fmt.Errorf("run step does not belong to this run")
	}
	return nil
}

func (s *Service) markPlanChanged(planID int64) error {
	if err := s.repo.TouchPlan(planID); err != nil {
		return err
	}
	return s.cancelRunningRunsForPlan(planID)
}

func (s *Service) markGroupChanged(groupID int64) error {
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return err
	}
	if err := s.repo.TouchPlan(group.PlanID); err != nil {
		return err
	}
	if err := s.repo.TouchGroup(groupID); err != nil {
		return err
	}
	return s.cancelRunningRunsForPlan(group.PlanID)
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

func (s *Service) cancelRunningRunsForGroup(groupID int64) error {
	runs, err := s.repo.ListGroupRuns(groupID)
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

func (s *Service) ensurePlanNameUnique(name string, excludeID int64) error {
	exists, hidden, err := s.planNameConflict(name, excludeID)
	if err != nil || !exists {
		return err
	}
	if hidden {
		return NameConflictError{
			Message:      "Un plan masqué utilise déjà ce nom. Restaurez-le ou choisissez un autre nom.",
			ConflictType: "plan",
			Hidden:       true,
		}
	}
	return NameConflictError{Message: "Un plan utilise déjà ce nom.", ConflictType: "plan"}
}

func (s *Service) ensurePlanNameCanBeRestored(name string, excludeID int64) error {
	summaries, err := s.repo.ListPlanSummaries(true)
	if err != nil {
		return err
	}
	normalized := normalizeNameForCompare(name)
	for _, plan := range summaries {
		if plan.ID == excludeID || normalizeNameForCompare(plan.Name) != normalized {
			continue
		}
		if plan.DeletedAt == nil {
			return NameConflictError{
				Message:      "Impossible de restaurer ce plan : un plan actif utilise déjà ce nom.",
				ConflictType: "plan",
			}
		}
		return NameConflictError{
			Message:      "Un plan masqué utilise déjà ce nom. Restaurez-le ou choisissez un autre nom.",
			ConflictType: "plan",
			Hidden:       true,
		}
	}
	return nil
}

func (s *Service) planNameExists(name string, excludeID int64) (bool, error) {
	exists, _, err := s.planNameConflict(name, excludeID)
	return exists, err
}

func (s *Service) planNameConflict(name string, excludeID int64) (bool, bool, error) {
	summaries, err := s.repo.ListPlanSummaries(true)
	if err != nil {
		return false, false, err
	}
	normalized := normalizeNameForCompare(name)
	for _, plan := range summaries {
		if plan.ID != excludeID && normalizeNameForCompare(plan.Name) == normalized {
			return true, plan.DeletedAt != nil, nil
		}
	}
	return false, false, nil
}

func (s *Service) ensureGroupNameUnique(planID int64, name string, excludeID int64) error {
	exists, err := s.groupNameExists(planID, name, excludeID)
	if err != nil || !exists {
		return err
	}
	return NameConflictError{
		Message:      "Un sous-plan utilise déjà ce nom dans ce plan.",
		ConflictType: "group",
	}
}

func (s *Service) groupNameExists(planID int64, name string, excludeID int64) (bool, error) {
	groups, err := s.repo.ListGroups(planID)
	if err != nil {
		return false, err
	}
	normalized := normalizeNameForCompare(name)
	for _, group := range groups {
		if group.ID != excludeID && normalizeNameForCompare(group.Name) == normalized {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) ensureSheetNameUnique(groupID int64, name string, excludeID int64) error {
	exists, err := s.sheetNameExists(groupID, name, excludeID)
	if err != nil || !exists {
		return err
	}
	return NameConflictError{
		Message:      "Une fiche utilise déjà ce nom dans ce sous-plan.",
		ConflictType: "sheet",
	}
}

func (s *Service) sheetNameExists(groupID int64, name string, excludeID int64) (bool, error) {
	sheets, err := s.repo.ListSheetsByGroup(groupID)
	if err != nil {
		return false, err
	}
	normalized := normalizeNameForCompare(name)
	for _, sheet := range sheets {
		if sheet.ID != excludeID && normalizeNameForCompare(sheet.Name) == normalized {
			return true, nil
		}
	}
	return false, nil
}

func normalizeNameForCompare(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Join(strings.Fields(value), " ")
	return strings.ToLower(value)
}

func nextCopyName(baseName string, exists func(name string) (bool, error)) (string, error) {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = "Copie"
	}
	for index := 1; ; index++ {
		name := fmt.Sprintf("%s (copie)", baseName)
		if index > 1 {
			name = fmt.Sprintf("%s (copie %d)", baseName, index)
		}
		found, err := exists(name)
		if err != nil {
			return "", err
		}
		if !found {
			return name, nil
		}
	}
}

func (s *Service) sheetAndDocument(sheetID, documentID int64) (model.TestSheet, model.TestDocument, error) {
	sheet, err := s.repo.GetSheet(sheetID)
	if err != nil {
		return model.TestSheet{}, model.TestDocument{}, err
	}
	document, err := s.repo.GetDocument(documentID)
	if err != nil {
		return model.TestSheet{}, model.TestDocument{}, err
	}
	return sheet, document, nil
}

func (s *Service) stepSheetAndDocument(stepID, documentID int64) (model.TestSheetStep, model.TestSheet, model.TestDocument, error) {
	step, err := s.repo.GetStep(stepID)
	if err != nil {
		return model.TestSheetStep{}, model.TestSheet{}, model.TestDocument{}, err
	}
	sheet, err := s.repo.GetSheet(step.SheetID)
	if err != nil {
		return model.TestSheetStep{}, model.TestSheet{}, model.TestDocument{}, err
	}
	document, err := s.repo.GetDocument(documentID)
	if err != nil {
		return model.TestSheetStep{}, model.TestSheet{}, model.TestDocument{}, err
	}
	return step, sheet, document, nil
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return "document"
	}
	name = unsafeFilenameCharacters.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		return "document"
	}
	if len(name) > 120 {
		extension := filepath.Ext(name)
		base := strings.TrimSuffix(name, extension)
		if len(extension) > 20 {
			extension = ""
		}
		limit := 120 - len(extension)
		if limit < 1 {
			limit = 1
		}
		if len(base) > limit {
			base = base[:limit]
		}
		name = base + extension
	}
	return name
}

func detectContentType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return ""
	}
	return http.DetectContentType(buffer[:n])
}

func IsNotFound(err error) bool {
	return repository.IsNotFound(err)
}

func IsConflict(err error) bool {
	var nameConflict NameConflictError
	return errors.Is(err, ErrRunNotEditable) || errors.As(err, &nameConflict)
}

func ConflictPayload(err error) (map[string]any, bool) {
	var nameConflict NameConflictError
	if !errors.As(err, &nameConflict) {
		return nil, false
	}
	payload := map[string]any{
		"error":        nameConflict.Message,
		"code":         "name_conflict",
		"conflictType": nameConflict.ConflictType,
	}
	if nameConflict.Hidden {
		payload["hidden"] = true
	}
	return payload, true
}
