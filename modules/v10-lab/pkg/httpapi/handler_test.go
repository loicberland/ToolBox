package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"toolBox/modules/v10-lab/internal/lab"
	"toolBox/pkg/toolboxruntime"

	"github.com/gorilla/mux"
)

func TestMaquetteCRUDAndValidate(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	var summaries []MaquetteSummary
	getJSON(t, router, "/api/v10-lab/maquettes", &summaries, http.StatusOK)
	if len(summaries) != 1 || summaries[0].Name != config.Name || summaries[0].AppName != "prod" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}

	var loaded lab.Config
	getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808", &loaded, http.StatusOK)
	if loaded.GedixConfig.FQDN != "localhost" {
		t.Fatalf("unexpected loaded config: %#v", loaded)
	}

	loaded.GedixConfig.Port = 20260
	postJSON(t, router, http.MethodPut, "/api/v10-lab/maquettes/ticket-T5808", loaded, http.StatusOK)
	getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808", &loaded, http.StatusOK)
	if loaded.GedixConfig.Port != 20260 {
		t.Fatalf("update was not persisted: %#v", loaded)
	}

	var validation ExecutionResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/ticket-T5808/validate", nil, &validation, http.StatusOK)
	if validation.Status != "valid" {
		t.Fatalf("unexpected validation response: %#v", validation)
	}

	var tokenStatus APITokenStatus
	getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808/api-token", &tokenStatus, http.StatusOK)
	if tokenStatus.HasToken {
		t.Fatal("expected no API token initially")
	}
	postJSONInto(t, router, "/api/v10-lab/maquettes/ticket-T5808/api-token", APITokenRequest{Token: "secret-token"}, &tokenStatus, http.StatusOK, http.MethodPut)
	if !tokenStatus.HasToken {
		t.Fatal("expected API token after save")
	}
	getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808", &loaded, http.StatusOK)
	if payload, err := json.Marshal(loaded); err != nil || strings.Contains(string(payload), "secret-token") {
		t.Fatalf("token leaked through maquette GET: %s err=%v", payload, err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v10-lab/maquettes/ticket-T5808", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "modules", "v10-lab", "data", "maquettes", "ticket-T5808", "maquette.json")); !os.IsNotExist(err) {
		t.Fatalf("expected registration json to be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "modules", "v10-lab", "data", "maquettes", "ticket-T5808", "data", "secrets.json")); !os.IsNotExist(err) {
		t.Fatalf("expected API token secret to be removed, err=%v", err)
	}
}

func TestDeleteMaquetteKeepsPhysicalDirectoryByDefault(t *testing.T) {
	root, router, config, target := registeredMaquetteForDelete(t)
	_ = root
	mustWrite(t, filepath.Join(target, "marker.txt"), "keep")
	if err := lab.SaveAPIToken(config.Name, "secret-token"); err != nil {
		t.Fatal(err)
	}
	deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name, http.StatusNoContent)
	if _, err := os.Stat(filepath.Join(target, "marker.txt")); err != nil {
		t.Fatalf("physical target should remain: %v", err)
	}
	assertMaquetteRegistrationDeleted(t, config.Name)
}

func TestDeleteMaquetteKeepsPhysicalDirectoryWhenFalse(t *testing.T) {
	_, router, config, target := registeredMaquetteForDelete(t)
	mustWrite(t, filepath.Join(target, "marker.txt"), "keep")
	deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=false", http.StatusNoContent)
	if _, err := os.Stat(filepath.Join(target, "marker.txt")); err != nil {
		t.Fatalf("physical target should remain: %v", err)
	}
	assertMaquetteRegistrationDeleted(t, config.Name)
}

func TestDeleteMaquetteRemovesPhysicalDirectoryWhenRequested(t *testing.T) {
	_, router, config, target := registeredMaquetteForDelete(t)
	mustWrite(t, filepath.Join(target, "nested", "marker.txt"), "remove")
	if err := lab.SaveAPIToken(config.Name, "secret-token"); err != nil {
		t.Fatal(err)
	}
	deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=true", http.StatusNoContent)
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("physical target should be removed, got %v", err)
	}
	assertMaquetteRegistrationDeleted(t, config.Name)
}

