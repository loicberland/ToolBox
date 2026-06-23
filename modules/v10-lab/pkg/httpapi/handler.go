package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
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
	r.HandleFunc("/api/v10-lab/folders/select-path", h.selectFolderPath).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/import-json/select-path", h.selectImportJSONPath).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/action-plans", h.listActionPlans).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/action-plans", h.saveActionPlan).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/action-plans/{id}", h.deleteActionPlan).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes", h.listMaquettes).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes", h.createMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/import-existing", h.importExistingMaquettes).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/import-json/preview", h.previewImportJSON).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/import-json", h.importJSON).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/duplicate", h.duplicateMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquette-groups", h.listMaquetteGroups).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquette-groups", h.createMaquetteGroup).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquette-groups/{name}", h.updateMaquetteGroup).Methods(http.MethodPut)
	r.HandleFunc("/api/v10-lab/maquette-groups/{name}", h.deleteMaquetteGroup).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.getMaquette).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.updateMaquette).Methods(http.MethodPut)
	r.HandleFunc("/api/v10-lab/maquettes/{name}", h.deleteMaquette).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/validate", h.validateMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/api-token", h.getAPITokenStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/api-token", h.saveAPIToken).Methods(http.MethodPut)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/api-token", h.deleteAPIToken).Methods(http.MethodDelete)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/run", h.runMaquette).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/actions/{actionId}/run", h.runMaquetteAction).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/executable-command/run", h.runModuleCommand).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/module-command/run", h.runModuleCommand).Methods(http.MethodPost)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/open-url", h.maquetteOpenURL).Methods(http.MethodGet)
	r.HandleFunc("/api/v10-lab/maquettes/{name}/open-folder", h.openMaquetteFolder).Methods(http.MethodPost)
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
	GroupName    string  `json:"groupName,omitempty"`
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

type ImportExistingMaquettesRequest struct {
	RootPath string `json:"rootPath"`
}

type ImportExistingMaquettesResponse struct {
	Imported []MaquetteSummary `json:"imported"`
	Skipped  []string          `json:"skipped"`
	Warnings []string          `json:"warnings"`
}

type ImportJSONPathRequest struct {
	Path string `json:"path"`
}

type ImportJSONPreviewResponse struct {
	Path   string     `json:"path"`
	Config lab.Config `json:"config"`
}

type ImportJSONRequest struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	GroupName string `json:"groupName"`
}

type MaquetteOpenURLResponse struct {
	URL string `json:"url"`
}

type ScanUnit struct {
	Name      string `json:"name"`
	Module    string `json:"module"`
	RawConfig string `json:"rawConfig"`
}

type ScanCfgResponse struct {
	EnvName         string     `json:"envName"`
	AppName         string     `json:"appName"`
	UnitKind        string     `json:"unitKind"`
	UnitPluralLabel string     `json:"unitPluralLabel"`
	Units           []ScanUnit `json:"units"`
	Connectors      []ScanUnit `json:"connectors,omitempty"`
	Agents          []ScanUnit `json:"agents,omitempty"`
	Adaptors        []ScanUnit `json:"adaptors,omitempty"`
	Warnings        []string   `json:"warnings,omitempty"`
}

type ModuleCommandRunRequest struct {
	TargetKind lab.ExecutableCommandTargetKind `json:"targetKind"`
	TargetName string                          `json:"targetName"`
	Command    string                          `json:"command"`
	UnitName   string                          `json:"unitName,omitempty"`
}

type MaquetteGroupRequest struct {
	Name string `json:"name"`
}

type APITokenStatus struct {
	HasToken bool `json:"hasToken"`
}

type APITokenRequest struct {
	Token string `json:"token"`
}

type DuplicateMaquetteRequest struct {
	Name       string `json:"name"`
	ParentPath string `json:"parentPath"`
	CopyData   bool   `json:"copyData"`
}

type SaveActionPlanRequest struct {
	Name        string             `json:"name"`
	ProductID   string             `json:"productId,omitempty"`
	Description string             `json:"description,omitempty"`
	Actions     []lab.PipelineStep `json:"actions"`
	Overwrite   bool               `json:"overwrite,omitempty"`
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

func (h *Handler) selectFolderPath(w http.ResponseWriter, _ *http.Request) {
	if runtime.GOOS != "windows" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "la selection graphique de dossier est disponible uniquement sous Windows; saisissez le chemin manuellement"})
		return
	}
	path, cancelled, err := openWindowsFolderDialog()
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SelectReleasePathResponse{Path: path, Cancelled: cancelled})
}

