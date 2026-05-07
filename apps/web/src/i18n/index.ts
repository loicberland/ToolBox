import fr from './fr.json';

export const messages = fr;

export function t(path: string): string {
  const value = path.split('.').reduce<unknown>((acc, key) => {
    if (acc && typeof acc === 'object' && key in acc) {
      return (acc as Record<string, unknown>)[key];
    }
    return undefined;
  }, messages);
  return typeof value === 'string' ? value : path;
}

export function statusLabel(status: string): string {
  return messages.status[status as keyof typeof messages.status] ?? status;
}