func TestDeleteMaquetteDirectoryValidationAndMissingDirectory(t *testing.T) {
	t.Run("missing directory is accepted", func(t *testing.T) {
		_, router, config, _ := registeredMaquetteForDelete(t)
		deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=true", http.StatusNoContent)
		assertMaquetteRegistrationDeleted(t, config.Name)
	})
	for _, target := range []string{"", "relative/path", string(filepath.Separator)} {
		t.Run("rejects dangerous target "+target, func(t *testing.T) {
			_, router, config, _ := registeredMaquetteForDelete(t)
			setRegisteredTargetPath(t, config.Name, target)
			deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=true", http.StatusBadRequest)
			if _, _, err := lab.LoadRegisteredConfig(config.Name); err != nil {
				t.Fatalf("registration should remain: %v", err)
			}
		})
	}
	t.Run("file target preserves registration", func(t *testing.T) {
		_, router, config, target := registeredMaquetteForDelete(t)
		mustWrite(t, target, "file")
		deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=true", http.StatusBadRequest)
		if _, _, err := lab.LoadRegisteredConfig(config.Name); err != nil {
			t.Fatalf("registration should remain: %v", err)
		}
	})
	t.Run("invalid query is rejected without deletion", func(t *testing.T) {
		_, router, config, target := registeredMaquetteForDelete(t)
		mustWrite(t, filepath.Join(target, "marker.txt"), "keep")
		deleteMaquetteRequest(t, router, "/api/v10-lab/maquettes/"+config.Name+"?deleteDirectory=nimportequoi", http.StatusBadRequest)
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("target should remain: %v", err)
		}
		if _, _, err := lab.LoadRegisteredConfig(config.Name); err != nil {
			t.Fatalf("registration should remain: %v", err)
		}
	})
}

func TestMaquetteCreateValidationErrorReturnsDetails(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.GedixConfig.Services = map[string]lab.ServiceDBConfig{
		"auth": {DBType: "postgres", DBDSN: "", ExtraKeys: map[string]string{}},
	}

	var response ExecutionResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes", config, &response, http.StatusBadRequest)
	assertValidationDsnResponse(t, response)
}

func TestMaquetteUpdateValidationErrorReturnsDetails(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)
	config.GedixConfig.Services = map[string]lab.ServiceDBConfig{
		"auth": {DBType: "postgres", DBDSN: "", ExtraKeys: map[string]string{}},
	}

	var response ExecutionResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/ticket-T5808", config, &response, http.StatusBadRequest, http.MethodPut)
	assertValidationDsnResponse(t, response)
}

func TestMaquetteRenameAndUnicodeName(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Name = "Gedix V10 Démo"
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)
	if _, err := os.Stat(filepath.Join(root, "modules", "v10-lab", "data", "maquettes", "Gedix_V10_Démo", "maquette.json")); err != nil {
		t.Fatalf("expected unicode registration path, got %v", err)
	}

	config.Name = "Gedix Démo"
	postJSON(t, router, http.MethodPut, "/api/v10-lab/maquettes/Gedix%20V10%20D%C3%A9mo", config, http.StatusOK)
	var summaries []MaquetteSummary
	getJSON(t, router, "/api/v10-lab/maquettes", &summaries, http.StatusOK)
	if len(summaries) != 1 || summaries[0].Name != "Gedix Démo" {
		t.Fatalf("unexpected summaries after rename: %#v", summaries)
	}
	if _, err := os.Stat(filepath.Join(root, "modules", "v10-lab", "data", "maquettes", "Gedix_Démo", "maquette.json")); err != nil {
		t.Fatalf("expected renamed unicode registration path, got %v", err)
	}
}

func TestMaquetteGroups(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	var group lab.MaquetteGroup
	postJSONInto(t, router, "/api/v10-lab/maquette-groups", MaquetteGroupRequest{Name: "Démo client"}, &group, http.StatusCreated)
	if group.Name != "Démo client" {
		t.Fatalf("unexpected group: %#v", group)
	}
	config := testConfig()
	config.Name = "Client A"
	config.GroupName = group.Name
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	var summaries []MaquetteSummary
	getJSON(t, router, "/api/v10-lab/maquettes", &summaries, http.StatusOK)
	if len(summaries) != 1 || summaries[0].GroupName != group.Name {
		t.Fatalf("expected grouped maquette, got %#v", summaries)
	}
	request := httptest.NewRequest(http.MethodDelete, "/api/v10-lab/maquette-groups/D%C3%A9mo%20client", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected non-empty group delete to fail, got %d body=%s", response.Code, response.Body.String())
	}
}

func TestImportJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Name = "Configuration source"
	config.Maquette.TargetPath = `D:\Chemin\Source`
	config.Release.ZipPath = `C:\Downloads\ancienne-release.zip`
	path := filepath.Join(t.TempDir(), "maquette.json")
	payload, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0600); err != nil {
		t.Fatal(err)
	}

	var preview ImportJSONPreviewResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/import-json/preview", ImportJSONPathRequest{Path: path}, &preview, http.StatusOK)
	if preview.Path != path || preview.Config.Name != config.Name {
		t.Fatalf("unexpected preview: %#v", preview)
	}

	var group lab.MaquetteGroup
	postJSONInto(t, router, "/api/v10-lab/maquette-groups", MaquetteGroupRequest{Name: "Production"}, &group, http.StatusCreated)
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes/import-json", ImportJSONRequest{Path: path, Name: "  Destination  ", GroupName: " production "}, http.StatusCreated)
	var imported lab.Config
	getJSON(t, router, "/api/v10-lab/maquettes/Destination", &imported, http.StatusOK)
	if imported.Name != "Destination" || imported.GroupName != group.Name || imported.GedixConfig.FQDN != config.GedixConfig.FQDN || imported.Maquette.AppName != config.Maquette.AppName || len(imported.Pipeline) != len(config.Pipeline) {
		t.Fatalf("import did not preserve and override expected fields: %#v", imported)
	}
	if imported.Maquette.TargetPath == config.Maquette.TargetPath || imported.Maquette.TargetPath != lab.DefaultMaquetteTargetPath(imported) {
		t.Fatalf("unexpected imported target path: %q", imported.Maquette.TargetPath)
	}
	if imported.Release.ZipPath != "" || imported.Release.Overwrite != config.Release.Overwrite {
		t.Fatalf("unexpected imported release: %#v", imported.Release)
	}

	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes/import-json", ImportJSONRequest{Path: path, Name: "destination"}, http.StatusConflict)
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes/import-json", ImportJSONRequest{Path: path, Name: "Other", GroupName: "missing"}, http.StatusUnprocessableEntity)
}

func TestImportJSONValidation(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)
	valid, err := json.Marshal(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for _, test := range []struct {
		name string
		path string
	}{
		{"missing file", filepath.Join(dir, "missing.json")},
		{"wrong extension", write("maquette.txt", valid)},
		{"malformed JSON", write("invalid.json", []byte("{"))},
		{"trailing document", write("trailing.json", append(valid, []byte(" {}")...))},
		{"unknown product", write("unknown-product.json", []byte(strings.Replace(string(valid), lab.GedixProdV10, "unknown-product", 1)))},
	} {
		t.Run(test.name, func(t *testing.T) {
			postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes/import-json/preview", ImportJSONPathRequest{Path: test.path}, http.StatusBadRequest)
		})
	}
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes/import-json", ImportJSONRequest{Path: write("valid.json", valid), Name: "   "}, http.StatusBadRequest)
}

