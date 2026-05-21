package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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

	request := httptest.NewRequest(http.MethodDelete, "/api/v10-lab/maquettes/ticket-T5808", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "modules", "v10-lab", "data", "maquettes", "ticket-T5808", "maquette.json")); !os.IsNotExist(err) {
		t.Fatalf("expected registration json to be removed, err=%v", err)
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
}

func testConfig() lab.Config {
	return lab.Config{
		Name:    "ticket-T5808",
		Product: lab.GedixProdV10,
		Release: lab.ReleaseConfig{
			ZipPath:   "D:/release.zip",
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
