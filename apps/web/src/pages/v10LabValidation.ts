import type { V10Config, V10Product } from '../api/v10Lab';
import { messages } from '../i18n';

const m = messages.v10Lab;

export function isServiceDsnRequired(dbType: string): boolean {
  const normalizedType = dbType.trim().toLowerCase();
  return normalizedType !== '' && normalizedType !== 'sqlite';
}

export function validateServiceDsns(config: V10Config, product: V10Product): string {
  for (const [serviceName, service] of Object.entries(config.gedixConfig.services ?? {})) {
    const dbType = service.dbType.trim().toLowerCase();
    if (isServiceDsnRequired(dbType) && !service.dbDsn.trim()) {
      return serviceDsnRequiredMessage(serviceLabel(product, serviceName), dbType);
    }
  }
  return '';
}

export function serviceDsnRequiredMessage(serviceName: string, dbType: string): string {
  return formatMessage(m.errors.serviceDsnRequired, { service: serviceName, dbType });
}

function serviceLabel(product: V10Product, serviceName: string): string {
  return product.services.find((service) => service.name === serviceName)?.label || serviceName;
}

function formatMessage(template: string, values: Record<string, string>): string {
  return Object.entries(values).reduce((result, [key, value]) => result.split(`{{${key}}}`).join(value), template);
}
