import { API_BASE_URL } from '../config/apiConfig';

export type ModuleAction = {
  id: string;
  name: string;
  description: string;
};

export type ModuleInfo = {
  id: string;
  name: string;
  description: string;
  actions: ModuleAction[];
};

export async function fetchModules(): Promise<ModuleInfo[]> {
  const response = await fetch(`${API_BASE_URL}/modules`);
  if (!response.ok) {
    throw new Error(`Unable to load modules: ${response.status}`);
  }
  return response.json();
}
