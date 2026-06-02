package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartScriptUsesToolboxConfigForURL(t *testing.T) {
	content := startScriptContent()
	for _, expected := range []string{
		`api-toolbox.exe" server --config "%~dp0toolbox.cfg"`,
		`web-server-toolbox.exe" start --config "%~dp0toolbox.cfg" --open`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected start script to contain %q, got:\n%s", expected, content)
		}
	}
	for _, unexpected := range []string{"powershell", "TOOLBOX_URL", "ToolBox Url.ps1"} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("start script should not contain %q, got:\n%s", unexpected, content)
		}
	}
}

func TestEnsureStartScriptRemovesLegacyURLScript(t *testing.T) {
	root := t.TempDir()
	legacyScriptPath := filepath.Join(root, "ToolBox Url.ps1")
	if err := os.WriteFile(legacyScriptPath, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ensureStartScript(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "ToolBox Start.bat")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacyScriptPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy PowerShell URL script to be removed, got %v", err)
	}
}

func TestPayloadMarkerUsesPackageName(t *testing.T) {
	if string(payloadMarker) != "TOOLBOX_PACKAGE_PAYLOAD_V1" {
		t.Fatalf("unexpected payload marker: %s", string(payloadMarker))
	}
}

func TestTargetAccessErrorExplainsPermissionProblem(t *testing.T) {
	err := targetAccessError(`C:\Program Files\ToolBox`, os.ErrPermission)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !strings.Contains(err.Error(), "Accès refusé au dossier cible") {
		t.Fatalf("missing access denied guidance: %v", err)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected original permission error to be wrapped: %v", err)
	}
}
