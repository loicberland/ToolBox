package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"toolBox/modules/test-sheet/pkg/model"
)

const exportFormat = "toolbox-test-sheet-export"
const exportSchemaVersion = 1

type exportManifest struct {
	Format        string              `json:"format"`
	SchemaVersion int                 `json:"schemaVersion"`
	ExportedAt    time.Time           `json:"exportedAt"`
	Options       model.ExportOptions `json:"options"`
}

type exportData struct {
	Plan      model.TestPlan    `json:"plan"`
	Groups    []model.TestGroup `json:"groups,omitempty"`
	Sheets    []model.TestSheet `json:"sheets,omitempty"`
	Documents []exportDocument  `json:"documents,omitempty"`
	Runs      []model.TestRun   `json:"runs,omitempty"`
}

type exportDocument struct {
	model.TestDocument
	ExportPath string `json:"exportPath,omitempty"`
}

func DefaultExportOptions() model.ExportOptions {
	return model.ExportOptions{
		IncludeGroups:    true,
		IncludeSheets:    true,
		IncludeSteps:     true,
		IncludeDocuments: true,
		IncludeHistory:   false,
		IncludeEvidences: false,
	}
}

func NormalizeExportOptions(options model.ExportOptions) model.ExportOptions {
	if options.IncludeSteps {
		options.IncludeSheets = true
		options.IncludeGroups = true
	}
	if options.IncludeSheets {
		options.IncludeGroups = true
	}
	if options.IncludeEvidences {
		options.IncludeHistory = true
	}
	return options
}