func (h *Handler) selectImportJSONPath(w http.ResponseWriter, _ *http.Request) {
	if runtime.GOOS != "windows" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "la selection graphique de fichier est disponible uniquement sous Windows"})
		return
	}
	path, cancelled, err := openWindowsFileDialog("Fichiers JSON (*.json)|*.json", "Selectionner une configuration de maquette V10 Lab")
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

func (h *Handler) duplicateMaquette(w http.ResponseWriter, r *http.Request) {
	var request DuplicateMaquetteRequest
	if !decode(w, r, &request) {
		return
	}
	config, err := lab.DuplicateRegisteredMaquette(mux.Vars(r)["name"], lab.DuplicateMaquetteRequest{
		Name: request.Name, ParentPath: request.ParentPath, CopyData: request.CopyData,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, config)
}

func (h *Handler) importExistingMaquettes(w http.ResponseWriter, r *http.Request) {
	var request ImportExistingMaquettesRequest
	if !decode(w, r, &request) {
		return
	}
	result, err := importExistingMaquettes(request.RootPath)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) previewImportJSON(w http.ResponseWriter, r *http.Request) {
	var request ImportJSONPathRequest
	if !decode(w, r, &request) {
		return
	}
	config, err := readImportJSON(request.Path)
	if err != nil {
		respondImportJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ImportJSONPreviewResponse{Path: strings.TrimSpace(request.Path), Config: config})
}

func (h *Handler) importJSON(w http.ResponseWriter, r *http.Request) {
	var request ImportJSONRequest
	if !decode(w, r, &request) {
		return
	}
	config, err := readImportJSON(request.Path)
	if err != nil {
		respondImportJSONError(w, err)
		return
	}
	config.Name = strings.TrimSpace(request.Name)
	config.GroupName, err = canonicalImportGroupName(request.GroupName)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	config.Maquette.TargetPath = ""
	config.Release.ZipPath = ""
	lab.NormalizeConfigForSave(&config)
	if err := lab.ValidateConfig(config); err != nil {
		respondImportJSONError(w, err)
		return
	}
	if existing, found, err := registeredMaquetteByName(config.Name); err != nil {
		respondError(w, err)
		return
	} else if found {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("Une maquette portant le nom %q existe deja.", existing.Name)})
		return
	}
	item, err := lab.SaveRegisteredConfig(config)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) listMaquetteGroups(w http.ResponseWriter, _ *http.Request) {
	groups, err := lab.ListMaquetteGroups()
	respond(w, groups, err)
}

func (h *Handler) listActionPlans(w http.ResponseWriter, r *http.Request) {
	items, err := lab.ListSavedActionPlans(r.URL.Query().Get("productId"))
	respond(w, items, err)
}