func TestSavedActionPlansHTTP(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	var empty []lab.SavedActionPlan
	getJSON(t, router, "/api/v10-lab/action-plans?productId="+lab.GedixProdV10, &empty, http.StatusOK)
	if len(empty) != 0 {
		t.Fatalf("expected no saved plans, got %#v", empty)
	}

	var saved lab.SavedActionPlan
	postJSONInto(t, router, "/api/v10-lab/action-plans", SaveActionPlanRequest{
		Name:      "Initialisation",
		ProductID: lab.GedixProdV10,
		Actions:   []lab.PipelineStep{{Action: "create-workshop", Params: map[string]any{"entity_name": "Atelier"}}},
	}, &saved, http.StatusOK)
	if saved.ID == "" || len(saved.Actions) != 1 {
		t.Fatalf("unexpected saved action plan: %#v", saved)
	}

	postJSON(t, router, http.MethodPost, "/api/v10-lab/action-plans", SaveActionPlanRequest{Name: "Initialisation", ProductID: lab.GedixProdV10}, http.StatusBadRequest)
	postJSONInto(t, router, "/api/v10-lab/action-plans", SaveActionPlanRequest{Name: "Initialisation", ProductID: lab.GedixProdV10, Overwrite: true}, &saved, http.StatusOK)

	var items []lab.SavedActionPlan
	getJSON(t, router, "/api/v10-lab/action-plans?productId="+lab.GedixProdV10, &items, http.StatusOK)
	if len(items) != 1 || items[0].ID != saved.ID {
		t.Fatalf("unexpected saved plans: %#v", items)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v10-lab/action-plans/"+saved.ID, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestActionsByProduct(t *testing.T) {
	t.Setenv(toolboxruntime.EnvRoot, t.TempDir())
	router := mux.NewRouter()
	NewHandler().Register(router)

	var actions []lab.Action
	getJSON(t, router, "/api/v10-lab/actions?product="+lab.GedixProdV10, &actions, http.StatusOK)
	if len(actions) == 0 {
		t.Fatal("expected actions")
	}
	byID := map[string]bool{}
	for _, action := range actions {
		if action.Kind != lab.KindAPI {
			t.Fatalf("expected only API actions in pipeline builder, got %#v", action)
		}
		if action.Hidden {
			t.Fatalf("hidden action should not be returned to pipeline builder: %#v", action)
		}
		byID[action.ID] = true
	}
	for _, id := range []string{"create-plant", "create-workshop", "create-machine-group", "create-target", "create-machine", "create-machining-job-default-states", "create-presetting-program-default-states", "create-document-default-states", "create-machining-job"} {
		if !byID[id] {
			t.Fatalf("expected visible API action %s, got %#v", id, actions)
		}
	}
	if byID["create-cnc-folder"] || byID["stop-maquette"] || byID["stop-services"] {
		t.Fatalf("unexpected placeholder action, got %#v", actions)
	}
}

func TestAPIPipelineStepsDropsSystemActions(t *testing.T) {
	steps := apiPipelineSteps([]lab.PipelineStep{
		{Action: "create-env"},
		{Action: "create-workshop"},
		{Action: "configure-gedix-cfg"},
	}, lab.GedixProdV10)
	if len(steps) != 1 || steps[0].Action != "create-workshop" {
		t.Fatalf("expected only API steps, got %#v", steps)
	}
}

func TestRunMaquetteActionStartsSingleSystemAction(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	mustWrite(t, filepath.Join(root, "maquette", "gedix.cfg"), "fqdn=\"old\"\n# port=80\n")
	config.Maquette.TargetPath = filepath.Join(root, "maquette")
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	var started ExecutionResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/ticket-T5808/actions/configure-gedix-cfg/run", nil, &started, http.StatusAccepted)
	if started.Status != "running" {
		t.Fatalf("expected running action, got %#v", started)
	}
}

func TestRunModuleCommandRejectsUnclosedQuote(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	var started ExecutionResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/ticket-T5808/executable-command/run", ModuleCommandRunRequest{
		TargetKind: lab.ExecutableCommandTargetRoot,
		TargetName: "gx.exe",
		Command:    `status "unterminated`,
	}, &started, http.StatusAccepted)

	var current ExecutionResponse
	for index := 0; index < 20; index++ {
		getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808/run/current", &current, http.StatusOK)
		if !current.Running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != "failed" || len(current.Errors) == 0 || !strings.Contains(current.Errors[0], "guillemet non") {
		t.Fatalf("expected unclosed quote failure, got %#v", current)
	}
}

func TestCurrentRunTracksLiveLogsAndConflict(t *testing.T) {
	handler := NewHandler()
	run, ok := handler.acquireRun("ticket-T5808")
	if !ok {
		t.Fatal("expected run acquisition")
	}
	if _, ok := handler.acquireRun("ticket-T5808"); ok {
		t.Fatal("expected second run to be rejected")
	}

	writer := io.MultiWriter(currentRunWriter{run: run})
	if _, err := writer.Write([]byte("[INFO] step 1\n")); err != nil {
		t.Fatal(err)
	}
	snapshot := run.snapshot()
	if !snapshot.Running || snapshot.Status != "running" || !strings.Contains(snapshot.Log, "step 1") {
		t.Fatalf("unexpected running snapshot: %#v", snapshot)
	}

	run.finish("success", nil, 42)
	snapshot = run.snapshot()
	if snapshot.Running || snapshot.Status != "success" || snapshot.DurationMs != 42 {
		t.Fatalf("unexpected finished snapshot: %#v", snapshot)
	}
	if _, ok := handler.acquireRun("ticket-T5808"); !ok {
		t.Fatal("expected a new run after finish")
	}
}

