package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"toolBox/modules/test-sheet/pkg/model"
	"toolBox/modules/test-sheet/pkg/service"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/test-sheet/plans", h.listPlans).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/plans/summary", h.listPlanSummaries).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/plans", h.createPlan).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/plans/{planId}", h.getPlan).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/plans/{planId}", h.updatePlan).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/plans/{planId}", h.deletePlan).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/plans/{planId}/permanent", h.permanentDeletePlan).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/plans/{planId}/restore", h.restorePlan).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/plans/{planId}/duplicate", h.duplicatePlan).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/plans/{planId}/documents", h.listDocuments).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/plans/{planId}/documents", h.uploadDocument).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/plans/{planId}/sheets", h.listSheets).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/plans/{planId}/sheets", h.createSheet).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}", h.updateSheet).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}", h.deleteSheet).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/duplicate", h.duplicateSheet).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/documents/{documentId}", h.linkSheetDocument).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/documents/{documentId}", h.unlinkSheetDocument).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/plans/{planId}/sheets/reorder", h.reorderSheets).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/steps", h.listSteps).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/steps", h.createStep).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/steps/{stepId}", h.updateStep).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/steps/{stepId}", h.deleteStep).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/steps/{stepId}/duplicate", h.duplicateStep).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/steps/{stepId}/documents/{documentId}", h.linkStepDocument).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/steps/{stepId}/documents/{documentId}", h.unlinkStepDocument).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/sheets/{sheetId}/steps/reorder", h.reorderSteps).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/documents/{documentId}/download", h.downloadDocument).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/documents/{documentId}", h.deleteDocument).Methods(http.MethodDelete)
	r.HandleFunc("/api/test-sheet/plans/{planId}/runs", h.createRun).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/plans/{planId}/runs", h.listPlanRuns).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/runs", h.listRunSummaries).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/runs/{runId}", h.getRun).Methods(http.MethodGet)
	r.HandleFunc("/api/test-sheet/runs/{runId}/replay", h.replayRun).Methods(http.MethodPost)
	r.HandleFunc("/api/test-sheet/runs/{runId}/cancel", h.cancelRun).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/runs/{runId}/sheets/{runSheetId}", h.updateRunSheet).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/runs/{runId}/steps/{runStepId}", h.updateRunStep).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/runs/{runId}/finish", h.finishRun).Methods(http.MethodPut)
	r.HandleFunc("/api/test-sheet/runs/{runId}/report", h.report).Methods(http.MethodGet)
}

func (h *Handler) listPlans(w http.ResponseWriter, _ *http.Request) {
	plans, err := h.service.ListPlans()
	respond(w, plans, err)
}

func (h *Handler) listPlanSummaries(w http.ResponseWriter, r *http.Request) {
	includeDeleted := r.URL.Query().Get("includeDeleted") == "true"
	summaries, err := h.service.ListPlanSummaries(includeDeleted)
	respond(w, summaries, err)
}

func (h *Handler) createPlan(w http.ResponseWriter, r *http.Request) {
	var input model.PlanInput
	if !decode(w, r, &input) {
		return
	}
	plan, err := h.service.CreatePlan(input)
	respondCreated(w, plan, err)
}

func (h *Handler) getPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	plan, err := h.service.GetPlan(id)
	respond(w, plan, err)
}

func (h *Handler) updatePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	var input model.PlanInput
	if !decode(w, r, &input) {
		return
	}
	plan, err := h.service.UpdatePlan(id, input)
	respond(w, plan, err)
}

func (h *Handler) deletePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.DeletePlan(id))
}

func (h *Handler) permanentDeletePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.PermanentDeletePlan(id))
}

func (h *Handler) restorePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	plan, err := h.service.RestorePlan(id)
	respond(w, plan, err)
}

func (h *Handler) duplicatePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	plan, err := h.service.DuplicatePlan(id)
	respondCreated(w, plan, err)
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	documents, err := h.service.ListDocuments(planID)
	respond(w, documents, err)
}

func (h *Handler) uploadDocument(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart body"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document file is required"})
		return
	}
	_ = file.Close()
	document, err := h.service.UploadDocument(planID, header, r.FormValue("description"))
	respondCreated(w, document, err)
}

func (h *Handler) listSheets(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	sheets, err := h.service.ListSheets(planID)
	respond(w, sheets, err)
}

func (h *Handler) createSheet(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	var input model.SheetInput
	if !decode(w, r, &input) {
		return
	}
	sheet, err := h.service.CreateSheet(planID, input)
	respondCreated(w, sheet, err)
}

func (h *Handler) updateSheet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	var input model.SheetInput
	if !decode(w, r, &input) {
		return
	}
	sheet, err := h.service.UpdateSheet(id, input)
	respond(w, sheet, err)
}

func (h *Handler) deleteSheet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.DeleteSheet(id))
}

func (h *Handler) duplicateSheet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	sheet, err := h.service.DuplicateSheet(id)
	respondCreated(w, sheet, err)
}

func (h *Handler) linkSheetDocument(w http.ResponseWriter, r *http.Request) {
	sheetID, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.LinkSheetDocument(sheetID, documentID))
}

func (h *Handler) unlinkSheetDocument(w http.ResponseWriter, r *http.Request) {
	sheetID, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.UnlinkSheetDocument(sheetID, documentID))
}

