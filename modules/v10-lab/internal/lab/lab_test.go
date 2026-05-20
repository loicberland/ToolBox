package lab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigAcceptsSystemShape(t *testing.T) {
	config := Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
		Release: ReleaseConfig{
			ZipPath: "D:/release.zip",
		},
		Pipeline: []PipelineStep{
			{Action: "create-env"},
			{Action: "configure-gedix-cfg"},
			{Action: "start-maquette"},
		},
	}

	if err := ValidateConfig(config); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConfigReportsUnknownActionAndMissingField(t *testing.T) {
	config := Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
		Pipeline: []PipelineStep{
			{Action: "create-foo"},
			{Action: "create-machine", Params: map[string]any{"name": "FANUC"}},
		},
	}

	err := ValidateConfig(config)
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T %v", err, err)
	}
	if len(validationErr.Items) != 2 {
		t.Fatalf("expected 2 validation items, got %#v", validationErr.Items)
	}
}

func TestValidateConfigReportsInvalidDBTypeAndDuplicateDebugTarget(t *testing.T) {
	config := Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
		GedixConfig: GedixConfig{
			Services: map[string]ServiceDBConfig{
				"auth": {DBType: "db2"},
			},
		},
		Runtime: RuntimeConfig{DebugTargets: []string{"auth", "auth"}},
	}

	err := ValidateConfig(config)
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T %v", err, err)
	}
	message := strings.Join(validationErr.Items, "\n")
	if !strings.Contains(message, "dbType") || !strings.Contains(message, "doublon") {
		t.Fatalf("missing expected validation errors: %#v", validationErr.Items)
	}
}

func TestActionsForProductIncludesSystemAndGedixActions(t *testing.T) {
	actions := ActionsForProduct(GedixProdV10)
	byID := map[string]bool{}
	for _, action := range actions {
		byID[action.ID] = true
	}

	for _, id := range []string{"create-env", "configure-gedix-cfg", "start-maquette", "stop-maquette", "kill-gx-processes", "update-env", "create-machine-group", "create-machine", "create-cnc-folder"} {
		if !byID[id] {
			t.Fatalf("expected action %s in product actions", id)
		}
	}
}

func TestDetectEnvAndDebugTargets(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "env_demo", "app_prod", "connector-focas-01"))
	mustWrite(t, filepath.Join(root, "gx-front.exe"), "")
	mustWrite(t, filepath.Join(root, "gedix.cfg"), "")
	mustWrite(t, filepath.Join(root, "env_demo", "app_prod", "gx-app.exe"), "")
	mustWrite(t, filepath.Join(root, "env_demo", "app_prod", "gx-auth.exe"), "")
	mustWrite(t, filepath.Join(root, "env_demo", "app_prod", "connector-focas-01", "gx-connector.exe"), "")

	paths, err := DetectGedixPaths(Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
		Maquette: MaquetteConfig{
			TargetPath: root,
			AppName:    "prod",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if paths.EnvName != "env_demo" || !strings.HasSuffix(paths.AppPath, filepath.Join("env_demo", "app_prod")) {
		t.Fatalf("unexpected paths: %#v", paths)
	}
	service, err := DetectDebugTarget(paths, "auth")
	if err != nil || service.Kind != DebugTargetService {
		t.Fatalf("expected service, got %#v err=%v", service, err)
	}
	connector, err := DetectDebugTarget(paths, "connector-focas-01")
	if err != nil || connector.Kind != DebugTargetConnector {
		t.Fatalf("expected connector, got %#v err=%v", connector, err)
	}
}

func TestCfgUpdatesRootPortDBAndSQLite(t *testing.T) {
	content := `# port=80
fqdn="old"

[environments.env_demo.applications.prod.services.auth]
db-type="oracle"
db-dsn="/old"

[environments.env_demo.applications.prod.services.filestore]
db-type="oracle"
db-dsn="/old"
`
	content = setRootKey(content, "fqdn", "localhost", true)
	content = setPort(content, 20260)
	content = setSectionKey(content, "environments.env_demo.applications.prod.services.auth", "db-type", "oracle", true)
	content = setSectionKey(content, "environments.env_demo.applications.prod.services.auth", "db-dsn", "/USER/PASSWORD@HOST:1521/SERVICE", true)
	content = removeOrCommentKey(content, "environments.env_demo.applications.prod.services.filestore", "db-type")
	content = removeOrCommentKey(content, "environments.env_demo.applications.prod.services.filestore", "db-dsn")

	for _, expected := range []string{
		`fqdn="localhost"`,
		`port=20260`,
		`db-dsn="/USER/PASSWORD@HOST:1521/SERVICE"`,
		`#db-type="oracle"`,
		`#db-dsn="/old"`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in:\n%s", expected, content)
		}
	}
}

func TestEnsureSafeDeletePathRejectsRoots(t *testing.T) {
	if err := ensureSafeDeletePath(filepath.VolumeName(os.TempDir()) + string(os.PathSeparator)); err == nil {
		t.Fatal("expected root deletion to be rejected")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
