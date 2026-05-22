package lab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigAcceptsSystemShape(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "release.zip")
	mustWrite(t, zipPath, "")
	config := Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
		Release: ReleaseConfig{
			ZipPath: zipPath,
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
	if paths.EnvName != "demo" || !strings.HasSuffix(paths.AppPath, filepath.Join("env_demo", "app_prod")) {
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

func TestQuoteCmdArgDoesNotQuoteDebugTargetList(t *testing.T) {
	if got := quoteCmdArg("auth,connector-focas-01"); got != "auth,connector-focas-01" {
		t.Fatalf("debug target list should not be quoted, got %q", got)
	}
	if got := quoteCmdArg(`D:\Program Files\Gedix\gx-app.exe`); got != `"D:\Program Files\Gedix\gx-app.exe"` {
		t.Fatalf("path with spaces should be quoted, got %q", got)
	}
}

func TestCfgUpdatesRootPortDBAndSQLite(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.services.auth"
	content = setRootKey(content, "fqdn", "localhost", true)
	content = setPort(content, 20260)
	content = setSectionKey(content, section, "db-type", "oracle", true)
	content = setSectionKey(content, section, "db-dsn", "/USER/PASSWORD@HOST:1521/SERVICE", true)
	content = setSectionKey(content, section, "my-key", "my-value", true)
	content = setSectionKey(content, section, "db-type", "oracle", true)
	content = setSectionKey(content, section, "db-dsn", "/USER/PASSWORD@HOST:1521/SERVICE", true)

	for _, expected := range []string{
		`fqdn="localhost"`,
		`port=20260`,
		`# db-type=""`,
		`# db-dsn=""`,
		`db-type="oracle"`,
		`db-dsn="/USER/PASSWORD@HOST:1521/SERVICE"`,
		`my-key="my-value"`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in:\n%s", expected, content)
		}
	}
	if strings.Count(content, `db-type="oracle"`) != 1 {
		t.Fatalf("expected idempotent db-type in:\n%s", content)
	}
	if strings.Count(content, `db-dsn="/USER/PASSWORD@HOST:1521/SERVICE"`) != 1 {
		t.Fatalf("expected idempotent db-dsn in:\n%s", content)
	}
}

func TestCfgSQLiteCommentsActiveDBWithoutRemovingTemplateComments(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.services.auth"
	content = setSectionKey(content, section, "db-type", "oracle", true)
	content = setSectionKey(content, section, "db-dsn", "/old", true)
	content = removeOrCommentKey(content, section, "db-type")
	content = removeOrCommentKey(content, section, "db-dsn")

	if !strings.Contains(content, `#db-type="oracle"`) || !strings.Contains(content, `#db-dsn="/old"`) {
		t.Fatalf("expected active DB keys to be commented:\n%s", content)
	}
	if !strings.Contains(content, `# db-type=""`) || !strings.Contains(content, `# db-dsn=""`) {
		t.Fatalf("expected template comments to remain:\n%s", content)
	}
}

func TestCfgSQLiteWithExplicitDSNWritesDBKeys(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.services.auth"
	content = setSectionKey(content, section, "db-type", "sqlite", true)
	content = setSectionKey(content, section, "db-dsn", "custom.sqlite", true)

	if !strings.Contains(content, `db-type="sqlite"`) || !strings.Contains(content, `db-dsn="custom.sqlite"`) {
		t.Fatalf("expected explicit sqlite DB keys:\n%s", content)
	}
}

func TestCfgMissingServiceSectionDoesNotCreateSection(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.services.fake-service"
	next := setSectionKey(content, section, "db-type", "oracle", true)
	if next != content {
		t.Fatalf("missing section should not change content")
	}
	if sectionExists(next, section) {
		t.Fatal("missing service section was created")
	}
}

func TestCfgConnectorExistingAndMissing(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.connectors.connector-focas-01"
	content = appendRawConfigToSection(content, section, "key1=\"value1\"\nkey2=\"value2\"")
	if !strings.Contains(content, `type="focas"`) || !strings.Contains(content, `host="127.0.0.1"`) {
		t.Fatalf("connector type/host should remain:\n%s", content)
	}
	if !strings.Contains(content, `key1="value1"`) || !strings.Contains(content, `key2="value2"`) {
		t.Fatalf("raw config not inserted:\n%s", content)
	}
	missing := appendRawConfigToSection(content, "environments.demo.applications.prod.connectors.connector-unknown", `x="y"`)
	if missing != content {
		t.Fatal("missing connector section should not be created")
	}
}

func TestEnsureSafeDeletePathRejectsRoots(t *testing.T) {
	if err := ensureSafeDeletePath(filepath.VolumeName(os.TempDir()) + string(os.PathSeparator)); err == nil {
		t.Fatal("expected root deletion to be rejected")
	}
}

func TestSafeRemoveTempDir(t *testing.T) {
	root := t.TempDir()
	temp := filepath.Join(root, "ticket-T5808-20260520-153000")
	final := filepath.Join(root, "final")
	mustMkdir(t, temp)
	mustMkdir(t, final)
	if err := safeRemoveTempDir(temp, root, final); err != nil {
		t.Fatalf("expected temp removal, got %v", err)
	}
	if _, err := os.Stat(temp); !os.IsNotExist(err) {
		t.Fatalf("expected temp dir removed, stat err=%v", err)
	}
	if err := safeRemoveTempDir(final, final); err == nil {
		t.Fatal("expected protected final path to be rejected")
	}
	if err := safeRemoveTempDir("", final); err == nil {
		t.Fatal("expected empty path to be rejected")
	}
}

func minimalGedixCfg() string {
	return `fqdn="old-host"
# port=80

[environments.demo.applications.prod.services]

[environments.demo.applications.prod.services.auth]
host="127.0.0.1"
# db-type=""
# db-dsn=""

[environments.demo.applications.prod.services.entreprise]
host="127.0.0.1"
# db-type=""
# db-dsn=""

[environments.demo.applications.prod.connectors]

[environments.demo.applications.prod.connectors.connector-focas-01]
type="focas"
host="127.0.0.1"
`
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
