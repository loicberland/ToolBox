package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"toolBox/modules/v10-lab/internal/lab"

	"github.com/gorilla/mux"
)

type Handler struct {
	mu   sync.Mutex
	runs map[string]*currentRun
}

func NewHandler() *Handler {
	return &Handler{runs: map[string]*currentRun{}}
}

func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/v10-lab/products", h.products).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/actions", h.actions).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/db-templates", h.dbTemplates).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/default-target", h.defaultTarget).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/releases/select-path", h.selectReleasePath).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes", h.listMaquettes).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes", h.createMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.getMaquette).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.updateMaquette).Methods(http.MethodPut)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.deleteMaquette).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/validate", h.validateMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/run", h.runMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/actions/{actionId}/run", h.runMaquetteAction).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/run/current", h.currentRun).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/run/current/logs", h.currentRun).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/scan-cfg", h.scanCfg).Methods(http.MethodPost)
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
	Running    bool     `json:"running,omitempty"`
	Status     string   `json:"status"`
	Log        string   `json:"log,omitempty"`
	Output     string   `json:"output,omitempty"`
	Errors     []string `json:"errors,omitempty"`
	DurationMs int64    `json:"durationMs,omitempty"`
}

type currentRun struct {
	mu         sync.Mutex
	name       string
	running    bool
	status     string
	log        strings.Builder
	errors     []string
	startedAt  time.Time
	finishedAt time.Time
	durationMs int64
}

type currentRunWriter struct {
	run *currentRun
}

type LogSummary struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"`
}

type SelectReleasePathResponse struct {
	Path      string `json:"path,omitempty"`
	Cancelled bool   `json:"cancelled"`
}

type ScanConnector struct {
	Name      string `json:"name"`
	RawConfig string `json:"rawConfig"`
}

type ScanCfgResponse struct {
	EnvName    string          `json:"envName"`
	AppName    string          `json:"appName"`
	Connectors []ScanConnector `json:"connectors"`
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
	writeJSON(w, http.StatusOK, pipelineActions(actions))
}

func (h *Handler) dbTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, lab.DBTemplates())
}

func (h *Handler) defaultTarget(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	config := lab.Config{Name: name}
	writeJSON(w, http.StatusOK, map[string]string{"targetPath": lab.DefaultMaquetteTargetPath(config)})
}

func (h *Handler) selectReleasePath(w http.ResponseWriter, _ *http.Request) {
	if runtime.GOOS != "windows" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "la selection graphique de fichier est disponible uniquement sous Windows; saisissez le chemin manuellement"})
		return
	}
	path, cancelled, err := openWindowsZipDialog()
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SelectReleasePathResponse{Path: path, Cancelled: cancelled})
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
	if _, err := lab.SaveRegisteredConfig(config); err != nil {
		respondError(w, err)
		return
	}
	saved, _, err := lab.LoadRegisteredConfig(config.Name)
	respond(w, saved, err)
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
	name := mux.Vars(r)["name"]
	config, _, err := lab.LoadRegisteredConfig(name)
	if err != nil {
		respondError(w, err)
		return
	}
	config.Pipeline = apiPipelineSteps(config.Pipeline, config.Product)
	run, ok := h.acquireRun(name)
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Une execution est deja en cours pour cette maquette."})
		return
	}

	go h.executeRun(run, config)
	writeJSON(w, http.StatusAccepted, run.snapshot())
}

func (h *Handler) runMaquetteAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	actionID := vars["actionId"]
	if !isRunnableSystemAction(actionID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("action systeme non autorisee %q", actionID)})
		return
	}
	config, _, err := lab.LoadRegisteredConfig(name)
	if err != nil {
		respondError(w, err)
		return
	}
	run, ok := h.acquireRun(name)
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Une execution est deja en cours pour cette maquette."})
		return
	}

	go h.executeActionRun(run, config, actionID)
	writeJSON(w, http.StatusAccepted, run.snapshot())
}

func (h *Handler) currentRun(w http.ResponseWriter, r *http.Request) {
	run := h.getRun(mux.Vars(r)["name"])
	if run == nil {
		writeJSON(w, http.StatusOK, ExecutionResponse{Status: "idle", Running: false})
		return
	}
	writeJSON(w, http.StatusOK, run.snapshot())
}

func (h *Handler) scanCfg(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	if err != nil {
		respondError(w, err)
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fichier cfg requis"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fichier cfg requis"})
		return
	}
	defer file.Close()
	if !strings.EqualFold(filepath.Ext(header.Filename), ".cfg") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seuls les fichiers .cfg sont acceptes"})
		return
	}
	payload, err := io.ReadAll(io.LimitReader(file, 16<<20))
	if err != nil {
		respondError(w, err)
		return
	}
	envName := firstNonEmpty(r.FormValue("envName"), config.Maquette.EnvName)
	appName := firstNonEmpty(r.FormValue("appName"), config.Maquette.AppName, "prod")
	result, err := scanConnectors(string(payload), envName, appName)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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