func (h *Handler) saveActionPlan(w http.ResponseWriter, r *http.Request) {
	var request SaveActionPlanRequest
	if !decode(w, r, &request) {
		return
	}
	plan, err := lab.SaveActionPlan(lab.SaveActionPlanInput{
		Name:        request.Name,
		ProductID:   request.ProductID,
		Description: request.Description,
		Actions:     request.Actions,
		Overwrite:   request.Overwrite,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *Handler) deleteActionPlan(w http.ResponseWriter, r *http.Request) {
	if err := lab.DeleteSavedActionPlan(mux.Vars(r)["id"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createMaquetteGroup(w http.ResponseWriter, r *http.Request) {
	var request MaquetteGroupRequest
	if !decode(w, r, &request) {
		return
	}
	group, err := lab.CreateMaquetteGroup(request.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *Handler) updateMaquetteGroup(w http.ResponseWriter, r *http.Request) {
	var request MaquetteGroupRequest
	if !decode(w, r, &request) {
		return
	}
	group, err := lab.RenameMaquetteGroup(mux.Vars(r)["name"], request.Name)
	respond(w, group, err)
}

func (h *Handler) deleteMaquetteGroup(w http.ResponseWriter, r *http.Request) {
	if err := lab.DeleteMaquetteGroup(mux.Vars(r)["name"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if _, err := lab.SaveRegisteredConfigReplacing(name, config); err != nil {
		respondError(w, err)
		return
	}
	saved, _, err := lab.LoadRegisteredConfig(config.Name)
	respond(w, saved, err)
}

func (h *Handler) deleteMaquette(w http.ResponseWriter, r *http.Request) {
	deleteDirectory := false
	if raw := r.URL.Query().Get("deleteDirectory"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "deleteDirectory invalide"})
			return
		}
		deleteDirectory = parsed
	}
	if err := lab.DeleteRegisteredConfigWithDirectory(mux.Vars(r)["name"], deleteDirectory); err != nil {
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

func (h *Handler) getAPITokenStatus(w http.ResponseWriter, r *http.Request) {
	hasToken, err := lab.HasAPIToken(mux.Vars(r)["name"])
	respond(w, APITokenStatus{HasToken: hasToken}, err)
}

func (h *Handler) saveAPIToken(w http.ResponseWriter, r *http.Request) {
	var request APITokenRequest
	if !decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token API requis"})
		return
	}
	if err := lab.SaveAPIToken(mux.Vars(r)["name"], request.Token); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APITokenStatus{HasToken: true})
}

func (h *Handler) deleteAPIToken(w http.ResponseWriter, r *http.Request) {
	if err := lab.DeleteAPIToken(mux.Vars(r)["name"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) runModuleCommand(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var request ModuleCommandRunRequest
	if !decode(w, r, &request) {
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

	go h.executeModuleCommandRun(run, config, request)
	writeJSON(w, http.StatusAccepted, run.snapshot())
}

func (h *Handler) maquetteOpenURL(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	if err != nil {
		respondError(w, err)
		return
	}
	url, err := maquetteOpenURL(config)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, MaquetteOpenURLResponse{URL: url})
}

func (h *Handler) openMaquetteFolder(w http.ResponseWriter, r *http.Request) {
	config, _, err := lab.LoadRegisteredConfig(mux.Vars(r)["name"])
	if err != nil {
		respondError(w, err)
		return
	}
	if err := openMaquetteTargetFolder(config); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "opened"})
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
	product, err := lab.ProductDefinitionByID(config.Product)
	if err != nil {
		respondError(w, err)
		return
	}
	appName := firstNonEmpty(r.FormValue("appName"), config.Maquette.AppName, product.DefaultAppName, "prod")
	importExistingKeys := strings.EqualFold(r.FormValue("importExistingKeys"), "true")
	result, err := scanUnits(string(payload), envName, appName, product, importExistingKeys)
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force=true est requis pour couper les services GX"})
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
		GroupName:    config.GroupName,
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
	var conflictErr lab.DuplicateConflictError
	if errors.As(err, &conflictErr) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	var validationErr lab.ValidationError
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, ExecutionResponse{Status: "invalid", Errors: validationErr.Items})
		return
	}
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

func (h *Handler) executeModuleCommandRun(run *currentRun, config lab.Config, request ModuleCommandRunRequest) {
	startedAt := time.Now()
	targetKind := request.TargetKind
	targetName := request.TargetName
	if targetKind == "" && targetName == "" && request.UnitName != "" {
		targetKind = lab.ExecutableCommandTargetConnector
		targetName = request.UnitName
	}
	err := lab.RunExecutableCommand(config, lab.ExecutableCommandRequest{
		TargetKind: targetKind,
		TargetName: targetName,
		Command:    request.Command,
	}, io.MultiWriter(os.Stdout, currentRunWriter{run: run}))
	duration := time.Since(startedAt).Milliseconds()
	if err != nil {
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

func scanUnits(content string, envName string, appName string, product lab.ProductDefinition, importExistingKeys bool) (ScanCfgResponse, error) {
	envs := map[string]bool{}
	warnings := []string{}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	byKind := map[lab.UnitKind][]ScanUnit{}
	for _, definition := range product.UnitDefinitionsForProduct() {
		units, scanWarnings := scanUnitsForDefinition(lines, envName, appName, definition, envs, importExistingKeys)
		byKind[definition.Kind] = units
		warnings = append(warnings, scanWarnings...)
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
	response := ScanCfgResponse{
		EnvName:         envName,
		AppName:         appName,
		UnitKind:        string(product.UnitKind),
		UnitPluralLabel: product.UnitPluralLabel,
		Units:           byKind[product.PrimaryUnitDefinition().Kind],
		Connectors:      byKind[lab.UnitKindConnector],
		Agents:          byKind[lab.UnitKindAgent],
		Adaptors:        byKind[lab.UnitKindAdaptor],
		Warnings:        warnings,
	}
	return response, nil
}

func scanUnitsForDefinition(lines []string, envName string, appName string, definition lab.ProductUnitDefinition, envs map[string]bool, importExistingKeys bool) ([]ScanUnit, []string) {
	sectionPattern := regexp.MustCompile(fmt.Sprintf(`(?i)^\s*\[environments\.([^.]+)\.applications\.([^.]+)\.%s\.([^\]]+)\]\s*$`, regexp.QuoteMeta(definition.CfgSectionName)))
	units := []ScanUnit{}
	warnings := []string{}
	for index, line := range lines {
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
		module := scanUnitModule(lines, index+1)
		if module == "" {
			warnings = append(warnings, fmt.Sprintf("type absent pour %s %s", definition.SingularLabel, matches[3]))
		}
		rawConfig := ""
		if importExistingKeys {
			rawConfig = scanUnitRawConfig(lines, index+1)
		}
		units = append(units, ScanUnit{Name: matches[3], Module: module, RawConfig: rawConfig})
	}
	return units, warnings
}

func scanUnitRawConfig(lines []string, start int) string {
	items := []string{}
	current := []string{}
	inMultilineValue := false
	flushCurrent := func() {
		if len(current) == 0 {
			return
		}
		items = append(items, strings.Join(current, "\n"))
		current = []string{}
		inMultilineValue = false
	}
	for index := start; index < len(lines); index++ {
		line := strings.TrimRight(lines[index], "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			flushCurrent()
			break
		}
		key, ok := cfgKeyFromRawLine(line)
		if ok {
			flushCurrent()
			if !strings.EqualFold(key, "type") {
				current = append(current, line)
				inMultilineValue = hasOddTripleQuotes(line)
			}
			continue
		}
		if len(current) > 0 && (inMultilineValue || (trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, ";"))) {
			current = append(current, line)
			if hasOddTripleQuotes(line) {
				inMultilineValue = !inMultilineValue
			}
		}
	}
	flushCurrent()
	return strings.Join(items, "\n")
}

func hasOddTripleQuotes(line string) bool {
	return strings.Count(line, `"""`)%2 == 1
}

func cfgKeyFromRawLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(stripCfgComment(line))
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	index := strings.Index(trimmed, "=")
	if index <= 0 {
		return "", false
	}
	key := strings.TrimSpace(trimmed[:index])
	if key == "" || !isCfgRawKey(key) {
		return "", false
	}
	return key, true
}

func isCfgRawKey(key string) bool {
	for _, char := range key {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func scanUnitModule(lines []string, start int) string {
	for index := start; index < len(lines); index++ {
		line := strings.TrimSpace(stripCfgComment(lines[index]))
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			return ""
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := cfgKeyValue(line)
		if ok && strings.EqualFold(key, "type") {
			return lab.NormalizeModuleType(value)
		}
	}
	return ""
}

func cfgKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	value = strings.TrimSpace(value)
	return key, value, true
}

func maquetteOpenURL(config lab.Config) (string, error) {
	return lab.GedixWebBaseURL(config)
}

func openMaquetteTargetFolder(config lab.Config) error {
	targetDir := strings.TrimSpace(config.Maquette.TargetPath)
	if targetDir == "" {
		return fmt.Errorf("le repertoire cible de la maquette n'est pas renseigne")
	}
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("repertoire cible introuvable: %s", targetDir)
		}
		return fmt.Errorf("repertoire cible inaccessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("le repertoire cible n'est pas un dossier: %s", targetDir)
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("l'ouverture dans l'explorateur est uniquement disponible sous Windows")
	}
	return exec.Command("explorer.exe", targetDir).Start()
}

func importExistingMaquettes(rootPath string) (ImportExistingMaquettesResponse, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return ImportExistingMaquettesResponse{}, fmt.Errorf("rootPath requis")
	}
	rootPath = filepath.Clean(rootPath)
	info, err := os.Stat(rootPath)
	if err != nil {
		return ImportExistingMaquettesResponse{}, err
	}
	if !info.IsDir() {
		return ImportExistingMaquettesResponse{}, fmt.Errorf("rootPath doit être un dossier")
	}
	existingTargets, existingNames, err := registeredMaquetteIndexes()
	if err != nil {
		return ImportExistingMaquettesResponse{}, err
	}
	result := ImportExistingMaquettesResponse{Imported: []MaquetteSummary{}, Skipped: []string{}, Warnings: []string{}}
	err = filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", path, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		found, envName, appName, productID := detectExistingMaquette(path)
		if !found {
			return nil
		}
		key := cleanPathKey(path)
		if existingTargets[key] {
			result.Skipped = append(result.Skipped, path)
			result.Warnings = append(result.Warnings, fmt.Sprintf("Maquette déjà connue, chemin ignoré: %s", path))
			return filepath.SkipDir
		}
		name := uniqueMaquetteName(importedMaquetteBaseName(path), existingNames)
		config := lab.Config{
			Name:    name,
			Product: productID,
			Maquette: lab.MaquetteConfig{
				TargetPath: path,
				EnvName:    envName,
				AppName:    appName,
			},
			GedixConfig: lab.GedixConfig{
				Port:       80,
				Services:   map[string]lab.ServiceDBConfig{},
				Connectors: map[string]lab.ProductUnitConfig{},
				Agents:     map[string]lab.ProductUnitConfig{},
				Adaptors:   map[string]lab.ProductUnitConfig{},
			},
			Runtime:  lab.RuntimeConfig{OpenConsole: true},
			Pipeline: []lab.PipelineStep{},
		}
		if _, err := lab.SaveRegisteredConfig(config); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Import impossible pour %s: %v", path, err))
			return filepath.SkipDir
		}
		existingTargets[key] = true
		existingNames[strings.ToLower(name)] = true
		result.Imported = append(result.Imported, summaryFor(config))
		return filepath.SkipDir
	})
	if err != nil {
		return ImportExistingMaquettesResponse{}, err
	}
	return result, nil
}

func registeredMaquetteIndexes() (map[string]bool, map[string]bool, error) {
	items, err := lab.ListMaquettes()
	if err != nil {
		return nil, nil, err
	}
	targets := map[string]bool{}
	names := map[string]bool{}
	for _, item := range items {
		config, _, err := lab.LoadRegisteredConfig(item.Name)
		if err != nil {
			continue
		}
		targets[cleanPathKey(lab.ResolveMaquetteTargetPath(config))] = true
		names[strings.ToLower(config.Name)] = true
	}
	return targets, names, nil
}

func detectExistingMaquette(path string) (bool, string, string, string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, "", "", ""
	}
	hasGX := false
	hasFront := false
	hasEnc := false
	hasKey := false
	envs := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(strings.ToLower(name), "env_") {
				envs = append(envs, name)
			}
			continue
		}
		switch {
		case strings.EqualFold(name, "gx.exe"):
			hasGX = true
		case strings.EqualFold(name, "gx-front.exe"):
			hasFront = true
		case strings.EqualFold(filepath.Ext(name), ".enc"):
			hasEnc = true
		case strings.EqualFold(filepath.Ext(name), ".key"):
			hasKey = true
		}
	}
	if !hasGX || !hasFront || !hasEnc || !hasKey || len(envs) == 0 {
		return false, "", "", ""
	}
	sort.Strings(envs)
	envName := envs[0][len("env_"):]
	appName, productID := detectImportedProduct(filepath.Join(path, envs[0]))
	return true, envName, appName, productID
}

