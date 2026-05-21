package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"toolBox/modules/v10-lab/internal/lab"

	"github.com/gorilla/mux"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/v10-lab/products", h.products).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/actions", h.actions).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/db-templates", h.dbTemplates).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes", h.listMaquettes).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes", h.createMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.getMaquette).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.updateMaquette).Methods(http.MethodPut)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.deleteMaquette).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/validate", h.validateMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/run", h.runMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/logs", h.logs).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/logs/{logFile}", h.logFile).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/kill-gx-processes", h.killGXProcesses).Methods(http.MethodPost)
}

type MaquetteSummary struct {
	Name         string  `json:"name"`
	Product      string  `json:"product"`
	TargetPath   string  `json:"targetPath"`
	AppName      string  `json:"appName"`
	ExistsOnDisk bool    `json:"existsOnDisk"`
	LastRunAt    *string `json:"lastRunAt,omitempty"`
	LastStatus   *string `json:"lastStatus,omitempty"`
}

type ExecutionResponse struct {
	Status string   `json:"status"`
	Output string   `json:"output,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

type LogSummary struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"`
}

func (h *Handler) products(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, lab.Products())
}

func (h *Handler) actions(w http.ResponseWriter, r *http.Request) {
	product := r.URL.Query().Get("product")
	if product != "" && !lab.ProductExists(product) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("produit inconnu %q", product)})
		return
	}
	actions := lab.Actions()
	if product != "" {
		actions = lab.ActionsForProduct(product)
	}
	writeJSON(w, http.StatusOK, actions)
}

func (h *Handler) dbTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, lab.DBTemplates())
}

func (h *Handler) listMaquettes(w http.ResponseWriter, _ *http.Request) {
	items, err := lab.ListMaquettes()
	if err != nil {
		respondError(w, err)
		return
	}
	summaries := make([]MaquetteSummary, 0, len(items))
	for _, item := range items {
		config, _, err := lab.LoadRegisteredConfig(item.Name)
		if err != nil {
			continue
		}
		summaries = append(summaries, summaryFor(config))
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (h *Handler) createMaquette(w http.ResponseWriter, r *http.Request) {
	var config lab.Config
	if !decode(w, r, &config) {
		return
	}
	if _, _, err := lab.LoadRegisteredConfig(config.Name); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "maquette deja enregistree"})
		return
	}
	item, err := lab.SaveRegisteredConfig(config)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) getMaquette(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	respond(w, config, err)
}

func (h *Handler) updateMaquette(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var config lab.Config
	if !decode(w, r, &config) {
		return
	}
	if strings.TrimSpace(config.Name) == "" {
		config.Name = name
	}
	if !strings.EqualFold(safeName(config.Name), safeName(name)) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "le renommage de maquette n'est pas disponible dans cette phase"})
		return
	}
	item, err := lab.SaveRegisteredConfig(config)
	respond(w, item, err)
}

func (h *Handler) deleteMaquette(w http.ResponseWriter, r *http.Request) {
	if err := lab.DeleteRegisteredConfig(mux.Vars(r)["name"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) validateMaquette(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	if err != nil {
		respondError(w, err)
		return
	}
	if err := lab.ValidateConfig(config); err != nil {
		if validationErr, ok := err.(lab.ValidationError); ok {
			writeJSON(w, http.StatusBadRequest, ExecutionResponse{Status: "invalid", Errors: validationErr.Items})
			return
		}
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ExecutionResponse{Status: "valid", Output: "Validation OK"})
}

func (h *Handler) runMaquette(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	if err != nil {
		respondError(w, err)
		return
	}
	var output bytes.Buffer
	err = lab.RunPipeline(context.Background(), config, &output)
	if err != nil {
		if validationErr, ok := err.(lab.ValidationError); ok {
			writeJSON(w, http.StatusBadRequest, ExecutionResponse{Status: "failed", Output: output.String(), Errors: validationErr.Items})
			return
		}
		writeJSON(w, http.StatusBadRequest, ExecutionResponse{Status: "failed", Output: output.String(), Errors: []string{err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, ExecutionResponse{Status: "success", Output: output.String()})
}

func (h *Handler) logs(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(lab.RegisteredLogsDir(mux.Vars(r)["name"]))
	if os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, []LogSummary{})
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	logs := []LogSummary{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		logs = append(logs, LogSummary{Name: entry.Name(), SizeBytes: info.Size(), ModifiedAt: info.ModTime().Format(time.RFC3339)})
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ModifiedAt > logs[j].ModifiedAt
	})
	writeJSON(w, http.StatusOK, logs)
}

func (h *Handler) logFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := filepath.Base(vars["logFile"])
	if name != vars["logFile"] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nom de log invalide"})
		return
	}
	data, err := os.ReadFile(filepath.Join(lab.RegisteredLogsDir(vars["name"]), name))
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) killGXProcesses(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Force bool `json:"force"`
	}
	if !decode(w, r, &request) {
		return
	}
	if !request.Force {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force=true est requis pour taskkill gx-*"})
		return
	}
	var output bytes.Buffer
	if err := lab.KillGXProcesses(&output, true, false); err != nil {
		writeJSON(w, http.StatusBadRequest, ExecutionResponse{Status: "failed", Output: output.String(), Errors: []string{err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, ExecutionResponse{Status: "success", Output: output.String()})
}

func summaryFor(config lab.Config) MaquetteSummary {
	lab.ApplyDefaults(&config)
	target := lab.ResolveMaquetteTargetPath(config)
	_, statErr := os.Stat(target)
	lastRunAt, lastStatus := latestRunInfo(config.Name)
	return MaquetteSummary{
		Name:         config.Name,
		Product:      config.Product,
		TargetPath:   target,
		AppName:      config.Maquette.AppName,
		ExistsOnDisk: statErr == nil,
		LastRunAt:    lastRunAt,
		LastStatus:   lastStatus,
	}
}

func latestRunInfo(name string) (*string, *string) {
	entries, err := os.ReadDir(lab.RegisteredLogsDir(name))
	if err != nil {
		return nil, nil
	}
	var newest os.FileInfo
	var newestName string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if newest == nil || info.ModTime().After(newest.ModTime()) {
			newest = info
			newestName = entry.Name()
		}
	}
	if newest == nil {
		return nil, nil
	}
	runAt := newest.ModTime().Format(time.RFC3339)
	status := "unknown"
	if data, err := os.ReadFile(filepath.Join(lab.RegisteredLogsDir(name), newestName)); err == nil {
		text := string(data)
		if strings.Contains(text, "Execution terminee.") || strings.Contains(text, "Exécution terminée.") {
			status = "success"
		} else if strings.Contains(text, "Erreur:") || strings.Contains(text, "Erreur validation:") {
			status = "failed"
		}
	}
	return &runAt, &status
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body JSON requis"})
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body JSON invalide"})
		return false
	}
	return true
}

func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func respondError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if os.IsNotExist(err) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func safeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
