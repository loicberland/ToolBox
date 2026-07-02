import React, { useEffect, useState } from 'react';
import { UnitKind, V10Config, V10Product } from '../../api/v10Lab';
import { messages } from '../../../../i18n';
import { Button } from '../../../../shared/components/ui/Button';
import {
  ConnectorFormRow,
  hasDuplicateConnector,
  makeID,
  matchesSearch,
  normalizeModuleType,
  normalizeSearch,
  unitAddLabel,
  unitConfigKey,
  unitDefinitionForKind,
  unitHelp,
  unitNameLabel,
  unitRowsFromConfig,
  unitScanLabel,
  unitSearchPlaceholder,
  unitsForConfig,
} from '../../utils/v10LabUtils';

const m = messages.v10Lab;

function SearchInput({ value, placeholder, onChange }: { value: string; placeholder: string; onChange: (value: string) => void }) {
  return (
    <div className="v10-search-row">
      <input type="search" value={value} placeholder={placeholder} onChange={(event) => onChange(event.currentTarget.value)} />
    </div>
  );
}
export function UnitsForm({ config, product, unitKind, onChange, onScanCfg }: { config: V10Config; product: V10Product; unitKind: UnitKind; onChange: (config: V10Config) => void; onScanCfg: (unitKind: UnitKind, file: File, importExistingKeys: boolean, replaceExistingUnits: boolean) => void }) {
  const definition = unitDefinitionForKind(product, unitKind);
  const [rows, setRows] = useState<ConnectorFormRow[]>(() => unitRowsFromConfig(config, product, unitKind));
  const [importExistingKeys, setImportExistingKeys] = useState(false);
  const [replaceExistingUnits, setReplaceExistingUnits] = useState(false);
  const [search, setSearch] = useState('');

  useEffect(() => {
    setRows(unitRowsFromConfig(config, product, unitKind));
  }, [config.name, product.id, unitKind]);

  useEffect(() => {
    setRows((current) => {
      const ids = new Map(current.map((row) => [row.name, row.id]));
      return Object.entries(unitsForConfig(config, product, unitKind)).map(([name, connector]) => ({
        id: ids.get(name) ?? makeID(),
        name,
        module: connector.module ?? '',
        rawConfig: connector.rawConfig,
      }));
    });
  }, [config.gedixConfig.connectors, config.gedixConfig.agents, config.gedixConfig.adaptors, config.gedixConfig.units, product.id, unitKind]);

  const commitRows = (nextRows: ConnectorFormRow[]) => {
    setRows(nextRows);
    const units: Record<string, { module: string; rawConfig: string }> = {};
    for (const row of nextRows) {
      const name = row.name.trim();
      if (name) {
        units[name] = { module: normalizeModuleType(row.module), rawConfig: row.rawConfig };
      }
    }
    const unitKey = unitConfigKey(unitKind);
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, [unitKey]: units } });
  };

  const duplicate = hasDuplicateConnector(rows);
  const filteredRows = rows.filter((row) => matchesSearch(normalizeSearch(search), [row.name, row.module, row.rawConfig]));
  const addUnit = () => {
    setSearch('');
    commitRows([...rows, { id: makeID(), name: `${definition.folderPrefix || 'connector-'}${rows.length + 1}`, module: '', rawConfig: '' }]);
  };

  return (
    <div className="v10-connector-list">
      <p className="readonly-notice">{unitHelp(definition)}</p>
      <p className="readonly-notice">{m.units.moduleHelp}</p>
      <p className="muted">{m.scanCfgHelp}</p>
      <div className="button-row">
        <label className="ui-button secondary sm v10-file-button">
          {unitScanLabel(unitKind)}
          <input
            type="file"
            accept=".cfg"
            onChange={(event) => {
              const file = event.currentTarget.files?.[0];
              if (file) {
                onScanCfg(unitKind, file, importExistingKeys, replaceExistingUnits);
              }
              event.currentTarget.value = '';
            }}
          />
        </label>
        <label className="checkbox-row v10-inline-checkbox">
          <input type="checkbox" checked={importExistingKeys} onChange={(event) => setImportExistingKeys(event.currentTarget.checked)} />
          {m.units.importExistingKeys}
        </label>
        <label className="checkbox-row v10-inline-checkbox">
          <input type="checkbox" checked={replaceExistingUnits} onChange={(event) => setReplaceExistingUnits(event.currentTarget.checked)} />
          {m.units.replaceExistingUnits}
        </label>
      </div>
      {duplicate && <p className="error">{m.duplicateConnector}</p>}
      <SearchInput value={search} placeholder={unitSearchPlaceholder(unitKind)} onChange={setSearch} />
      {rows.length > 0 && filteredRows.length === 0 && <p className="muted">{m.search.noResults}</p>}
      {filteredRows.map((row) => (
        <div className="v10-connector-row" key={row.id}>
          <div>
            <label>{unitNameLabel(unitKind)}
              <input value={row.name} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, name: event.currentTarget.value } : item))} />
            </label>
            <label>{m.units.module}
              <input value={row.module} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, module: event.currentTarget.value } : item))} />
            </label>
          </div>
          <label>{m.units.rawConfig}
            <textarea value={row.rawConfig} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, rawConfig: event.currentTarget.value } : item))} />
          </label>
          <Button type="button" variant="danger" size="sm" onClick={() => commitRows(rows.filter((item) => item.id !== row.id))}>{m.delete}</Button>
        </div>
      ))}
      <Button type="button" variant="secondary" onClick={addUnit}>{unitAddLabel(unitKind)}</Button>
    </div>
  );
}

