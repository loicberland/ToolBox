declare global {
  interface Window {
    TOOLBOX?: {
      services?: {
        api?: {
          url?: string;
        };
      };
    };
  }
}

export const API_BASE_URL = window.TOOLBOX?.services?.api?.url ?? '/api';
