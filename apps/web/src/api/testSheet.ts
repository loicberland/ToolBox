import { API_BASE_URL } from '../config/apiConfig';

export type TestPlan = {
  id: number;
  name: string;
  description: string;
  mockupSettings: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
};

export type TestSheet = {
  id: number;
  planId: number;
  name: string;
  description: string;
  prerequisites: string;
  config: string;
  command: string;
  notes: string;
  action: string;
  expectedResult: string;
  executionOrder: number;
  mockupSettings: string;
  steps?: TestSheetStep[];
  documents?: TestDocument[];
};

export type TestSheetStep = {
  id: number;
  sheetId: number;
  action: string;
  field: string;
  expectedResult: string;
  executionOrder: number;
  documents?: TestDocument[];
};

export type TestDocument = {
  id: number;
  planId: number;
  originalName: string;
  storedName: string;
  mimeType: string;
  sizeBytes: number;
  sha256: string;
  description: string;
  createdAt: string;
};

export type Evidence = {
  id: number;
  runSheetId: number;
  name: string;
  mimeType: string;
  sizeBytes: number;
  comment: string;
  createdAt: string;
};

export type TestRun = {
  id: number;
  runNumber: number;
  planId: number;
  planName: string;
  status: TestRunStatus | string;
  startedAt: string;
  finishedAt?: string;
  sheets: TestRunSheet[];
};

export type TestRunStatus = 'pending' | 'running' | 'completed' | 'canceled' | 'archived';

export type TestRunSummary = {
  id: number;
  runNumber: number;
  planId: number;
  planName: string;
  status: TestRunStatus;
  startedAt: string;
  finishedAt?: string;
  totalSheets: number;
  totalSteps: number;
  pendingSteps: number;
  passedSteps: number;
  failedSteps: number;
  blockedSteps: number;
  skippedSteps: number;
};

export type TestPlanSummary = {
  id: number;
  name: string;
  description: string;
  status: TestRunStatus | 'pending';
  sheetCount: number;
  runCount: number;
  latestRun?: TestRunSummary;
  updatedAt: string;
  deletedAt?: string;
};

export type TestRunSheet = {
  id: number;
  runId: number;
  sourceSheetId?: number;
  name: string;
  description: string;
  prerequisites: string;
  config: string;
  command: string;
  notes: string;
  action: string;
  expectedResult: string;
  executionOrder: number;
  status: 'pending' | 'passed' | 'failed' | 'blocked' | 'skipped';
  actualResult: string;
  comment: string;
  steps?: TestRunStep[];
  evidences?: Evidence[];
  documents?: TestDocument[];
};

export type TestRunStep = {
  id: number;
  runSheetId: number;
  sourceStepId?: number;
  action: string;
  field: string;
  expectedResult: string;
  executionOrder: number;
  status: 'pending' | 'passed' | 'failed' | 'blocked' | 'skipped';
  actualResult: string;
  comment: string;
  documents?: TestDocument[];
};