func (h *Handler) acquireRun(name string) (*currentRun, bool) {
	key := safeName(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing := h.runs[key]; existing != nil && existing.isRunning() {
		return nil, false
	}
	run := &currentRun{name: name, running: true, status: "running", startedAt: time.Now()}
	h.runs[key] = run
	return run, true
}

func (h *Handler) getRun(name string) *currentRun {
	key := safeName(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runs[key]
}

func (h *Handler) executeRun(run *currentRun, config lab.Config) {
	startedAt := time.Now()
	err := lab.RunPipeline(context.Background(), config, io.MultiWriter(os.Stdout, currentRunWriter{run: run}))
	duration := time.Since(startedAt).Milliseconds()
	if err != nil {
		if validationErr, ok := err.(lab.ValidationError); ok {
			run.finish("failed", validationErr.Items, duration)
			return
		}
		run.finish("failed", []string{err.Error()}, duration)
		return
	}
	run.finish("success", nil, duration)
}

func (h *Handler) executeActionRun(run *currentRun, config lab.Config, actionID string) {
	startedAt := time.Now()
	err := lab.RunAction(context.Background(), config, actionID, io.MultiWriter(os.Stdout, currentRunWriter{run: run}))
	duration := time.Since(startedAt).Milliseconds()
	if err != nil {
		if validationErr, ok := err.(lab.ValidationError); ok {
			run.finish("failed", validationErr.Items, duration)
			return
		}
		run.finish("failed", []string{err.Error()}, duration)
		return
	}
	run.finish("success", nil, duration)
}

func (w currentRunWriter) Write(payload []byte) (int, error) {
	w.run.appendLog(string(payload))
	return len(payload), nil
}

func (r *currentRun) appendLog(value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.log.WriteString(value)
}

func (r *currentRun) isRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

func (r *currentRun) finish(status string, errors []string, durationMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
	r.status = status
	r.errors = errors
	r.durationMs = durationMs
	r.finishedAt = time.Now()
}

func (r *currentRun) snapshot() ExecutionResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	log := r.log.String()
	return ExecutionResponse{
		Running:    r.running,
		Status:     r.status,
		Log:        log,
		Output:     log,
		Errors:     append([]string{}, r.errors...),
		DurationMs: r.durationMs,
	}
}

func scanConnectors(content string, envName string, appName string) (ScanCfgResponse, error) {
	sectionPattern := regexp.MustCompile(`(?i)^\s*\[environments\.([^.]+)\.applications\.([^.]+)\.connectors\.([^\]]+)\]\s*$`)
	envs := map[string]bool{}
	connectors := []ScanConnector{}
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		envs[matches[1]] = true
		if envName != "" && !strings.EqualFold(matches[1], envName) {
			continue
		}
		if !strings.EqualFold(matches[2], appName) {
			continue
		}
		connectors = append(connectors, ScanConnector{Name: matches[3], RawConfig: ""})
	}
	if envName == "" {
		if len(envs) == 1 {
			for env := range envs {
				envName = env
			}
		} else if len(envs) > 1 {
			return ScanCfgResponse{}, fmt.Errorf("plusieurs environnements detectes, renseignez l'environnement")
		}
	}
	return ScanCfgResponse{EnvName: envName, AppName: appName, Connectors: connectors}, nil
}

func openWindowsZipDialog() (string, bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Filter = "Archives ZIP (*.zip)|*.zip"
$dialog.Title = "Selectionner une release Gedix V10"
$dialog.CheckFileExists = $true
$dialog.Multiselect = $false
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    Write-Output $dialog.FileName
}`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return "", false, err
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", true, nil
	}
	return path, false, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pipelineActions(actions []lab.Action) []lab.Action {
	items := []lab.Action{}
	for _, action := range actions {
		if action.Kind != lab.KindAPI {
			continue
		}
		items = append(items, action)
	}
	return items
}

func apiPipelineSteps(steps []lab.PipelineStep, product string) []lab.PipelineStep {
	items := []lab.PipelineStep{}
	for _, step := range steps {
		action, ok := lab.FindAction(step.Action)
		if !ok || action.Kind != lab.KindAPI || !action.SupportsProduct(product) {
			continue
		}
		items = append(items, step)
	}
	return items
}

func isRunnableSystemAction(actionID string) bool {
	switch actionID {
	case "create-env", "configure-gedix-cfg", "start-maquette", "update-env":
		return true
	default:
		return false
	}
}
