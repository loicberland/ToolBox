import { API_BASE_URL } from '../config/apiConfig';

export const DEFAULT_V10_PRODUCT_ID = 'gedix-prod-v10';

export type UnitKind = 'connector' | 'agent' | 'adaptor' | '';
export type ExecutableCommandTargetKind = 'root' | 'service' | 'connector' | 'agent' | 'adaptor';

export type V10UnitDefinition = {
  kind: UnitKind;
  singularLabel: string;
  pluralLabel: string;
  cfgSectionName: string;
  folderPrefix: string;
  runtimeExecutablePattern?: string;
  moduleExecutablePattern?: string;
};

export type V10Product = {
  id: string;
  name: string;
  label: string;
  description: string;
  defaultAppName: string;
  services: Array<{
    name: string;
    label: string;
    hasDatabase: boolean;
    supportsExtraKeys: boolean;
  }>;
  unitKind?: UnitKind;
  unitSingularLabel: string;
  unitPluralLabel: string;
  unitCfgSectionName: string;
  unitFolderPrefix: string;
  unitExecutableName?: string;
  unitRuntimeExecutablePattern?: string;
  unitModuleExecutablePattern?: string;
  unitDefinitions?: V10UnitDefinition[];
};

export type V10ActionField = {
  name: string;
  label: string;
  type: string;
  required: boolean;
  default: unknown;
  description: string;
  options?: Array<{ label: string; value: string }>;
  optionsSource?: string;
  hiddenWhen?: Record<string, unknown>;
  hiddenWhenAny?: Array<Record<string, unknown>>;
  itemFields?: V10ActionField[];
  uniqueItemField?: string;
  min?: number;
  itemMin?: number;
  multiple?: boolean;
};

export type V10Action = {
  id: string;
  label: string;
  description: string;
  kind: string;
  products: string[];
  fields: V10ActionField[];
};

export type DBTemplate = {
  type: string;
  template: string;
};

export type ServiceDBConfig = {
  dbType: string;
  dbDsn: string;
  extraKeys?: Record<string, string>;
};

export type ConnectorConfig = {
  module?: string;
  rawConfig: string;
};

export type PipelineStep = {
  action: string;
  label: string;
  params: Record<string, unknown>;
};

export type V10Config = {
  name: string;
  product: string;
  release: {
    zipPath: string;
    workDir: string;
    overwrite: boolean;
    sourcePath?: string;
    targetPath?: string;
  };
  maquette: {
    targetPath: string;
    envName: string;
    appName: string;
  };
  gedixConfig: {
    fqdn: string;
    port: number;
    services: Record<string, ServiceDBConfig>;
    connectors: Record<string, ConnectorConfig>;
    agents?: Record<string, ConnectorConfig>;
    adaptors?: Record<string, ConnectorConfig>;
    units?: Record<string, ConnectorConfig>;
  };
  runtime: {
    debugTargets: string[];
    debugTargetFlags?: Record<string, string[]>;
    openConsole: boolean;
  };
  groupName?: string;
  api?: {
    baseUrl: string;
    tokenRef: string;
  };
  database?: unknown;
  services?: unknown[];
  pipeline: PipelineStep[];
};

export type MaquetteSummary = {
  name: string;
  product: string;
  targetPath: string;
  appName: string;
  existsOnDisk: boolean;
  lastRunAt?: string;
  lastStatus?: string;
  groupName?: string;
};

export type MaquetteGroup = {
  name: string;
};

export type DuplicateMaquetteRequest = {
  name: string;
  parentPath: string;
  copyData: boolean;
};

export type ExecutionResponse = {
  running?: boolean;
  status: string;
  log?: string;
  output?: string;
  errors?: string[];
  durationMs?: number;
};

export type LogSummary = {
  name: string;
  sizeBytes: number;
  modifiedAt: string;
};

export type ApiTokenStatus = {
  hasToken: boolean;
};

