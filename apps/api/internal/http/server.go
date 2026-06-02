package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"toolBox/apps/api/internal/config"
	"toolBox/apps/api/internal/jobs"
	"toolBox/apps/api/internal/modules"
	testsheetapi "toolBox/modules/test-sheet/pkg/httpapi"
	"toolBox/modules/test-sheet/pkg/repository"
	"toolBox/modules/test-sheet/pkg/service"
	v10labapi "toolBox/modules/v10-lab/pkg/httpapi"
	"toolBox/pkg/modulecontract"
	"toolBox/pkg/toolboxruntime"
	"toolBox/pkg/toolboxversion"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	registry      *modules.Registry
	jobs          *jobs.Store
	testSheetRepo *repository.SQLiteRepository
	runtime       toolboxruntime.Layout
}

func NewServer(runtimeLayout toolboxruntime.Layout) (*Server, error) {
	_ = os.Setenv(toolboxruntime.EnvRoot, runtimeLayout.RootDir)
	testSheetLayout := runtimeLayout.Module("test-sheet")
	v10LabLayout := runtimeLayout.Module("v10-lab")
	if err := testSheetLayout.EnsureBaseDirs(); err != nil {
		return nil, fmt.Errorf("init test-sheet runtime dirs: %w", err)
	}
	if err := v10LabLayout.EnsureBaseDirs(); err != nil {
		return nil, fmt.Errorf("init v10-lab runtime dirs: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(v10LabLayout.DataDir, "maquettes"), 0755); err != nil {
		return nil, fmt.Errorf("init v10-lab maquettes dir: %w", err)
	}
	if err := os.MkdirAll(testSheetFilesDocumentsDir(testSheetLayout), 0755); err != nil {
		return nil, fmt.Errorf("init test-sheet document dir: %w", err)
	}
	if err := os.MkdirAll(testSheetFilesRunsDir(testSheetLayout), 0755); err != nil {
		return nil, fmt.Errorf("init test-sheet runs dir: %w", err)
	}
	testSheetRepo, err := repository.Open(filepath.Join(testSheetLayout.DataDir, "test-sheet.db"))
	if err != nil {
		return nil, fmt.Errorf("init test-sheet database: %w", err)
	}
	return &Server{
		registry:      modules.NewRegistry(),
		jobs:          jobs.NewStore(),
		testSheetRepo: testSheetRepo,
		runtime:       runtimeLayout,
	}, nil
}

func ListenAndServe(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	runtimeLayout, err := toolboxruntime.ForApp(configPath)
	if err != nil {
		return err
	}
	server, err := NewServer(runtimeLayout)
	if err != nil {
		return err
	}
	handler := server.Routes()
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.WebOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})
	return http.ListenAndServe(cfg.Addr, c.Handler(handler))
}

func (s *Server) Routes() http.Handler {
	r := mux.NewRouter()
	testsheetapi.NewHandler(service.New(s.testSheetRepo)).Register(r)
	v10labapi.NewHandler().Register(r)
	r.HandleFunc("/api/health", s.health).Methods(http.MethodGet)
	r.HandleFunc("/api/version", s.version).Methods(http.MethodGet)
	r.HandleFunc("/api/modules", s.listModules).Methods(http.MethodGet)
	r.HandleFunc("/api/modules/{moduleId}", s.getModule).Methods(http.MethodGet)
	r.HandleFunc("/api/modules/{moduleId}/actions/{actionId}", s.runAction).Methods(http.MethodPost)
	r.HandleFunc("/api/jobs/{jobId}", s.getJob).Methods(http.MethodGet)
	return r
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]toolboxversion.VersionInfo{
		"api": toolboxversion.Info(toolboxversion.APIVersion),
	})
}

func (s *Server) listModules(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.List())
}

func (s *Server) getModule(w http.ResponseWriter, r *http.Request) {
	moduleID := mux.Vars(r)["moduleId"]
	module, ok := s.registry.Get(moduleID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "module not found"})
		return
	}
	writeJSON(w, http.StatusOK, module)
}

func (s *Server) runAction(w http.ResponseWriter, r *http.Request) {
	var request modulecontract.ActionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}

	vars := mux.Vars(r)
	moduleID := vars["moduleId"]
	actionID := vars["actionId"]
	if _, ok := s.registry.Get(moduleID); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "module not found"})
		return
	}

	response, err := s.runModuleAction(moduleID, actionID, request.Args)
	respondModuleAction(w, response, err)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	jobID := mux.Vars(r)["jobId"]
	job, ok := s.jobs.Get(jobID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) runModuleAction(moduleID, actionID string, args []string) (modulecontract.ActionResponse, error) {
	moduleLayout := s.runtime.Module(moduleID)
	commandArgs := append([]string{"run", actionID, "--json"}, args...)
	cmd := exec.Command(moduleLayout.Exe, commandArgs...)
	cmd.Dir = moduleLayout.Dir
	cmd.Env = append(os.Environ(), moduleLayout.Env()...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return modulecontract.ActionResponse{}, fmt.Errorf("run module %s action %s: %w: %s", moduleID, actionID, err, stderr.String())
	}

	var response modulecontract.ActionResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return modulecontract.ActionResponse{}, fmt.Errorf("decode module response: %w", err)
	}
	return response, nil
}

func respondModuleAction(w http.ResponseWriter, response modulecontract.ActionResponse, err error) {
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func testSheetFilesDocumentsDir(layout toolboxruntime.ModuleLayout) string {
	return filepath.Join(layout.FilesDir, "documents")
}

func testSheetFilesRunsDir(layout toolboxruntime.ModuleLayout) string {
	return filepath.Join(layout.FilesDir, "runs")
}
