package lab

import (
	"archive/zip"
	"bytes"
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

func TestConsoleCommandLineQuotesExecutablePathWithSpaces(t *testing.T) {
	workDir := `D:\Data\Gedix10\01_Clients\GMP Industrie`
	got := consoleCommandLine(
		workDir+`\gx-front.exe`,
		"listen",
	)
	want := `"D:\Data\Gedix10\01_Clients\GMP Industrie\gx-front.exe" listen`
	if got != want {
		t.Fatalf("unexpected console command:\ngot:  %s\nwant: %s", got, want)
	}

	cmdArgs := openConsoleArgs("V10 Lab gx-front", `C:\Temp\v10-lab-run-1.cmd`)
	if len(cmdArgs) != 7 || cmdArgs[5] != "call" || cmdArgs[6] != `C:\Temp\v10-lab-run-1.cmd` {
		t.Fatalf("console should launch the generated script via call, got %#v", cmdArgs)
	}
}

func TestConsoleLauncherScriptQuotesPathsWithoutBackslashEscapedQuotes(t *testing.T) {
	content := consoleLauncherScriptContent(
		`D:\Data\Gedix10\01_Clients\GMP Industrie`,
		`D:\Data\Gedix10\01_Clients\GMP Industrie\gx-front.exe`,
		"listen",
	)
	for _, expected := range []string{
		`cd /d "D:\Data\Gedix10\01_Clients\GMP Industrie"`,
		`"D:\Data\Gedix10\01_Clients\GMP Industrie\gx-front.exe" listen`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected script to contain %q, got:\n%s", expected, content)
		}
	}
	if strings.Contains(content, `\"`) {
		t.Fatalf("batch script must not contain backslash-escaped quotes:\n%s", content)
	}
	if !strings.Contains(content, "\r\n") {
		t.Fatalf("batch script should use CRLF line endings, got %q", content)
	}
}