func detectImportedProduct(envPath string) (string, string) {
	apps := []string{}
	entries, err := os.ReadDir(envPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "app_") {
				apps = append(apps, entry.Name()[len("app_"):])
			}
		}
	}
	defaultProduct, _ := lab.ProductDefinitionByID(lab.GedixProdV10)
	defaultApp := firstNonEmpty(defaultProduct.DefaultAppName, "prod")
	if len(apps) != 1 {
		return defaultApp, lab.GedixProdV10
	}
	appName := apps[0]
	matches := []lab.ProductDefinition{}
	for _, product := range lab.Products() {
		if strings.EqualFold(product.DefaultAppName, appName) {
			matches = append(matches, product)
		}
	}
	if len(matches) == 1 {
		return appName, matches[0].ID
	}
	return appName, lab.GedixProdV10
}

func importedMaquetteBaseName(path string) string {
	base := filepath.Base(path)
	if strings.EqualFold(base, "Gedix") {
		parent := filepath.Base(filepath.Dir(path))
		if strings.TrimSpace(parent) != "" && parent != "." && parent != string(filepath.Separator) {
			return parent
		}
	}
	return base
}

func uniqueMaquetteName(base string, existing map[string]bool) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Maquette"
	}
	candidate := base
	for index := 2; existing[strings.ToLower(candidate)]; index++ {
		candidate = fmt.Sprintf("%s-%d", base, index)
	}
	return candidate
}