func TestSelectReleasePathNonWindowsAndScanCfg(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	cfg := `[environments.demo.applications.prod.connectors.connector-filesystem01]
type="module-filesystem"
host="localhost"

[environments.demo.applications.prod.connectors.connector-dnc-01]
type = module-focas
`
	var scan ScanCfgResponse
	postMultipart(t, router, "/api/v10-lab/maquettes/ticket-T5808/scan-cfg", "gedix.cfg", []byte(cfg), nil, &scan, http.StatusOK)
	if scan.EnvName != "demo" || len(scan.Connectors) != 2 || scan.Connectors[0].Name != "connector-filesystem01" || scan.Connectors[0].Module != "filesystem" {
		t.Fatalf("unexpected scan response: %#v", scan)
	}
}

func TestScanCfgUsesProductUnitSection(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Name = "watch"
	config.Product = lab.GedixWatchV10
	config.Maquette.AppName = "watch"
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	cfg := `[environments.demo.applications.watch.agents.agent-watch-01]
type = "module-watch"
host = "localhost"
`
	var scan ScanCfgResponse
	postMultipart(t, router, "/api/v10-lab/maquettes/watch/scan-cfg", "gedix.cfg", []byte(cfg), nil, &scan, http.StatusOK)
	if scan.EnvName != "demo" || scan.UnitKind != "agent" || len(scan.Units) != 1 || scan.Units[0].Name != "agent-watch-01" || scan.Units[0].Module != "watch" {
		t.Fatalf("unexpected scan response: %#v", scan)
	}
}

func TestScanCfgReturnsLegacySecureConnectorsAndAdaptors(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Name = "legacy"
	config.Product = lab.GedixLegacySecure
	config.Maquette.AppName = "legacy_secure"
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	cfg := `[environments.demo.applications.legacy_secure.connectors.connector-digi-legacy-01]
type = "digi-legacy"

[environments.demo.applications.legacy_secure.adaptors.adaptor-digi-01]
type = "digi"
custom = "value"
`
	var scan ScanCfgResponse
	postMultipart(t, router, "/api/v10-lab/maquettes/legacy/scan-cfg", "gedix.cfg", []byte(cfg), map[string]string{"importExistingKeys": "true"}, &scan, http.StatusOK)
	if scan.EnvName != "demo" || len(scan.Connectors) != 1 || scan.Connectors[0].Module != "digi-legacy" {
		t.Fatalf("unexpected connector scan response: %#v", scan)
	}
	if len(scan.Adaptors) != 1 || scan.Adaptors[0].Name != "adaptor-digi-01" || scan.Adaptors[0].Module != "digi" || !strings.Contains(scan.Adaptors[0].RawConfig, `custom = "value"`) {
		t.Fatalf("unexpected adaptor scan response: %#v", scan)
	}
}

func TestScanCfgCanImportExistingRawKeys(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Pipeline = nil
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	cfg := `[environments.demo.applications.prod.connectors.connector-filesystem01]
# commented=true
type="module-filesystem"
host="localhost"

[environments.demo.applications.prod.connectors.connector-dnc-01]
type = module-focas
`
	var scan ScanCfgResponse
	postMultipart(t, router, "/api/v10-lab/maquettes/ticket-T5808/scan-cfg", "gedix.cfg", []byte(cfg), map[string]string{"importExistingKeys": "true"}, &scan, http.StatusOK)
	if len(scan.Connectors) != 2 || scan.Connectors[0].RawConfig != "host=\"localhost\"" {
		t.Fatalf("unexpected scan response: %#v", scan)
	}
}