export type SelectReleasePathResponse = {
  path?: string;
  cancelled: boolean;
};

export type ImportExistingMaquettesResponse = {
  imported: MaquetteSummary[];
  skipped: string[];
  warnings: string[];
};

export type ImportJSONPreviewResponse = {
  path: string;
  config: V10Config;
};

export type ScanCfgResponse = {
  envName: string;
  appName: string;
  unitKind?: UnitKind;
  unitPluralLabel?: string;
  units?: Array<{ name: string; module?: string; rawConfig: string }>;
  connectors?: Array<{ name: string; module?: string; rawConfig: string }>;
  agents?: Array<{ name: string; module?: string; rawConfig: string }>;
  adaptors?: Array<{ name: string; module?: string; rawConfig: string }>;
  warnings?: string[];
};

export type V10SavedActionPlan = {
  id: string;
  name: string;
  productId?: string;
  description?: string;
  actions: PipelineStep[];
  createdAt: string;
  updatedAt: string;
};

export type SaveActionPlanPayload = {
  name: string;
  productId?: string;
  description?: string;
  actions: PipelineStep[];
  overwrite?: boolean;
};

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
      if (Array.isArray(payload.errors) && payload.errors.length > 0) {
        message = payload.errors.join('\n');
      } else {
        message = payload.error ?? message;
      }
    } catch {
      // Keep generic message.
    }
    throw new Error(message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json();
}

function jsonRequest(method: string, body: unknown): RequestInit {
  return {
    method,
    body: JSON.stringify(body),
  };
}

