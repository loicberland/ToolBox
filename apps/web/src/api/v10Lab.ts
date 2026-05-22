import { API_BASE_URL } from '../config/apiConfig';

export const DEFAULT_V10_PRODUCT_ID = 'gedix-prod-v10';

export type V10Product = {
  id: string;
  name: string;
  description: string;
};

export type V10ActionField = {
  name: string;
  label: string;
  type: string;
  required: boolean;
  default: unknown;
  description: string;
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
  };
  runtime: {
    debugTargets: string[];
    openConsole: boolean;
  };
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
};

export type ExecutionResponse = {
  status: string;
  output?: string;
  errors?: string[];
  durationMs?: number;
};

export type LogSummary = {
  name: string;
  sizeBytes: number;
  modifiedAt: string;
};

export type SelectReleasePathResponse = {
  path?: string;
  cancelled: boolean;
};

export type ScanCfgResponse = {
  envName: string;
  appName: string;
  connectors: Array<{ name: string; rawConfig: string }>;
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
  listMaquettes: () => request<MaquetteSummary[]>('/v10-lab/maquettes'),
  createMaquette: (config: V10Config) => request('/v10-lab/maquettes', jsonRequest('POST', config)),
  getMaquette: (name: string) => request<V10Config>(`/v10-lab/maquettes/${encodeURIComponent(name)}`),
  updateMaquette: (name: string, config: V10Config) => request<V10Config>(`/v10-lab/maquettes/${encodeURIComponent(name)}`, jsonRequest('PUT', config)),
  deleteMaquette: (name: string) => request<void>(`/v10-lab/maquettes/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  validateMaquette: (name: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/validate`, { method: 'POST' }),
  runMaquette: (name: string) => request<ExecutionResponse>(`/v10-lab/maquettes/${encodeURIComponent(name)}/run`, { method: 'POST' }),
  logs: (name: string) => request<LogSummary[]>(`/v10-lab/maquettes/${encodeURIComponent(name)}/logs`),
  scanCfg: (name: string, file: File, envName: string, appName: string) => scanCfg(name, file, envName, appName),
  logFile: async (name: string, logFile: string) => {
    const response = await fetch(`${API_BASE_URL}/v10-lab/maquettes/${encodeURIComponent(name)}/logs/${encodeURIComponent(logFile)}`);
    if (!response.ok) {
      throw new Error(`Erreur HTTP: ${response.status}`);
    }
    return response.text();
  },
  killGXProcesses: () => request<ExecutionResponse>('/v10-lab/kill-gx-processes', jsonRequest('POST', { force: true })),
};

async function scanCfg(name: string, file: File, envName: string, appName: string): Promise<ScanCfgResponse> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('envName', envName);
  formData.append('appName', appName);
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