func TestScanCfgImportsMultilineRawKeys(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	config := testConfig()
	config.Pipeline = nil
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	cfg := `[environments.live.applications.prod.connectors.heidenhain]
type="module-heidenhain"
heidenhain-tnccmd-host="192.168.85.177"
heidenhain-tnccmd-header-content="""
2  ;REF PROGRAME=${job.name}
3  ;INDICE      =${job.version}
4  ;ETAT        =${state.name}
5  ;DATE ETAT   =${program.created_at}
6  ;PROGRAMMEUR =${program.created_by}
7  ;TRANSFERE LE=${date.now_fr}
"""
heidenhain-tnccmd-header-first-line="1  ;********* ENTETE GEDIX *********"

[environments.live.applications.prod.connectors.next]
type="module-filesystem"
host="localhost"
`
	var scan ScanCfgResponse
	postMultipart(t, router, "/api/v10-lab/maquettes/ticket-T5808/scan-cfg", "gedix.cfg", []byte(cfg), map[string]string{"importExistingKeys": "true"}, &scan, http.StatusOK)
	if len(scan.Connectors) != 2 {
		t.Fatalf("unexpected connector count: %#v", scan)
	}
	raw := scan.Connectors[0].RawConfig
	expected := `heidenhain-tnccmd-host="192.168.85.177"
heidenhain-tnccmd-header-content="""
2  ;REF PROGRAME=${job.name}
3  ;INDICE      =${job.version}
4  ;ETAT        =${state.name}
5  ;DATE ETAT   =${program.created_at}
6  ;PROGRAMMEUR =${program.created_by}
7  ;TRANSFERE LE=${date.now_fr}
"""
heidenhain-tnccmd-header-first-line="1  ;********* ENTETE GEDIX *********"`
	if raw != expected {
		t.Fatalf("unexpected raw config:\n%s", raw)
	}
	if strings.Contains(raw, `type="module-heidenhain"`) {
		t.Fatalf("type should not be imported in raw config: %s", raw)
	}
}

func TestMaquetteOpenURLReadsNonCommentedCfgKeys(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	target := filepath.Join(root, "Gedix")
	mustWrite(t, filepath.Join(target, "gx-front.exe"), "")
	mustWrite(t, filepath.Join(target, "env_live", "app_prod", "gx-app.exe"), "")
	mustWrite(t, filepath.Join(target, "gedix.cfg"), "# fqdn=old\nfqdn=example.test\n# port=81\nport=8443\ntls=true\n")
	config := testConfig()
	config.Pipeline = nil
	config.Maquette.TargetPath = target
	config.Maquette.EnvName = "live"
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)

	var response MaquetteOpenURLResponse
	getJSON(t, router, "/api/v10-lab/maquettes/ticket-T5808/open-url", &response, http.StatusOK)
	if response.URL != "https://example.test:8443" {
		t.Fatalf("unexpected open URL: %#v", response)
	}
}

