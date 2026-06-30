import React, { useEffect, useState } from 'react';
import { DBTemplate, ServiceDBConfig, V10Config, V10Product } from '../../../api/v10Lab';
import { messages } from '../../../../../i18n';
import { Button } from '../../../../../shared/components/ui/Button';
import { RequiredDot } from '../../form/RequiredDot';
import { isServiceDsnRequired } from '../../../validation/serviceValidation';
import { ExtraKeyRow, extraKeyRowsFromService, makeID, matchesSearch, normalizeSearch } from '../../../utils/v10LabUtils';

const m = messages.v10Lab;
export function SearchInput({ value, placeholder, onChange }: { value: string; placeholder: string; onChange: (value: string) => void }) {
  return (
    <label className="v10-search-field">
      <input value={value} placeholder={placeholder} onChange={(event) => onChange(event.currentTarget.value)} />
    </label>
  );
}

export function ServicesForm({ config, product, templates, saveAttempted, onChange }: { config: V10Config; product: V10Product; templates: DBTemplate[]; saveAttempted: boolean; onChange: (config: V10Config) => void }) {
  const [search, setSearch] = useState('');
  const updateService = (name: string, service: ServiceDBConfig) => {
    const services = { ...config.gedixConfig.services };
    services[name] = service;
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, services } });
  };
  const normalizedSearch = normalizeSearch(search);
  const services = product.services.filter((serviceDefinition) => {
    const service = config.gedixConfig.services[serviceDefinition.name];
    return matchesSearch(normalizedSearch, [
      serviceDefinition.name,
      serviceDefinition.label,
      service?.dbType,
      service?.dbDsn,
    ]);
  });

  return (
    <div className="v10-service-list">
      {product.services.length === 0 && <p className="muted">{m.noServicesForProduct}</p>}
      {product.services.length > 0 && <SearchInput value={search} placeholder={m.search.servicePlaceholder} onChange={setSearch} />}
      {product.services.length > 0 && services.length === 0 && <p className="muted">{m.search.noResults}</p>}
      {services.map((serviceDefinition) => {
        const name = serviceDefinition.name;
        const existingService = config.gedixConfig.services[name];
        const service = existingService ?? { dbType: 'sqlite', dbDsn: '', extraKeys: {} };
        const dbType = service.dbType || 'sqlite';
        const dsnRequired = isServiceDsnRequired(dbType);
        const dsnInvalid = saveAttempted && dsnRequired && !service.dbDsn.trim();
        return (
          <div className="v10-service-row" key={name}>
            <div>
              <strong>{serviceDefinition.label || name}</strong>
              {!serviceDefinition.hasDatabase && <p className="muted">{m.noDatabase}</p>}
            </div>
            {serviceDefinition.hasDatabase && (
              <div className="v10-service-config">
                <label>{m.dbType}
                  <select value={dbType} onChange={(event) => updateService(name, { ...service, dbType: event.currentTarget.value })}>
                    {['sqlite', 'mysql', 'postgres', 'mssql', 'oracle'].map((type) => <option key={type} value={type}>{type}</option>)}
                  </select>
                </label>
                <label>
                  <span className="v10-field-label">
                    {dbType === 'sqlite' ? m.sqliteDsn : m.dbDsn}
                    {dsnRequired && <RequiredDot />}
                  </span>
                  <input
                    className={dsnInvalid ? 'field-invalid' : ''}
                    placeholder={dbType === 'sqlite' ? m.sqliteDsnPlaceholder : ''}
                    value={service.dbDsn ?? ''}
                    required={dsnRequired}
                    aria-required={dsnRequired}
                    aria-invalid={dsnInvalid || undefined}
                    onChange={(event) => updateService(name, { ...service, dbDsn: event.currentTarget.value })}
                  />
                  {dsnInvalid && <span className="field-error-text">{m.dsnRequired}</span>}
                </label>
                <label>{m.dsnTemplate}
                  <select value="" onChange={(event) => updateService(name, { ...service, dbDsn: event.currentTarget.value })}>
                    <option value="">{m.insertTemplate}</option>
                    {templates.filter((template) => template.template).map((template) => <option key={template.type} value={template.template}>{template.type}</option>)}
                  </select>
                </label>
              </div>
            )}
            {serviceDefinition.supportsExtraKeys && <ExtraKeysEditor serviceKey={`${config.name}:${name}`} service={service} onChange={(next) => updateService(name, next)} />}
          </div>
        );
      })}
    </div>
  );
}

export function ExtraKeysEditor({ serviceKey, service, onChange }: { serviceKey: string; service: ServiceDBConfig; onChange: (service: ServiceDBConfig) => void }) {
  const [rows, setRows] = useState<ExtraKeyRow[]>(() => extraKeyRowsFromService(service));

  useEffect(() => {
    setRows(extraKeyRowsFromService(service));
  }, [serviceKey]);

  const commitRows = (nextRows: ExtraKeyRow[]) => {
    setRows(nextRows);
    const extraKeys: Record<string, string> = {};
    for (const row of nextRows) {
      const key = row.key.trim();
      if (key) {
        extraKeys[key] = row.value;
      }
    }
    onChange({ ...service, extraKeys });
  };

  return (
    <div className="v10-extra-keys">
      <div className="section-header compact">
        <h4>{m.extraKeys}</h4>
        <Button type="button" size="sm" variant="secondary" onClick={() => commitRows([...rows, { id: makeID(), key: '', value: '' }])}>{m.addExtraKey}</Button>
      </div>
      {rows.map((row) => (
        <div className="v10-key-row" key={row.id}>
          <input value={row.key} placeholder={m.extraKeyName} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, key: event.currentTarget.value } : item))} />
          <input value={row.value} placeholder={m.extraKeyValue} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, value: event.currentTarget.value } : item))} />
          <Button type="button" size="sm" variant="danger" onClick={() => commitRows(rows.filter((item) => item.id !== row.id))}>{m.delete}</Button>
        </div>
      ))}
    </div>
  );
}

export const ServicesTab = ServicesForm;