export type PlanInput = Pick<TestPlan, 'name' | 'description' | 'mockupSettings'>;
export type SheetInput = Omit<TestSheet, 'id' | 'planId'>;
export type RunSheetInput = Pick<TestRunSheet, 'status' | 'actualResult' | 'comment'>;
export type StepInput = Omit<TestSheetStep, 'id' | 'sheetId'>;
export type RunStepInput = Pick<TestRunStep, 'status' | 'actualResult' | 'comment'>;

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
  });
  if (!response.ok) {
    let message = `Erreur HTTP: ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error ?? message;
    } catch {
      // Keep generic message for non-JSON errors.
    }
    throw new Error(message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json();
}

export const testSheetApi = {
  listPlans: () => request<TestPlan[]>('/test-sheet/plans'),
  listPlanSummaries: (includeDeleted = false) => request<TestPlanSummary[]>(`/test-sheet/plans/summary${includeDeleted ? '?includeDeleted=true' : ''}`),
  createPlan: (input: PlanInput) => request<TestPlan>('/test-sheet/plans', jsonRequest('POST', input)),
  getPlan: (planId: number) => request<TestPlan>(`/test-sheet/plans/${planId}`),
  updatePlan: (planId: number, input: PlanInput) => request<TestPlan>(`/test-sheet/plans/${planId}`, jsonRequest('PUT', input)),
  deletePlan: (planId: number) => request<void>(`/test-sheet/plans/${planId}`, { method: 'DELETE' }),
  permanentDeletePlan: (planId: number) => request<void>(`/test-sheet/plans/${planId}/permanent`, { method: 'DELETE' }),
  restorePlan: (planId: number) => request<TestPlan>(`/test-sheet/plans/${planId}/restore`, { method: 'PUT' }),
  duplicatePlan: (planId: number) => request<TestPlan>(`/test-sheet/plans/${planId}/duplicate`, { method: 'POST' }),
  listSheets: (planId: number) => request<TestSheet[]>(`/test-sheet/plans/${planId}/sheets`),
  createSheet: (planId: number, input: SheetInput) => request<TestSheet>(`/test-sheet/plans/${planId}/sheets`, jsonRequest('POST', input)),
  updateSheet: (sheetId: number, input: SheetInput) => request<TestSheet>(`/test-sheet/sheets/${sheetId}`, jsonRequest('PUT', input)),
  deleteSheet: (sheetId: number) => request<void>(`/test-sheet/sheets/${sheetId}`, { method: 'DELETE' }),
  duplicateSheet: (sheetId: number) => request<TestSheet>(`/test-sheet/sheets/${sheetId}/duplicate`, { method: 'POST' }),
  reorderSheets: (planId: number, sheetIds: number[]) => request<void>(`/test-sheet/plans/${planId}/sheets/reorder`, jsonRequest('PUT', { sheetIds })),
  listDocuments: (planId: number) => request<TestDocument[]>(`/test-sheet/plans/${planId}/documents`),
  uploadDocument: (planId: number, file: File, description = '') => uploadDocument(planId, file, description),
  deleteDocument: (documentId: number) => request<void>(`/test-sheet/documents/${documentId}`, { method: 'DELETE' }),
  linkSheetDocument: (sheetId: number, documentId: number) => request<void>(`/test-sheet/sheets/${sheetId}/documents/${documentId}`, { method: 'POST' }),
  unlinkSheetDocument: (sheetId: number, documentId: number) => request<void>(`/test-sheet/sheets/${sheetId}/documents/${documentId}`, { method: 'DELETE' }),
  linkStepDocument: (stepId: number, documentId: number) => request<void>(`/test-sheet/steps/${stepId}/documents/${documentId}`, { method: 'POST' }),
  unlinkStepDocument: (stepId: number, documentId: number) => request<void>(`/test-sheet/steps/${stepId}/documents/${documentId}`, { method: 'DELETE' }),
  documentDownloadUrl: (documentId: number) => `${API_BASE_URL}/test-sheet/documents/${documentId}/download`,
  listSteps: (sheetId: number) => request<TestSheetStep[]>(`/test-sheet/sheets/${sheetId}/steps`),
  createStep: (sheetId: number, input: StepInput) => request<TestSheetStep>(`/test-sheet/sheets/${sheetId}/steps`, jsonRequest('POST', input)),
  updateStep: (stepId: number, input: StepInput) => request<TestSheetStep>(`/test-sheet/steps/${stepId}`, jsonRequest('PUT', input)),
  deleteStep: (stepId: number) => request<void>(`/test-sheet/steps/${stepId}`, { method: 'DELETE' }),
  duplicateStep: (stepId: number) => request<TestSheetStep>(`/test-sheet/steps/${stepId}/duplicate`, { method: 'POST' }),
  reorderSteps: (sheetId: number, stepIds: number[]) => request<void>(`/test-sheet/sheets/${sheetId}/steps/reorder`, jsonRequest('PUT', { stepIds })),
  createRun: (planId: number) => request<TestRun>(`/test-sheet/plans/${planId}/runs`, { method: 'POST' }),
  listPlanRuns: (planId: number) => request<TestRunSummary[]>(`/test-sheet/plans/${planId}/runs`),
  listRunSummaries: () => request<TestRunSummary[]>('/test-sheet/runs'),
  getRun: (runId: number) => request<TestRun>(`/test-sheet/runs/${runId}`),
  replayRun: (runId: number) => request<TestRun>(`/test-sheet/runs/${runId}/replay`, { method: 'POST' }),
  cancelRun: (runId: number) => request<TestRun>(`/test-sheet/runs/${runId}/cancel`, { method: 'PUT' }),
  updateRunSheet: (runId: number, runSheetId: number, input: RunSheetInput) =>
    request<TestRunSheet>(`/test-sheet/runs/${runId}/sheets/${runSheetId}`, jsonRequest('PUT', input)),
  listRunSheetEvidences: (runId: number, runSheetId: number) =>
    request<Evidence[]>(`/test-sheet/runs/${runId}/sheets/${runSheetId}/evidences`),
  uploadRunSheetEvidence: (runId: number, runSheetId: number, file: File) =>
    uploadRunSheetEvidence(runId, runSheetId, file),
  evidenceDownloadUrl: (evidenceId: number) => `${API_BASE_URL}/test-sheet/evidences/${evidenceId}/download`,
  deleteEvidence: (evidenceId: number) => request<void>(`/test-sheet/evidences/${evidenceId}`, { method: 'DELETE' }),
  updateRunStep: (runId: number, runStepId: number, input: RunStepInput) =>
    request<TestRunStep>(`/test-sheet/runs/${runId}/steps/${runStepId}`, jsonRequest('PUT', input)),
  finishRun: (runId: number) => request<TestRun>(`/test-sheet/runs/${runId}/finish`, { method: 'PUT' }),
  getReport: async (runId: number) => {
    const response = await fetch(`${API_BASE_URL}/test-sheet/runs/${runId}/report`);
    if (!response.ok) {
      throw new Error(`Erreur HTTP: ${response.status}`);
    }
    return response.text();
  },
};

async function uploadDocument(planId: number, file: File, description: string): Promise<TestDocument> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('description', description);
  const response = await fetch(`${API_BASE_URL}/test-sheet/plans/${planId}/documents`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    let message = `Erreur HTTP: ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error ?? message;
    } catch {
      // Keep generic message for non-JSON errors.
    }
    throw new Error(message);
  }
  return response.json();
}

async function uploadRunSheetEvidence(runId: number, runSheetId: number, file: File): Promise<Evidence> {
  const formData = new FormData();
  formData.append('file', file);
  const response = await fetch(`${API_BASE_URL}/test-sheet/runs/${runId}/sheets/${runSheetId}/evidences`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    let message = `Erreur HTTP: ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error ?? message;
    } catch {
      // Keep generic message for non-JSON errors.
    }
    throw new Error(message);
  }
  return response.json();
}

function jsonRequest(method: string, body: unknown): RequestInit {
  return {
    method,
    body: JSON.stringify(body),
  };
}