func (h *Handler) reorderSheets(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	var input model.ReorderInput
	if !decode(w, r, &input) {
		return
	}
	respondNoContent(w, h.service.ReorderSheets(planID, input.SheetIDs))
}

func (h *Handler) listSteps(w http.ResponseWriter, r *http.Request) {
	sheetID, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	steps, err := h.service.ListSteps(sheetID)
	respond(w, steps, err)
}

func (h *Handler) createStep(w http.ResponseWriter, r *http.Request) {
	sheetID, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	var input model.StepInput
	if !decode(w, r, &input) {
		return
	}
	step, err := h.service.CreateStep(sheetID, input)
	respondCreated(w, step, err)
}

func (h *Handler) updateStep(w http.ResponseWriter, r *http.Request) {
	stepID, ok := pathID(w, r, "stepId")
	if !ok {
		return
	}
	var input model.StepInput
	if !decode(w, r, &input) {
		return
	}
	step, err := h.service.UpdateStep(stepID, input)
	respond(w, step, err)
}

func (h *Handler) deleteStep(w http.ResponseWriter, r *http.Request) {
	stepID, ok := pathID(w, r, "stepId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.DeleteStep(stepID))
}

func (h *Handler) duplicateStep(w http.ResponseWriter, r *http.Request) {
	stepID, ok := pathID(w, r, "stepId")
	if !ok {
		return
	}
	step, err := h.service.DuplicateStep(stepID)
	respondCreated(w, step, err)
}

func (h *Handler) linkStepDocument(w http.ResponseWriter, r *http.Request) {
	stepID, ok := pathID(w, r, "stepId")
	if !ok {
		return
	}
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.LinkStepDocument(stepID, documentID))
}

func (h *Handler) unlinkStepDocument(w http.ResponseWriter, r *http.Request) {
	stepID, ok := pathID(w, r, "stepId")
	if !ok {
		return
	}
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.UnlinkStepDocument(stepID, documentID))
}

func (h *Handler) reorderSteps(w http.ResponseWriter, r *http.Request) {
	sheetID, ok := pathID(w, r, "sheetId")
	if !ok {
		return
	}
	var input model.ReorderInput
	if !decode(w, r, &input) {
		return
	}
	respondNoContent(w, h.service.ReorderSteps(sheetID, input.StepIDs))
}

func (h *Handler) downloadDocument(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	document, err := h.service.GetDocument(documentID)
	if err != nil {
		respondError(w, err)
		return
	}
	if document.MimeType != "" {
		w.Header().Set("Content-Type", document.MimeType)
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+url.QueryEscape(document.OriginalName)+`"`)
	http.ServeFile(w, r, document.StoragePath)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathID(w, r, "documentId")
	if !ok {
		return
	}
	respondNoContent(w, h.service.DeleteDocument(documentID))
}

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	run, err := h.service.CreateRun(planID)
	respondCreated(w, run, err)
}

func (h *Handler) listPlanRuns(w http.ResponseWriter, r *http.Request) {
	planID, ok := pathID(w, r, "planId")
	if !ok {
		return
	}
	runs, err := h.service.ListPlanRuns(planID)
	respond(w, runs, err)
}

func (h *Handler) listRunSummaries(w http.ResponseWriter, _ *http.Request) {
	runs, err := h.service.ListRunSummaries()
	respond(w, runs, err)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	run, err := h.service.GetRun(runID)
	respond(w, run, err)
}

func (h *Handler) replayRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	run, err := h.service.ReplayRun(runID)
	respondCreated(w, run, err)
}

func (h *Handler) archiveRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	run, err := h.service.ArchiveRun(runID)
	respond(w, run, err)
}

func (h *Handler) cancelRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	run, err := h.service.CancelRun(runID)
	respond(w, run, err)
}

func (h *Handler) updateRunSheet(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	runSheetID, ok := pathID(w, r, "runSheetId")
	if !ok {
		return
	}
	var input model.RunSheetResultInput
	if !decode(w, r, &input) {
		return
	}
	sheet, err := h.service.UpdateRunSheet(runID, runSheetID, input)
	respond(w, sheet, err)
}

func (h *Handler) updateRunStep(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	runStepID, ok := pathID(w, r, "runStepId")
	if !ok {
		return
	}
	var input model.RunStepResultInput
	if !decode(w, r, &input) {
		return
	}
	step, err := h.service.UpdateRunStep(runID, runStepID, input)
	respond(w, step, err)
}

func (h *Handler) finishRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	run, err := h.service.FinishRun(runID)
	respond(w, run, err)
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runId")
	if !ok {
		return
	}
	report, err := h.service.GenerateMarkdownReport(runID)
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report))
}

func decode(w http.ResponseWriter, r *http.Request, output any) bool {
	if r.Body == nil {
		return true
	}
	if err := json.NewDecoder(r.Body).Decode(output); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(mux.Vars(r)[name], 10, 64)
	if err != nil || value <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + name})
		return 0, false
	}
	return value, true
}

func respond(w http.ResponseWriter, payload any, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func respondCreated(w http.ResponseWriter, payload any, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, payload)
}

func respondNoContent(w http.ResponseWriter, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respondError(w http.ResponseWriter, err error) {
	if service.IsNotFound(err) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if service.IsConflict(err) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
