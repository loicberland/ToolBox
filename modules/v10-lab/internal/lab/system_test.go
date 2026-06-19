package lab

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCreateTargetAvailableExistingWithoutOverwrite(t *testing.T) {
	target := t.TempDir()

	err := checkCreateTargetAvailable(target, false)
	if err == nil {
		t.Fatal("expected existing target error")
	}
	if !strings.Contains(err.Error(), "overwrite=false") || !strings.Contains(err.Error(), target) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckCreateTargetAvailableExistingWithOverwrite(t *testing.T) {
	target := t.TempDir()

	if err := checkCreateTargetAvailable(target, true); err != nil {
		t.Fatalf("expected existing target with overwrite to be accepted, got %v", err)
	}
}

func TestCheckCreateTargetAvailableMissingWithoutOverwrite(t *testing.T) {
	target := t.TempDir() + "-missing"

	if err := checkCreateTargetAvailable(target, false); err != nil {
		t.Fatalf("expected missing target to be accepted, got %v", err)
	}
}

func TestCheckCreateTargetAvailableEmptyTarget(t *testing.T) {
	err := checkCreateTargetAvailable(" ", false)
	if err == nil {
		t.Fatal("expected empty target error")
	}
	if !strings.Contains(err.Error(), "targetPath vide") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseZipPath(t *testing.T) {
	root := t.TempDir()
	validZip := filepath.Join(root, "release.ZIP")
	mustWrite(t, validZip, "not a real archive")
	notZip := filepath.Join(root, "release.txt")
	mustWrite(t, notZip, "")

	tests := []struct {
		name    string
		zipPath string
		want    string
	}{
		{name: "empty", zipPath: " ", want: "Sélectionnez un ZIP de release"},
		{name: "missing", zipPath: filepath.Join(root, "missing.zip"), want: "introuvable"},
		{name: "directory", zipPath: root, want: "désigne un dossier"},
		{name: "bad extension", zipPath: notZip, want: "extension .zip"},
		{name: "existing zip", zipPath: validZip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReleaseZipPath(tt.zipPath)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("expected valid zip path, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestCreateEnvRejectsMissingZipBeforePreparingTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Gedix_Target")
	workDir := filepath.Join(root, "work")
	var output bytes.Buffer

	err := CreateEnv(ActionContext{
		Writer: &output,
		Config: Config{
			Name:    "Demo",
			Product: GedixProdV10,
			Release: ReleaseConfig{
				ZipPath: filepath.Join(root, "missing.zip"),
				WorkDir: workDir,
			},
			Maquette: MaquetteConfig{TargetPath: target},
		},
	}, nil)

	if err == nil || !strings.Contains(err.Error(), "introuvable") {
		t.Fatalf("expected missing zip error, got %v", err)
	}
	if _, statErr := os.Stat(workDir); !os.IsNotExist(statErr) {
		t.Fatalf("work dir should not be created before zip validation, stat=%v", statErr)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target should not be created before zip validation, stat=%v", statErr)
	}
}

func TestUpdateEnvRejectsMissingZipBeforeTempOrTargetChanges(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Gedix_Target")
	workDir := filepath.Join(root, "work")
	makeValidMaquetteTarget(t, target)
	marker := filepath.Join(target, "keep.txt")
	mustWrite(t, marker, "keep")
	var output bytes.Buffer

	err := UpdateEnv(ActionContext{
		Writer: &output,
		Config: Config{
			Name:    "Demo",
			Product: GedixProdV10,
			Release: ReleaseConfig{
				ZipPath: filepath.Join(root, "missing.zip"),
				WorkDir: workDir,
			},
			Maquette: MaquetteConfig{TargetPath: target, AppName: "prod"},
		},
	}, nil)

	if err == nil || !strings.Contains(err.Error(), "introuvable") {
		t.Fatalf("expected missing zip error, got %v", err)
	}
	if _, statErr := os.Stat(workDir); !os.IsNotExist(statErr) {
		t.Fatalf("work dir should not be created before zip validation, stat=%v", statErr)
	}
	assertFileContent(t, marker, "keep")
}