func TestOpenMaquetteTargetFolderValidation(t *testing.T) {
	if err := openMaquetteTargetFolder(lab.Config{}); err == nil || !strings.Contains(err.Error(), "repertoire cible") {
		t.Fatalf("expected missing target error, got %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	err := openMaquetteTargetFolder(lab.Config{Maquette: lab.MaquetteConfig{TargetPath: missing}})
	if err == nil || !strings.Contains(err.Error(), "introuvable") {
		t.Fatalf("expected missing folder error, got %v", err)
	}
	if runtime.GOOS != "windows" {
		err = openMaquetteTargetFolder(lab.Config{Maquette: lab.MaquetteConfig{TargetPath: t.TempDir()}})
		if err == nil || !strings.Contains(err.Error(), "uniquement disponible sous Windows") {
			t.Fatalf("expected non-windows error, got %v", err)
		}
	}
}

func TestImportExistingMaquettesSkipsKnownTargetAndDoesNotRecurse(t *testing.T) {
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)

	scanRoot := filepath.Join(root, "clients")
	maquetteRoot := filepath.Join(scanRoot, "Client", "Gedix")
	mustWrite(t, filepath.Join(maquetteRoot, "gx.exe"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "gx-front.exe"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "license.enc"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "license.key"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "env_live", "app_prod", "gx-app.exe"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "nested", "gx.exe"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "nested", "gx-front.exe"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "nested", "license.enc"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "nested", "license.key"), "")
	mustWrite(t, filepath.Join(maquetteRoot, "nested", "env_live", "app_prod", "gx-app.exe"), "")

	var first ImportExistingMaquettesResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/import-existing", ImportExistingMaquettesRequest{RootPath: scanRoot}, &first, http.StatusOK)
	if len(first.Imported) != 1 || first.Imported[0].Name != "Client" || first.Imported[0].TargetPath != maquetteRoot {
		t.Fatalf("unexpected import response: %#v", first)
	}

	var second ImportExistingMaquettesResponse
	postJSONInto(t, router, "/api/v10-lab/maquettes/import-existing", ImportExistingMaquettesRequest{RootPath: scanRoot}, &second, http.StatusOK)
	if len(second.Imported) != 0 || len(second.Skipped) != 1 {
		t.Fatalf("unexpected second import response: %#v", second)
	}
}

func testConfig() lab.Config {
	zipPath := filepath.Join(os.TempDir(), "v10-lab-test-release.zip")
	_ = os.WriteFile(zipPath, []byte("zip"), 0644)
	return lab.Config{
		Name:    "ticket-T5808",
		Product: lab.GedixProdV10,
		Release: lab.ReleaseConfig{
			ZipPath:   zipPath,
			Overwrite: false,
		},
		Maquette: lab.MaquetteConfig{
			AppName: "prod",
		},
		GedixConfig: lab.GedixConfig{
			FQDN:       "localhost",
			Port:       80,
			Services:   map[string]lab.ServiceDBConfig{},
			Connectors: map[string]lab.ConnectorConfig{},
		},
		Pipeline: []lab.PipelineStep{
			{Action: "configure-gedix-cfg", Label: "Configurer gedix.cfg", Params: map[string]any{}},
		},
	}
}

func registeredMaquetteForDelete(t *testing.T) (string, http.Handler, lab.Config, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv(toolboxruntime.EnvRoot, root)
	router := mux.NewRouter()
	NewHandler().Register(router)
	target := filepath.Join(root, "physical-maquette")
	config := testConfig()
	config.Maquette.TargetPath = target
	postJSON(t, router, http.MethodPost, "/api/v10-lab/maquettes", config, http.StatusCreated)
	return root, router, config, target
}

func deleteMaquetteRequest(t *testing.T, router http.Handler, path string, expectedStatus int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("DELETE %s status=%d body=%s", path, response.Code, response.Body.String())
	}
}

func assertMaquetteRegistrationDeleted(t *testing.T, name string) {
	t.Helper()
	if _, _, err := lab.LoadRegisteredConfig(name); !os.IsNotExist(err) {
		t.Fatalf("registration should be deleted, got %v", err)
	}
	if hasToken, err := lab.HasAPIToken(name); err != nil || hasToken {
		t.Fatalf("API token should be deleted, hasToken=%v err=%v", hasToken, err)
	}
}

func setRegisteredTargetPath(t *testing.T, name, target string) {
	t.Helper()
	config, path, err := lab.LoadRegisteredConfig(name)
	if err != nil {
		t.Fatal(err)
	}
	config.Maquette.TargetPath = target
	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func getJSON(t *testing.T, router http.Handler, path string, target any, expectedStatus int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func postJSON(t *testing.T, router http.Handler, method string, path string, body any, expectedStatus int) {
	t.Helper()
	postJSONInto(t, router, path, body, nil, expectedStatus, method)
}

func postJSONInto(t *testing.T, router http.Handler, path string, body any, target any, expectedStatus int, methods ...string) {
	t.Helper()
	method := http.MethodPost
	if len(methods) > 0 {
		method = methods[0]
	}
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &payload)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func postMultipart(t *testing.T, router http.Handler, path string, filename string, payload []byte, fields map[string]string, target any, expectedStatus int) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("POST %s status=%d body=%s", path, response.Code, response.Body.String())
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func assertValidationDsnResponse(t *testing.T, response ExecutionResponse) {
	t.Helper()
	if response.Status != "invalid" {
		t.Fatalf("expected invalid status, got %#v", response)
	}
	if len(response.Errors) == 0 {
		t.Fatalf("expected validation errors, got %#v", response)
	}
	message := strings.Join(response.Errors, "\n")
	if strings.TrimSpace(message) == "validation failed" {
		t.Fatalf("expected detailed validation error, got %#v", response.Errors)
	}
	for _, want := range []string{"Service \"auth\"", "DSN", "postgres"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in validation response: %#v", want, response.Errors)
		}
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
