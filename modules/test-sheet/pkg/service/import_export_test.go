package service

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"toolBox/modules/test-sheet/pkg/model"
	"toolBox/modules/test-sheet/pkg/repository"
)

func TestExportImportPlanWithDocumentsCreatesNewIDsAndKeepsRelations(t *testing.T) {
	svc := newTestService(t)
	plan, err := svc.CreatePlan(model.PlanInput{Name: "Plan exporte", Description: "Description"})
	if err != nil {
		t.Fatal(err)
	}
	groups, err := svc.ListGroups(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := svc.CreateSheetInGroup(groups[0].ID, model.SheetInput{Name: "Fiche", ExecutionOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	step, err := svc.CreateStep(sheet.ID, model.StepInput{Action: "Cliquer", ExpectedResult: "OK", ExecutionOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	document := createTestDocument(t, svc, plan.ID, "doc.txt", "contenu")
	if err := svc.LinkSheetDocument(sheet.ID, document.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.LinkStepDocument(step.ID, document.ID); err != nil {
		t.Fatal(err)
	}

	payload, err := svc.ExportPlan(plan.ID, DefaultExportOptions())
	if err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewImportZip(payload)
	if err != nil {
		t.Fatal(err)
	}
	if preview.PlanName != plan.Name || preview.Groups != 1 || preview.Sheets != 1 || preview.Steps != 1 || preview.Documents != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	result, err := svc.ImportPlanZip(payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.PlanID == plan.ID {
		t.Fatalf("import reused original plan id %d", plan.ID)
	}
	importedGroups, err := svc.ListGroups(result.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if len(importedGroups) != 1 || importedGroups[0].ID == groups[0].ID {
		t.Fatalf("groups were not recreated: %+v", importedGroups)
	}
	importedSheets, err := svc.ListSheetsByGroup(importedGroups[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(importedSheets) != 1 || importedSheets[0].ID == sheet.ID || importedSheets[0].GroupID != importedGroups[0].ID {
		t.Fatalf("sheets relation not preserved with new ids: %+v", importedSheets)
	}
	if len(importedSheets[0].Steps) != 1 || importedSheets[0].Steps[0].ID == step.ID || importedSheets[0].Steps[0].SheetID != importedSheets[0].ID {
		t.Fatalf("steps relation not preserved with new ids: %+v", importedSheets[0].Steps)
	}
	if len(importedSheets[0].Documents) != 1 || importedSheets[0].Documents[0].ID == document.ID {
		t.Fatalf("sheet documents not recreated: %+v", importedSheets[0].Documents)
	}
	if len(importedSheets[0].Steps[0].Documents) != 1 || importedSheets[0].Steps[0].Documents[0].ID != importedSheets[0].Documents[0].ID {
		t.Fatalf("step document relation not preserved: %+v", importedSheets[0].Steps[0].Documents)
	}
	if _, err := os.Stat(importedSheets[0].Documents[0].StoragePath); err != nil {
		t.Fatalf("imported physical document missing: %v", err)
	}
}

func TestImportRejectsInvalidManifest(t *testing.T) {
	svc := newTestService(t)
	payload := zipPayload(t, map[string]string{
		"manifest.json": `{"format":"wrong","schemaVersion":1}`,
		"data.json":     `{"plan":{"name":"Plan"}}`,
	})
	if _, err := svc.PreviewImportZip(payload); err == nil {
		t.Fatal("expected invalid manifest to be rejected")
	}
}

func TestImportRejectsUnsupportedSchemaVersion(t *testing.T) {
	svc := newTestService(t)
	payload := zipPayload(t, map[string]string{
		"manifest.json": `{"format":"toolbox-test-sheet-export","schemaVersion":99}`,
		"data.json":     `{"plan":{"name":"Plan"}}`,
	})
	if _, err := svc.PreviewImportZip(payload); err == nil {
		t.Fatal("expected unsupported schema version to be rejected")
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
	repo, err := repository.Open(filepath.Join(temp, "test-sheet.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return New(repo)
}

func createTestDocument(t *testing.T, svc *Service, planID int64, name, content string) model.TestDocument {
	t.Helper()
	created, err := svc.repo.CreateDocument(model.TestDocument{PlanID: planID, OriginalName: name})
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join("data", "test-sheet", "documents", "source")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	storagePath := filepath.Join(directory, name)
	if err := os.WriteFile(storagePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.repo.UpdateDocumentFile(created.ID, name, storagePath, "text/plain", int64(len(content)), "hash")
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func zipPayload(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