func cleanPathKey(path string) string {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err == nil {
		path = abs
	}
	return strings.ToLower(path)
}

func stripCfgComment(line string) string {
	inQuotes := false
	for index, char := range line {
		if char == '"' {
			inQuotes = !inQuotes
		}
		if !inQuotes && (char == '#' || char == ';') {
			return line[:index]
		}
	}
	return line
}

func openWindowsZipDialog() (string, bool, error) {
	return openWindowsFileDialog("Archives ZIP (*.zip)|*.zip", "Selectionner une release Gedix V10")
}

func openWindowsFileDialog(filter, title string) (string, bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Filter = ` + strconv.Quote(filter) + `
$dialog.Title = ` + strconv.Quote(title) + `
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

func readImportJSON(path string) (lab.Config, error) {
	path = strings.TrimSpace(path)
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return lab.Config{}, importJSONError{message: "Le fichier selectionne n'est pas un fichier JSON."}
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return lab.Config{}, importJSONError{message: "Le fichier JSON selectionne est introuvable."}
		}
		return lab.Config{}, fmt.Errorf("lecture du fichier JSON: %w", err)
	}
	if !info.Mode().IsRegular() {
		return lab.Config{}, importJSONError{message: "Le fichier selectionne n'est pas un fichier regulier."}
	}
	file, err := os.Open(path)
	if err != nil {
		return lab.Config{}, fmt.Errorf("lecture du fichier JSON: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var config lab.Config
	if err := decoder.Decode(&config); err != nil {
		return lab.Config{}, importJSONError{message: "Le fichier JSON est invalide."}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return lab.Config{}, importJSONError{message: "Le fichier JSON est invalide."}
	}
	if err := lab.ValidateConfig(config); err != nil {
		return lab.Config{}, importJSONError{message: importJSONValidationMessage(err)}
	}
	return config, nil
}

type importJSONError struct{ message string }

func (err importJSONError) Error() string { return err.message }

func importJSONValidationMessage(err error) string {
	if validation, ok := err.(lab.ValidationError); ok {
		for _, item := range validation.Items {
			if strings.HasPrefix(item, "product:") {
				return "Le produit indique dans le JSON n'est pas pris en charge."
			}
		}
	}
	return "Le fichier ne contient pas une configuration de maquette valide."
}

func respondImportJSONError(w http.ResponseWriter, err error) {
	var importErr importJSONError
	if errors.As(err, &importErr) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": importErr.message})
		return
	}
	respondError(w, err)
}

func canonicalImportGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	groups, err := lab.ListMaquetteGroups()
	if err != nil {
		return "", err
	}
	for _, group := range groups {
		if strings.EqualFold(normalizedImportGroupName(group.Name), normalizedImportGroupName(name)) {
			return group.Name, nil
		}
	}
	return "", fmt.Errorf("le groupe selectionne n'existe pas")
}

func normalizedImportGroupName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func registeredMaquetteByName(name string) (lab.RegisteredMaquette, bool, error) {
	items, err := lab.ListMaquettes()
	if err != nil {
		return lab.RegisteredMaquette{}, false, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			return item, true, nil
		}
	}
	return lab.RegisteredMaquette{}, false, nil
}

func openWindowsFolderDialog() (string, bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "Selectionner le repertoire racine des maquettes Gedix V10"
$dialog.ShowNewFolderButton = $false
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    Write-Output $dialog.SelectedPath
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
		if action.Kind != lab.KindAPI || action.Hidden {
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
	case "create-env", "configure-gedix-cfg", "start-maquette", "stop-maquette", "kill-gx-processes", "update-env", "start-services", "stop-services":
		return true
	default:
		return false
	}
}