func (s *Service) ExportPlan(planID int64, options model.ExportOptions) ([]byte, error) {
	options = NormalizeExportOptions(options)
	data, err := s.buildExportData(planID, options)
	if err != nil {
		return nil, err
	}
	manifest := exportManifest{
		Format:        exportFormat,
		SchemaVersion: exportSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Options:       options,
	}
	evidenceFiles := map[string]string{}
	if options.IncludeEvidences {
		evidenceFiles = setEvidenceExportPaths(data.Runs)
	}

	buffer := &bytes.Buffer{}
	archive := zip.NewWriter(buffer)
	if err := writeZipJSON(archive, "manifest.json", manifest); err != nil {
		_ = archive.Close()
		return nil, err
	}
	if err := writeZipJSON(archive, "data.json", data); err != nil {
		_ = archive.Close()
		return nil, err
	}
	if options.IncludeDocuments {
		for _, document := range data.Documents {
			if document.ExportPath == "" || document.StoragePath == "" {
				continue
			}
			if err := addFileToZip(archive, document.ExportPath, document.StoragePath); err != nil {
				_ = archive.Close()
				return nil, err
			}
		}
	}
	if options.IncludeEvidences {
		if err := addEvidenceFiles(archive, evidenceFiles); err != nil {
			_ = archive.Close()
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *Service) PreviewImportZip(payload []byte) (model.ImportPreview, error) {
	manifest, data, _, err := readExportZip(payload)
	if err != nil {
		return model.ImportPreview{}, err
	}
	return previewFromExport(manifest, data), nil
}

func (s *Service) ImportPlanZip(payload []byte) (model.ImportResult, error) {
	_, data, files, err := readExportZip(payload)
	if err != nil {
		return model.ImportResult{}, err
	}
	imported, err := s.importExportData(data, files)
	if err != nil {
		return model.ImportResult{}, err
	}
	return model.ImportResult{PlanID: imported.ID, Name: imported.Name}, nil
}

func (s *Service) buildExportData(planID int64, options model.ExportOptions) (exportData, error) {
	plan, err := s.repo.GetPlan(planID)
	if err != nil {
		return exportData{}, err
	}
	data := exportData{Plan: plan}
	if options.IncludeGroups {
		data.Groups, err = s.repo.ListGroups(planID)
		if err != nil {
			return exportData{}, err
		}
		for index := range data.Groups {
			data.Groups[index].Sheets = nil
			data.Groups[index].LatestRun = nil
		}
	}
	if options.IncludeSheets {
		data.Sheets, err = s.repo.ListSheets(planID)
		if err != nil {
			return exportData{}, err
		}
		if !options.IncludeSteps {
			for index := range data.Sheets {
				data.Sheets[index].Steps = nil
			}
		}
		if !options.IncludeDocuments {
			for index := range data.Sheets {
				data.Sheets[index].Documents = nil
				for stepIndex := range data.Sheets[index].Steps {
					data.Sheets[index].Steps[stepIndex].Documents = nil
				}
			}
		}
	}
	if options.IncludeDocuments {
		documents, err := s.repo.ListDocuments(planID)
		if err != nil {
			return exportData{}, err
		}
		for _, document := range documents {
			documentPath := path.Join("documents", fmt.Sprintf("doc-%d-%s", document.ID, safeFilename(document.OriginalName)))
			data.Documents = append(data.Documents, exportDocument{TestDocument: document, ExportPath: documentPath})
		}
	}
	if options.IncludeHistory {
		runs, err := s.repo.ListPlanRuns(planID)
		if err != nil {
			return exportData{}, err
		}
		for _, summary := range runs {
			run, err := s.repo.GetRun(summary.ID)
			if err != nil {
				return exportData{}, err
			}
			if !options.IncludeEvidences {
				clearRunEvidences(&run)
			}
			data.Runs = append(data.Runs, run)
		}
	}
	return data, nil
}

func (s *Service) importExportData(data exportData, files map[string]*zip.File) (model.TestPlan, error) {
	createdPlan, err := s.CreatePlan(model.PlanInput{
		Name:           data.Plan.Name,
		Description:    data.Plan.Description,
		MockupSettings: data.Plan.MockupSettings,
	})
	if err != nil {
		return model.TestPlan{}, err
	}
	planMap := map[int64]int64{data.Plan.ID: createdPlan.ID}
	groupMap := map[int64]int64{}
	sheetMap := map[int64]int64{}
	stepMap := map[int64]int64{}
	documentMap := map[int64]int64{}
	runMap := map[int64]int64{}
	runGroupMap := map[int64]int64{}
	runSheetMap := map[int64]int64{}
	runStepMap := map[int64]int64{}

	if len(data.Groups) > 0 {
		if groups, err := s.repo.ListGroups(createdPlan.ID); err == nil && len(groups) == 1 && len(data.Sheets) >= 0 {
			_ = s.repo.DeleteGroup(groups[0].ID)
		}
		for _, group := range data.Groups {
			created, err := s.repo.CreateGroup(createdPlan.ID, model.GroupInput{Name: group.Name, Description: group.Description, ExecutionOrder: group.ExecutionOrder})
			if err != nil {
				return model.TestPlan{}, err
			}
			groupMap[group.ID] = created.ID
		}
	}
	defaultGroupID := int64(0)
	if len(groupMap) == 0 {
		defaultGroupID, err = s.repo.DefaultGroupID(createdPlan.ID)
		if err != nil {
			return model.TestPlan{}, err
		}
	}
	for _, sheet := range data.Sheets {
		groupID := groupMap[sheet.GroupID]
		if groupID == 0 {
			groupID = defaultGroupID
		}
		created, err := s.repo.CreateSheetInGroup(groupID, model.SheetInput{
			Name: sheet.Name, Description: sheet.Description, Prerequisites: sheet.Prerequisites, Config: sheet.Config, Command: sheet.Command, Notes: sheet.Notes,
			Action: sheet.Action, ExpectedResult: sheet.ExpectedResult, ExecutionOrder: sheet.ExecutionOrder, MockupSettings: sheet.MockupSettings,
		})
		if err != nil {
			return model.TestPlan{}, err
		}
		sheetMap[sheet.ID] = created.ID
		for _, step := range sheet.Steps {
			createdStep, err := s.repo.CreateStep(created.ID, model.StepInput{Action: step.Action, Field: step.Field, ExpectedResult: step.ExpectedResult, ExecutionOrder: step.ExecutionOrder})
			if err != nil {
				return model.TestPlan{}, err
			}
			stepMap[step.ID] = createdStep.ID
		}
	}
	for _, document := range data.Documents {
		created, err := s.importDocument(createdPlan.ID, document, files)
		if err != nil {
			return model.TestPlan{}, err
		}
		documentMap[document.ID] = created.ID
	}
	for _, sheet := range data.Sheets {
		newSheetID := sheetMap[sheet.ID]
		for _, document := range sheet.Documents {
			if newDocumentID := documentMap[document.ID]; newSheetID != 0 && newDocumentID != 0 {
				if err := s.repo.LinkSheetDocument(newSheetID, newDocumentID); err != nil {
					return model.TestPlan{}, err
				}
			}
		}
		for _, step := range sheet.Steps {
			newStepID := stepMap[step.ID]
			for _, document := range step.Documents {
				if newDocumentID := documentMap[document.ID]; newStepID != 0 && newDocumentID != 0 {
					if err := s.repo.LinkStepDocument(newStepID, newDocumentID); err != nil {
						return model.TestPlan{}, err
					}
				}
			}
		}
	}
	for _, run := range data.Runs {
		run.PlanID = planMap[run.PlanID]
		run.GroupID = groupMap[run.GroupID]
		createdRun, err := s.repo.CreateImportedRun(run)
		if err != nil {
			return model.TestPlan{}, err
		}
		runMap[run.ID] = createdRun.ID
		for _, group := range run.Groups {
			oldGroupID := int64Value(group.SourceGroupID)
			group.RunID = createdRun.ID
			group.SourceGroupID = mappedPointer(groupMap, oldGroupID)
			createdGroup, err := s.repo.CreateImportedRunGroup(group)
			if err != nil {
				return model.TestPlan{}, err
			}
			runGroupMap[group.ID] = createdGroup.ID
			for _, sheet := range group.Sheets {
				oldSheetID := int64Value(sheet.SourceSheetID)
				sheet.RunID = createdRun.ID
				sheet.RunGroupID = createdGroup.ID
				sheet.SourceSheetID = mappedPointer(sheetMap, oldSheetID)
				createdSheet, err := s.repo.CreateImportedRunSheet(sheet)
				if err != nil {
					return model.TestPlan{}, err
				}
				runSheetMap[sheet.ID] = createdSheet.ID
				for _, step := range sheet.Steps {
					oldStepID := int64Value(step.SourceStepID)
					step.RunSheetID = createdSheet.ID
					step.SourceStepID = mappedPointer(stepMap, oldStepID)
					createdStep, err := s.repo.CreateImportedRunStep(step)
					if err != nil {
						return model.TestPlan{}, err
					}
					runStepMap[step.ID] = createdStep.ID
					for _, evidence := range step.Evidences {
						if _, err := s.importStepEvidence(createdRun.ID, createdStep.ID, evidence, files); err != nil {
							return model.TestPlan{}, err
						}
					}
				}
				for _, evidence := range sheet.Evidences {
					if _, err := s.importSheetEvidence(createdRun.ID, createdSheet.ID, evidence, files); err != nil {
						return model.TestPlan{}, err
					}
				}
			}
		}
	}
	_ = runMap
	_ = runGroupMap
	_ = runSheetMap
	_ = runStepMap
	if err := s.repo.TouchPlan(createdPlan.ID); err != nil {
		return model.TestPlan{}, err
	}
	return s.repo.GetPlan(createdPlan.ID)
}

func (s *Service) importDocument(planID int64, document exportDocument, files map[string]*zip.File) (model.TestDocument, error) {
	created, err := s.repo.CreateDocument(model.TestDocument{PlanID: planID, OriginalName: document.OriginalName, Description: document.Description})
	if err != nil {
		return model.TestDocument{}, err
	}
	if document.ExportPath == "" {
		return created, nil
	}
	zipFile := files[document.ExportPath]
	if zipFile == nil {
		_, _ = s.repo.DeleteDocument(created.ID)
		return model.TestDocument{}, fmt.Errorf("document file missing in zip: %s", document.ExportPath)
	}
	planDirectory := filepath.Join("data", "test-sheet", "documents", fmt.Sprintf("plan-%d", planID))
	storedName := fmt.Sprintf("doc-%d-%s", created.ID, safeFilename(document.OriginalName))
	storagePath := filepath.Join(planDirectory, storedName)
	if err := extractZipFile(zipFile, storagePath, maxDocumentUploadBytes); err != nil {
		_, _ = s.repo.DeleteDocument(created.ID)
		return model.TestDocument{}, err
	}
	return s.repo.UpdateDocumentFile(created.ID, storedName, storagePath, document.MimeType, document.SizeBytes, document.SHA256)
}

func (s *Service) importSheetEvidence(runID, runSheetID int64, evidence model.Evidence, files map[string]*zip.File) (model.Evidence, error) {
	created, err := s.repo.CreateEvidence(model.Evidence{RunSheetID: runSheetID, Name: evidence.Name, Comment: evidence.Comment})
	if err != nil {
		return model.Evidence{}, err
	}
	exportPath := evidence.ExportPath
	if exportPath == "" {
		return created, nil
	}
	storagePath := filepath.Join("data", "test-sheet", "runs", fmt.Sprintf("run-%d", runID), "evidences", fmt.Sprintf("sheet-%d", runSheetID), fmt.Sprintf("evidence-%d-%s", created.ID, safeFilename(evidence.Name)))
	if err := extractZipFile(files[exportPath], storagePath, maxDocumentUploadBytes); err != nil {
		_, _ = s.repo.DeleteEvidence(created.ID)
		return model.Evidence{}, err
	}
	return s.repo.UpdateEvidenceFile(created.ID, storagePath, evidence.MimeType, evidence.SizeBytes)
}

func (s *Service) importStepEvidence(runID, runStepID int64, evidence model.Evidence, files map[string]*zip.File) (model.Evidence, error) {
	created, err := s.repo.CreateStepEvidence(model.Evidence{RunStepID: runStepID, Name: evidence.Name})
	if err != nil {
		return model.Evidence{}, err
	}
	exportPath := evidence.ExportPath
	if exportPath == "" {
		return created, nil
	}
	storagePath := filepath.Join("data", "test-sheet", "runs", fmt.Sprintf("run-%d", runID), "evidences", fmt.Sprintf("step-%d", runStepID), fmt.Sprintf("evidence-%d-%s", created.ID, safeFilename(evidence.Name)))
	if err := extractZipFile(files[exportPath], storagePath, maxDocumentUploadBytes); err != nil {
		_, _ = s.repo.DeleteStepEvidence(created.ID)
		return model.Evidence{}, err
	}
	return s.repo.UpdateStepEvidenceFile(created.ID, storagePath, evidence.MimeType, evidence.SizeBytes)
}

func readExportZip(payload []byte) (exportManifest, exportData, map[string]*zip.File, error) {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("invalid zip file")
	}
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		if !isSafeZipPath(file.Name) {
			return exportManifest{}, exportData{}, nil, fmt.Errorf("zip contains an unsafe path: %s", file.Name)
		}
		files[file.Name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("manifest.json is missing")
	}
	dataFile := files["data.json"]
	if dataFile == nil {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("data.json is missing")
	}
	var manifest exportManifest
	if err := readZipJSON(manifestFile, &manifest); err != nil {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("invalid manifest.json")
	}
	if manifest.Format != exportFormat {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("unsupported export format")
	}
	if manifest.SchemaVersion != exportSchemaVersion {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("unsupported schema version")
	}
	var data exportData
	if err := readZipJSON(dataFile, &data); err != nil {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("invalid data.json")
	}
	if strings.TrimSpace(data.Plan.Name) == "" {
		return exportManifest{}, exportData{}, nil, fmt.Errorf("export data does not contain a valid plan")
	}
	return manifest, data, files, nil
}

func previewFromExport(manifest exportManifest, data exportData) model.ImportPreview {
	preview := model.ImportPreview{PlanName: data.Plan.Name, SchemaVersion: manifest.SchemaVersion, Groups: len(data.Groups), Sheets: len(data.Sheets), Documents: len(data.Documents), Runs: len(data.Runs)}
	for _, sheet := range data.Sheets {
		preview.Steps += len(sheet.Steps)
	}
	for _, run := range data.Runs {
		for _, group := range run.Groups {
			for _, sheet := range group.Sheets {
				preview.Evidences += len(sheet.Evidences)
				for _, step := range sheet.Steps {
					preview.Evidences += len(step.Evidences)
				}
			}
		}
	}
	return preview
}

func writeZipJSON(archive *zip.Writer, name string, payload any) error {
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func readZipJSON(file *zip.File, output any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return json.NewDecoder(io.LimitReader(reader, maxDocumentUploadBytes)).Decode(output)
}

func addFileToZip(archive *zip.Writer, zipPath, storagePath string) error {
	if !isSafeZipPath(zipPath) {
		return fmt.Errorf("unsafe zip path: %s", zipPath)
	}
	source, err := os.Open(storagePath)
	if err != nil {
		return err
	}
	defer source.Close()
	writer, err := archive.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, source)
	return err
}

func addEvidenceFiles(archive *zip.Writer, files map[string]string) error {
	for zipPath, sourcePath := range files {
		if err := addFileToZip(archive, zipPath, sourcePath); err != nil {
			return err
		}
	}
	return nil
}

func setEvidenceExportPaths(runs []model.TestRun) map[string]string {
	files := map[string]string{}
	for runIndex := range runs {
		run := &runs[runIndex]
		for groupIndex := range run.Groups {
			for sheetIndex := range run.Groups[groupIndex].Sheets {
				sheet := &run.Groups[groupIndex].Sheets[sheetIndex]
				for evidenceIndex := range sheet.Evidences {
					evidence := &sheet.Evidences[evidenceIndex]
					if evidence.Path != "" {
						evidence.ExportPath = path.Join("evidences", fmt.Sprintf("run-%d", run.ID), fmt.Sprintf("sheet-%d", sheet.ID), fmt.Sprintf("evidence-%d-%s", evidence.ID, safeFilename(evidence.Name)))
						files[evidence.ExportPath] = evidence.Path
					}
				}
				for stepIndex := range sheet.Steps {
					step := &sheet.Steps[stepIndex]
					for evidenceIndex := range step.Evidences {
						evidence := &step.Evidences[evidenceIndex]
						if evidence.Path != "" {
							evidence.ExportPath = path.Join("evidences", fmt.Sprintf("run-%d", run.ID), fmt.Sprintf("step-%d", step.ID), fmt.Sprintf("evidence-%d-%s", evidence.ID, safeFilename(evidence.Name)))
							files[evidence.ExportPath] = evidence.Path
						}
					}
				}
			}
		}
	}
	return files
}

func clearRunEvidences(run *model.TestRun) {
	for groupIndex := range run.Groups {
		for sheetIndex := range run.Groups[groupIndex].Sheets {
			run.Groups[groupIndex].Sheets[sheetIndex].Evidences = nil
			for stepIndex := range run.Groups[groupIndex].Sheets[sheetIndex].Steps {
				run.Groups[groupIndex].Sheets[sheetIndex].Steps[stepIndex].Evidences = nil
			}
		}
	}
}

func extractZipFile(file *zip.File, targetPath string, maxBytes int64) error {
	if file == nil {
		return fmt.Errorf("file missing in zip")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer target.Close()
	written, err := io.Copy(target, io.LimitReader(source, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		_ = os.Remove(targetPath)
		return fmt.Errorf("file is too large")
	}
	return nil
}

func isSafeZipPath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") || strings.Contains(value, "\\") {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned == value && cleaned != "." && !strings.HasPrefix(cleaned, "../") && cleaned != ".."
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func mappedPointer(mapping map[int64]int64, oldID int64) *int64 {
	if oldID == 0 {
		return nil
	}
	if newID := mapping[oldID]; newID != 0 {
		return &newID
	}
	return nil
}