export const v10LabApi = {
  products: () => request<V10Product[]>('/v10-lab/products'),
  actions: (product: string) => request<V10Action[]>(`/v10-lab/actions?product=${encodeURIComponent(product)}`),
  dbTemplates: () => request<DBTemplate[]>('/v10-lab/db-templates'),
  defaultTarget: (name: string) => request<{ targetPath: string }>(`/v10-lab/default-target?name=${encodeURIComponent(name)}`),
  selectReleasePath: () => request<SelectReleasePathResponse>('/v10-lab/releases/select-path', { method: 'POST' }),
  selectFolderPath: () => request<SelectReleasePathResponse>('/v10-lab/folders/select-path', { method: 'POST' }),

  selectImportJSONPath: () => request<SelectReleasePathResponse>('/v10-lab/maquettes/import-json/select-path', { method: 'POST' }),
  listMaquettes: () => request<MaquetteSummary[]>('/v10-lab/maquettes'),
  listSavedActionPlans: (productId?: string) => request<V10SavedActionPlan[]>(`/v10-lab/action-plans${productId ? `?productId=${encodeURIComponent(productId)}` : ''}`),
  saveActionPlan: (payload: SaveActionPlanPayload) => request<V10SavedActionPlan>('/v10-lab/action-plans', jsonRequest('POST', payload)),
  deleteSavedActionPlan: (id: string) => request<void>(`/v10-lab/action-plans/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listMaquetteGroups: () => request<MaquetteGroup[]>('/v10-lab/maquette-groups'),
  createMaquetteGroup: (name: string) => request<MaquetteGroup>('/v10-lab/maquette-groups', jsonRequest('POST', { name })),
  updateMaquetteGroup: (name: string, nextName: string) => request<MaquetteGroup>(`/v10-lab/maquette-groups/${encodeURIComponent(name)}`, jsonRequest('PUT', { name: nextName })),
  deleteMaquetteGroup: (name: string) => request<void>(`/v10-lab/maquette-groups/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  createMaquette: (config: V10Config) => request('/v10-lab/maquettes', jsonRequest('POST', config)),
  importExistingMaquettes: (rootPath: string) => request<ImportExistingMaquettesResponse>('/v10-lab/maquettes/import-existing', jsonRequest('POST', { rootPath })),
  previewImportJSON: (path: string) => request<ImportJSONPreviewResponse>('/v10-lab/maquettes/import-json/preview', jsonRequest('POST', { path })),
  importJSON: (path: string, name: string, groupName: string) => request<void>('/v10-lab/maquettes/import-json', jsonRequest('POST', { path, name, groupName })),
  getMaquette: (name: string) => request<V10Config>(`/v10-lab/maquettes/${encodeURIComponent(name)}`),
  updateMaquette: (name: string, config: V10Config) => request<V10Config>(`/v10-lab/maquettes/${encodeURIComponent(name)}`, jsonRequest('PUT', config)),
  duplicateMaquette: (sourceName: string, payload: DuplicateMaquetteRequest) => request<V10Config>(`/v10-lab/maquettes/${encodeURIComponent(sourceName)}/duplicate`, jsonRequest('POST', payload)),
  deleteMaquette: (name: string, deleteDirectory = false) => request<void>(`/v10-lab/maquettes/${encodeURIComponent(name)}${deleteDirectory ? '?deleteDirectory=true' : ''}`, { method: 'DELETE' }),
  getApiTokenStatus: (name: string) => request<ApiTokenStatus>(`/v10-lab/maquettes/${encodeURIComponent(name)}/api-token`),
  saveApiToken: (name: string, token: string) => request<ApiTokenStatus>(`/v10-lab/maquettes/${encodeURIComponent(name)}/api-token`, jsonRequest('PUT', { token })),
  deleteApiToken: (name: string) => request<void>(`/v10-lab/maquettes/${encodeURIComponent(name)}/api-token`, { method: 'DELETE' }),
  validateMaquette: (name: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/validate`, { method: 'POST' }),
  runMaquette: (name: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/run`, { method: 'POST' }),
  runAction: (name: string, actionId: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/actions/${encodeURIComponent(actionId)}/run`, { method: 'POST' }),
  runExecutableCommand: (name: string, targetKind: ExecutableCommandTargetKind, targetName: string, command: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/executable-command/run`, jsonRequest('POST', { targetKind, targetName, command })),
  getMaquetteOpenUrl: (name: string) => request<{ url: string }>(`/v10-lab/maquettes/${encodeURIComponent(name)}/open-url`),
  openMaquetteFolder: (name: string) => request<{ status: string }>(`/v10-lab/maquettes/${encodeURIComponent(name)}/open-folder`, { method: 'POST' }),
  currentRun: (name: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/run/current`),
  logs: (name: string) => request<LogSummary[]>(`/v10-lab/maquettes/${encodeURIComponent(name)}/logs`),
  scanCfg: (name: string, file: File, envName: string, appName: string, importExistingKeys = false) => scanCfg(name, file, envName, appName, importExistingKeys),
  logFile: async (name: string, logFile: string) => {
    const response = await fetch(`${API_BASE_URL}/v10-lab/maquettes/${encodeURIComponent(name)}/logs/${encodeURIComponent(logFile)}`);
    if (!response.ok) {
      throw new Error(`Erreur HTTP: ${response.status}`);
    }
    return response.text();
  },
  killGXProcesses: () => request<ExecutionResponse>('/v10-lab/kill-gx-processes', jsonRequest('POST', { force: true })),
};

async function scanCfg(name: string, file: File, envName: string, appName: string, importExistingKeys: boolean): Promise<ScanCfgResponse> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('envName', envName);
  formData.append('appName', appName);
  formData.append('importExistingKeys', importExistingKeys ? 'true' : 'false');
  const response = await fetch(`${API_BASE_URL}/v10-lab/maquettes/${encodeURIComponent(name)}/scan-cfg`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    await throwResponseError(response);
  }
  return response.json();
}

async function throwResponseError(response: Response): Promise<never> {
  let message = `Erreur HTTP: ${response.status}`;
  try {
    const payload = await response.json();
    message = payload.error ?? (Array.isArray(payload.errors) ? payload.errors.join('\n') : message);
  } catch {
    // Keep generic message.
  }
  throw new Error(message);
}
