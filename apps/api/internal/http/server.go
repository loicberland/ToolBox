package http

import (
	"encoding/json"
	"net/http"

	"toolBox/apps/api/internal/config"
	"toolBox/apps/api/internal/jobs"
	"toolBox/apps/api/internal/modules"
	testsheetapi "toolBox/modules/test-sheet/pkg/httpapi"
	"toolBox/modules/test-sheet/pkg/repository"
	"toolBox/modules/test-sheet/pkg/service"
	"toolBox/pkg/modulecontract"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	registry *modules.Registry
	jobs     *jobs.Store
}

func NewServer() *Server {
	return &Server{
		registry: modules.NewRegistry(),
		jobs:     jobs.NewStore(),
	}
}

func ListenAndServe() error {
	cfg := config.Load()
	handler := NewServer().Routes()
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
	if repo, err := repository.Open(""); err == nil {
		testsheetapi.NewHandler(service.New(repo)).Register(r)
	}
	r.HandleFunc("/api/health", s.health).Methods(http.MethodGet)
	r.HandleFunc("/api/modules", s.listModules).Methods(http.MethodGet)
	r.HandleFunc("/api/modules/{moduleId}", s.getModule).Methods(http.MethodGet)
	r.HandleFunc("/api/modules/{moduleId}/actions/{actionId}", s.runAction).Methods(http.MethodPost)
	r.HandleFunc("/api/jobs/{jobId}", s.getJob).Methods(http.MethodGet)
	return r
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

	response := modulecontract.ActionResponse{
		ModuleID: moduleID,
		ActionID: actionID,
		Status:   "accepted",
		Output: map[string]any{
			"message": "action mock accepted",
			"args":    request.Args,
		},
	}
	writeJSON(w, http.StatusAccepted, response)
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