func TestConsoleCommandLineKeepsDebugTargetsAsSingleArgument(t *testing.T) {
	got := consoleCommandLine(
		`D:\Data\Gedix10\01_Clients\GMP Industrie\env_live\app_prod\gx-app.exe`,
		"run",
		"-e",
		"auth,connector-focas-01",
	)
	want := `"D:\Data\Gedix10\01_Clients\GMP Industrie\env_live\app_prod\gx-app.exe" run -e auth,connector-focas-01`
	if got != want {
		t.Fatalf("unexpected debug console command:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestConsoleLauncherScriptQuotesArgumentsWithSpaces(t *testing.T) {
	content := consoleLauncherScriptContent(
		`D:\Data\Gedix10\01_Clients\GMP Industrie\env_live\app_prod`,
		`D:\Data\Gedix10\01_Clients\GMP Industrie\env_live\app_prod\gx-app.exe`,
		"run",
		"-e",
		"auth,connector-focas-01",
		"--some-path",
		`D:\A B\file.txt`,
	)
	for _, expected := range []string{
		`"D:\Data\Gedix10\01_Clients\GMP Industrie\env_live\app_prod\gx-app.exe" run -e auth,connector-focas-01 --some-path "D:\A B\file.txt"`,
		`-e auth,connector-focas-01`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected script to contain %q, got:\n%s", expected, content)
		}
	}
	if strings.Contains(content, `-e auth connector-focas-01`) {
		t.Fatalf("debug target list should stay a single argument:\n%s", content)
	}
}

func TestDebugTargetsUseCommaSeparatedExclusionArg(t *testing.T) {
	got := debugExclusionArg([]string{"auth", "connector-focas-01"})
	if got != "auth,connector-focas-01" {
		t.Fatalf("expected comma separated debug targets, got %q", got)
	}
}

func TestRunActionKeepsInternalUTF8Logs(t *testing.T) {
	var output bytes.Buffer
	config := Config{
		Name:    "ticket-T5808",
		Product: GedixProdV10,
	}

	if err := RunAction(t.Context(), config, "stop-maquette", &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Exécution terminée.") {
		t.Fatalf("expected UTF-8 internal log, got:\n%s", text)
	}
	if strings.Contains(text, "ExÃ") {
		t.Fatalf("internal log was mojibaked:\n%s", text)
	}
}

func TestDecodeCommandOutputKeepsUTF8AndDecodesWindowsCodePages(t *testing.T) {
	utf8Text := "Exécution terminée."
	if got := decodeCommandOutput([]byte(utf8Text)); got != utf8Text {
		t.Fatalf("UTF-8 command output should not be converted, got %q", got)
	}

	cp850OperationReussie := []byte{'O', 'p', 0x82, 'r', 'a', 't', 'i', 'o', 'n', ' ', 'r', 0x82, 'u', 's', 's', 'i', 'e'}
	if got := decodeCommandOutput(cp850OperationReussie); got != "Opération réussie" {
		t.Fatalf("expected CP850 output to be decoded, got %q", got)
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
	content = appendRawConfigToSection(content, section, "key1=\"value1\"\nkey2=\"value2\"")
	if !strings.Contains(content, `type="focas"`) || !strings.Contains(content, `host="127.0.0.1"`) {
		t.Fatalf("connector type/host should remain:\n%s", content)
	}
	if !strings.Contains(content, `key1="value1"`) || !strings.Contains(content, `key2="value2"`) {
		t.Fatalf("raw config not inserted:\n%s", content)
	}
	if strings.Count(content, `key1="value1"`) != 1 || strings.Count(content, `key2="value2"`) != 1 {
		t.Fatalf("connector raw config should be idempotent:\n%s", content)
	}
	content = appendRawConfigToSection(content, section, "type=\"ignored\"\nhost=\"ignored\"\nkey1=\"new\"")
	if strings.Contains(content, `type="ignored"`) || strings.Contains(content, `host="ignored"`) {
		t.Fatalf("connector type/host should not be overwritten:\n%s", content)
	}
	if !strings.Contains(content, `key1="new"`) || strings.Contains(content, `key1="value1"`) {
		t.Fatalf("connector raw config should update existing keys:\n%s", content)
	}
	missing := appendRawConfigToSection(content, "environments.demo.applications.prod.connectors.connector-unknown", `x="y"`)
	if missing != content {
		t.Fatal("missing connector section should not be created")
	}
}

func TestCfgConnectorRawConfigPreservesTripleQuotedBlocks(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.connectors.connector-focas-01"
	raw := connectorMultilineRawConfig()
	content = appendRawConfigToSection(content, section, raw)
	content = appendRawConfigToSection(content, section, raw)

	for _, expected := range []string{
		`filesystem-delete-remote-after-unload=true`,
		`filesystem-header-content = """`,
		`(DOSSIER =${job.name})`,
		`(INDICE  =${job.version})`,
		`(ETAT    =${state.name})`,
		`(DATE MOD=${program.created_at})`,
		`(CREE PAR=${program.created_by})`,
		`(TRANSF  =${date.now_fr})`,
		`filesystem-header-first-line = "(******* ENTETE GEDIX *******)"`,
		`filesystem-header-last-line = "(***** FIN ENTETE GEDIX *****)"`,
		`filesystem-header-line-number = 1`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected preserved raw line %q in:\n%s", expected, content)
		}
	}
	if strings.Count(content, `"""`) != 2 {
		t.Fatalf("expected opening and closing triple quotes only once:\n%s", content)
	}
	if strings.Count(content, `filesystem-delete-remote-after-unload`) != 1 ||
		strings.Count(content, `filesystem-header-content`) != 1 ||
		strings.Count(content, `filesystem-header-first-line`) != 1 {
		t.Fatalf("expected connector raw config to be idempotent:\n%s", content)
	}
}

func TestCfgConnectorRawConfigReplacesExistingMultilineBlock(t *testing.T) {
	section := "environments.demo.applications.prod.connectors.connector-focas-01"
	content := minimalGedixCfg()
	content = appendRawConfigToSection(content, section, "filesystem-header-content = \"\"\"\nOLD\n\"\"\"")
	content = appendRawConfigToSection(content, section, "filesystem-header-content = \"\"\"\nNEW\n\"\"\"")

	if strings.Contains(content, "OLD") || !strings.Contains(content, "NEW") {
		t.Fatalf("expected multiline block to be replaced:\n%s", content)
	}
	if strings.Count(content, `filesystem-header-content`) != 1 || strings.Count(content, `"""`) != 2 {
		t.Fatalf("expected one multiline block:\n%s", content)
	}
}

func TestCfgServiceExtraKeysUseRawValuesAndCanUpdateHost(t *testing.T) {
	content := minimalGedixCfg()
	section := "environments.demo.applications.prod.services.auth"
	content = setSectionRawBlock(content, section, cfgEntry{Key: "host", Lines: []string{`host="127.0.0.2"`}})
	content = setSectionRawBlock(content, section, cfgEntry{Key: "custom-key", Lines: []string{`custom-key=true`}})
	content = setSectionRawBlock(content, section, cfgEntry{Key: "quoted-key", Lines: []string{`quoted-key="texte avec espaces"`}})

	for _, expected := range []string{`host="127.0.0.2"`, `custom-key=true`, `quoted-key="texte avec espaces"`} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in:\n%s", expected, content)
		}
	}
	if strings.Count(content, `host="127.0.0.2"`) != 1 || strings.Contains(content, `custom-key="true"`) {
		t.Fatalf("expected raw service extra keys without duplicates or forced quotes:\n%s", content)
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

func TestUpdateEnvValidation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Gedix_Test")
	makeValidMaquetteTarget(t, target)
	zipPath := filepath.Join(root, "release.zip")
	mustZip(t, zipPath, map[string]string{})

	tests := []struct {
		name    string
		zipPath string
		target  string
		want    string
	}{
		{name: "empty release", zipPath: "", target: target, want: "release.zipPath est requis"},
		{name: "missing release", zipPath: filepath.Join(root, "missing.zip"), target: target, want: "ZIP release introuvable"},
		{name: "not zip", zipPath: mustFile(t, filepath.Join(root, "release.txt"), ""), target: target, want: "fichier .zip"},
		{name: "missing target", zipPath: zipPath, target: filepath.Join(root, "missing-target"), want: "dossier cible de maquette introuvable"},
		{name: "dangerous target", zipPath: zipPath, target: filepath.VolumeName(os.TempDir()) + string(os.PathSeparator), want: "chemin cible dangereux"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateEnvInputs(tt.zipPath, tt.target)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestUpdateEnvReportsMissingGXInZip(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Gedix_Test")
	makeValidMaquetteTarget(t, target)
	zipPath := filepath.Join(root, "release.zip")
	mustZip(t, zipPath, map[string]string{"README.txt": "no gx here"})
	var output bytes.Buffer

	err := UpdateEnv(ActionContext{
		Writer: &output,
		Config: Config{
			Name:    "Test",
			Product: GedixProdV10,
			Release: ReleaseConfig{
				ZipPath: zipPath,
				WorkDir: filepath.Join(root, "work"),
			},
			Maquette: MaquetteConfig{TargetPath: target, AppName: "prod"},
		},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "gx.exe introuvable") {
		t.Fatalf("expected missing gx.exe, got %v\nlogs:\n%s", err, output.String())
	}
}

func TestCopyDirForUpdatePreservesConfigLogsAndExistingFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "Gedix")
	target := filepath.Join(root, "Gedix_Test")
	mustWrite(t, filepath.Join(source, "gx.exe"), "new gx")
	mustWrite(t, filepath.Join(source, "gx-front.exe"), "new front")
	mustWrite(t, filepath.Join(source, "gedix.cfg"), "source cfg")
	mustWrite(t, filepath.Join(source, "env_demo", "app_prod", "gx-app.exe"), "new app")
	mustWrite(t, filepath.Join(source, "log", "old.log"), "source log")
	mustWrite(t, filepath.Join(target, "gedix.cfg"), "target cfg")
	mustWrite(t, filepath.Join(target, "log", "keep.log"), "target log")
	mustWrite(t, filepath.Join(target, "env_demo", "app_prod", "old.exe"), "keep me")

	if err := copyDirForUpdate(source, target); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(target, "gx.exe"), "new gx")
	assertFileContent(t, filepath.Join(target, "gx-front.exe"), "new front")
	assertFileContent(t, filepath.Join(target, "env_demo", "app_prod", "gx-app.exe"), "new app")
	assertFileContent(t, filepath.Join(target, "gedix.cfg"), "target cfg")
	assertFileContent(t, filepath.Join(target, "log", "keep.log"), "target log")
	assertFileContent(t, filepath.Join(target, "env_demo", "app_prod", "old.exe"), "keep me")
	if _, err := os.Stat(filepath.Join(target, "log", "old.log")); !os.IsNotExist(err) {
		t.Fatalf("source log should not be copied, stat err=%v", err)
	}
}

func connectorMultilineRawConfig() string {
	return `filesystem-delete-remote-after-unload=true
filesystem-header-content = """
(DOSSIER =${job.name})
(INDICE  =${job.version})
(ETAT    =${state.name})
(DATE MOD=${program.created_at})
(CREE PAR=${program.created_by})
(TRANSF  =${date.now_fr})
"""
filesystem-header-first-line = "(******* ENTETE GEDIX *******)"
filesystem-header-last-line = "(***** FIN ENTETE GEDIX *****)"
filesystem-header-line-number = 1`
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

func mustFile(t *testing.T, path string, content string) string {
	t.Helper()
	mustWrite(t, path, content)
	return path
}

func mustZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(out)
	for name, content := range files {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func makeValidMaquetteTarget(t *testing.T, target string) {
	t.Helper()
	mustWrite(t, filepath.Join(target, "gx-front.exe"), "")
	mustMkdir(t, filepath.Join(target, "env_demo"))
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected content for %s: got %q want %q", path, string(data), expected)
	}
}
