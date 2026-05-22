import React, { useEffect, useMemo, useState } from 'react';
import {
  DBTemplate,
  ExecutionResponse,
  LogSummary,
  MaquetteSummary,
  PipelineStep,
  ServiceDBConfig,
  DEFAULT_V10_PRODUCT_ID,
  V10Action,
  V10Config,
  V10Product,
  v10LabApi,
} from '../api/v10Lab';
import { Button } from '../components/ui/Button';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { messages } from '../i18n';

const serviceNames = ['webserver', 'auth', 'filestore', 'entreprise', 'etl', 'dnc', 'reactor', 'config'];
const noDatabaseServices = new Set(['webserver', 'reactor']);
const m = messages.v10Lab;
const tabs = [m.tabs.general, m.tabs.gedix, m.tabs.services, m.tabs.connectors, m.tabs.pipeline, m.tabs.execution, m.tabs.json] as const;
const systemPipelineActions = new Set(['create-env', 'configure-gedix-cfg', 'start-maquette', 'stop-maquette', 'start-services', 'stop-services', 'kill-gx-processes', 'update-env']);
type Tab = typeof tabs[number];
type RunState = 'idle' | 'running' | 'success' | 'failed';
type ConnectorFormRow = {
  id: string;
  name: string;
  rawConfig: string;
};
type ExtraKeyRow = {
  id: string;
  key: string;
  value: string;
};
type BeforeLeaveHandler = () => Promise<boolean>;

export function V10LabPage({ onBeforeLeaveChange }: { onBeforeLeaveChange?: (handler: BeforeLeaveHandler | null) => void }) {
  const [products, setProducts] = useState<V10Product[]>([]);
  const [actions, setActions] = useState<V10Action[]>([]);
  const [templates, setTemplates] = useState<DBTemplate[]>([]);
  const [maquettes, setMaquettes] = useState<MaquetteSummary[]>([]);
  const [selectedName, setSelectedName] = useState('');
  const [config, setConfig] = useState<V10Config | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>(m.tabs.general);
  const [showCreate, setShowCreate] = useState(false);
  const [draft, setDraft] = useState(() => defaultConfig());
  const [jsonText, setJsonText] = useState('');
  const [logs, setLogs] = useState<LogSummary[]>([]);
  const [selectedLog, setSelectedLog] = useState('');
  const [defaultTargetPath, setDefaultTargetPath] = useState('');
  const [busy, setBusy] = useState(false);
  const [runState, setRunState] = useState<RunState>('idle');
  const [isDirty, setIsDirty] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [confirmKill, setConfirmKill] = useState(false);
  const [execution, setExecution] = useState<ExecutionResponse | null>(null);

  useEffect(() => {
    void loadInitial();
  }, []);

  useEffect(() => {
    if (config) {
      setJsonText(JSON.stringify(config, null, 2));
    }
  }, [config]);

  useEffect(() => {
    if (config) {
      void loadActions(config.product);
    }
  }, [config?.product]);

  useEffect(() => {
    void loadDefaultTarget(showCreate ? draft.name : config?.name ?? '');
  }, [showCreate, draft.name, config?.name]);

  useEffect(() => {
    if (!onBeforeLeaveChange) {
      return;
    }
    onBeforeLeaveChange(async () => {
      if (!config || !isDirty) {
        return true;
      }
      setMessage(m.savingBeforeModuleChange);
      const saved = await saveCurrent();
      if (!saved) {
        setError(m.autosaveFailed);
      }
      return saved;
    });
    return () => onBeforeLeaveChange(null);
  }, [onBeforeLeaveChange, config, isDirty]);

  const selectedSummary = maquettes.find((item) => item.name === selectedName);

  async function loadInitial() {
    await run(async () => {
      const [productItems, templateItems, maquetteItems] = await Promise.all([
        v10LabApi.products(),
        v10LabApi.dbTemplates(),
        v10LabApi.listMaquettes(),
      ]);
      setProducts(productItems);
      setTemplates(templateItems);
      setMaquettes(maquetteItems);
      const product = productItems[0]?.id ?? DEFAULT_V10_PRODUCT_ID;
      setDraft(defaultConfig(product));
      await loadActions(product);
    });
  }

  async function loadActions(product: string) {
    const items = await v10LabApi.actions(product || DEFAULT_V10_PRODUCT_ID);
    setActions(items);
  }

  async function loadDefaultTarget(name: string) {
    try {
      const result = await v10LabApi.defaultTarget(name);
      setDefaultTargetPath(result.targetPath);
    } catch {
      setDefaultTargetPath('');
    }
  }

  async function reloadList() {
    const items = await v10LabApi.listMaquettes();
    setMaquettes(items);
  }

  async function openMaquette(name: string) {
    await run(async () => {
      const loaded = normalizeConfig(await v10LabApi.getMaquette(name));
      setSelectedName(loaded.name);
      setConfig(loaded);
      setIsDirty(false);
      setActiveTab(m.tabs.general);
      setExecution(null);
      setSelectedLog('');
      setLogs([]);
    });
  }

  async function createMaquette() {
    const validation = validateConfig(draft);
    if (validation) {
      setError(validation);
      return;
    }
    await run(async () => {
      const next = normalizeConfig(draft);
      await v10LabApi.createMaquette(next);
      setShowCreate(false);
      setDraft(defaultConfig(products[0]?.id));
      await reloadList();
      await openMaquette(next.name);
      setMessage(m.created);
    });
  }

  async function saveCurrent(): Promise<boolean> {
    if (!config) {
      return true;
    }
    const validation = validateConfig(config);
    if (validation) {
      setError(validation);
      return false;
    }
    let saved = false;
    await run(async () => {
      const next = normalizeConfig(await v10LabApi.updateMaquette(config.name, normalizeConfig(config)));
      setConfig(next);
      setIsDirty(false);
      await reloadList();
      setMessage(m.saved);
      saved = true;
    });
    return saved;
  }

  async function changeTab(nextTab: Tab) {
    if (nextTab === activeTab) {
      return;
    }
    if (config && isDirty) {
      setMessage(m.savingBeforeTabChange);
      const saved = await saveCurrent();
      if (!saved) {
        setError(m.autosaveFailed);
        return;
      }
    }
    setActiveTab(nextTab);
  }

  async function validateCurrent(name = selectedName) {
    if (!name) {
      return;
    }
    if (config && isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    await run(async () => {
      const result = await v10LabApi.validateMaquette(name);
      setExecution(result);
      setMessage(m.validationOk);
    });
  }

  async function runSystemAction(actionId: string, name = selectedName) {
    if (!name || runState === 'running') {
      return;
    }
    if (config && isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    setRunState('running');
    setExecution({ status: 'running', running: true, output: m.executionRunning });
    await run(async () => {
      const started = await v10LabApi.runAction(name, actionId);
      setExecution(started);
      const result = await pollCurrentRun(name);
      await reloadList();
      await refreshLogs(name);
      setRunState(result.status === 'success' ? 'success' : 'failed');
      setMessage(result.status === 'success' ? m.executionFinished : m.executionFailed);
    }, () => setRunState('failed'));
  }

  async function runCurrent(name = selectedName) {
    if (!name || runState === 'running') {
      return;
    }
    if (!config?.pipeline?.some((step) => !systemPipelineActions.has(step.action))) {
      setExecution({ status: 'idle', output: m.pipeline.noApiActions });
      setMessage(m.pipeline.noApiActions);
      return;
    }
    if (config && isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    setRunState('running');
    setExecution({ status: 'running', running: true, output: m.executionRunning });
    await run(async () => {
      const started = await v10LabApi.runMaquette(name);
      setExecution(started);
      const result = await pollCurrentRun(name);
      await reloadList();
      await refreshLogs(name);
      setRunState(result.status === 'success' ? 'success' : 'failed');
      setMessage(result.status === 'success' ? m.executionFinished : m.executionFailed);
    }, () => setRunState('failed'));
  }

  async function pollCurrentRun(name: string): Promise<ExecutionResponse> {
    let result = await v10LabApi.currentRun(name);
    setExecution(result);
    while (result.running) {
      await delay(1500);
      result = await v10LabApi.currentRun(name);
      setExecution(result);
    }
    return result;
  }

  async function refreshLogs(name = selectedName) {
    if (!name) {
      return;
    }
    const items = await v10LabApi.logs(name);
    setLogs(items);
  }

  async function readLog(logFile: string) {
    await run(async () => {
      const text = await v10LabApi.logFile(selectedName, logFile);
      setSelectedLog(text);
    });
  }

  async function deleteMaquette(name: string) {
    await run(async () => {
      await v10LabApi.deleteMaquette(name);
      setConfirmDelete(null);
      if (selectedName === name) {
        setSelectedName('');
        setConfig(null);
      }
      await reloadList();
      setMessage(m.deleted);
    });
  }

  async function killGXProcesses() {
    setConfirmKill(false);
    await run(async () => {
      const result = await v10LabApi.killGXProcesses();
      setExecution(result);
      await refreshLogs();
      setMessage(m.killFinished);
    });
  }

  async function saveJSON() {
    if (!config) {
      return;
    }
    try {
      const parsed = normalizeConfig(JSON.parse(jsonText) as V10Config);
      const saved = normalizeConfig(await v10LabApi.updateMaquette(config.name, parsed));
      setConfig(saved);
      setIsDirty(false);
      await reloadList();
      setMessage(m.jsonSaved);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'JSON invalide');
    }
  }

  async function selectReleaseZip(target: V10Config, onChange: (config: V10Config) => void) {
    await run(async () => {
      const result = await v10LabApi.selectReleasePath();
      if (!result.cancelled && result.path) {
        onChange({ ...target, release: { ...target.release, zipPath: result.path } });
      }
    });
  }

  async function closeMaquette() {
    if (isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    setSelectedName('');
    setConfig(null);
    setExecution(null);
    setSelectedLog('');
    setLogs([]);
    setRunState('idle');
  }

  async function toggleMaquette(name: string) {
    if (selectedName === name) {
      await closeMaquette();
      return;
    }
    if (config && isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    await openMaquette(name);
  }

  async function scanCfg(file: File) {
    if (!config) {
      return;
    }
    if (!stringsEqual(filepathExt(file.name), '.cfg')) {
      setError(m.errors.cfgOnly);
      return;
    }
    await run(async () => {
      const result = await v10LabApi.scanCfg(config.name, file, config.maquette.envName, config.maquette.appName || 'prod');
      const connectors = { ...config.gedixConfig.connectors };
      for (const connector of result.connectors) {
        if (!connectors[connector.name]) {
          connectors[connector.name] = { rawConfig: connector.rawConfig };
        }
      }
      updateConfig({
        ...config,
        maquette: { ...config.maquette, envName: result.envName || config.maquette.envName, appName: result.appName || config.maquette.appName },
        gedixConfig: { ...config.gedixConfig, connectors },
      });
      setMessage(`${result.connectors.length} connecteur(s) détecté(s).`);
    });
  }

  async function run(task: () => Promise<void>, onError?: () => void) {
    setBusy(true);
    setError('');
    setMessage('');
    try {
      await task();
    } catch (err) {
      onError?.();
      setError(err instanceof Error ? err.message : 'Erreur inconnue');
    } finally {
      setBusy(false);
    }
  }

  function updateConfig(next: V10Config) {
    setConfig(next);
    setIsDirty(true);
  }

  return (
    <div className="workspace v10-lab-workspace">
      <header className="page-header">
        <div className="page-title-group">
          <p className="page-eyebrow">{m.title}</p>
          <h2>{m.subtitle}</h2>
          <p>{m.description}</p>
        </div>
        <div className="page-actions">
          <Button type="button" onClick={() => setShowCreate((value) => !value)}>{m.newMaquette}</Button>
        </div>
      </header>

      {error && <p className="error whitespace">{error}</p>}
      {message && <p className="info-message">{message}</p>}

      {showCreate && (
        <section className="ui-card v10-section">
          <div className="ui-card-header">
            <h3>{m.newMaquette}</h3>
          </div>
          <MaquetteGeneralForm config={draft} products={products} defaultTargetPath={defaultTargetPath} onChange={setDraft} onSelectZip={selectReleaseZip} creating />
          <div className="button-row end">
            <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>{messages.common.cancel}</Button>
            <Button type="button" onClick={() => void createMaquette()} disabled={busy}>{m.create}</Button>
          </div>
        </section>
      )}

      <section className="ui-card v10-section">
        <div className="ui-card-header">
          <h3>{m.registeredMaquettes}</h3>
          <Button type="button" variant="secondary" size="sm" onClick={() => void reloadList()} disabled={busy}>{m.refreshLogs}</Button>
        </div>
        {maquettes.length === 0 ? (
          <div className="empty-state">
            <h3>{m.noMaquette}</h3>
          </div>
        ) : (
          <div className="v10-table">
            <div className="v10-table-head">
              <span>{m.name}</span>
              <span>{m.product}</span>
              <span>{m.installed}</span>
              <span>{m.latestRun}</span>
              <span>{m.actions}</span>
            </div>
            {maquettes.map((item) => (
              <div className={`v10-table-row ${item.name === selectedName ? 'active' : ''}`} key={item.name}>
                <strong>{item.name}</strong>
                <span>{item.product}</span>
                <span>{item.existsOnDisk ? m.yes : m.no}</span>
                <span>{item.lastRunAt ? `${formatDate(item.lastRunAt)} (${item.lastStatus ?? 'unknown'})` : '-'}</span>
                <div className="button-row">
                  <Button type="button" size="sm" variant="secondary" onClick={() => void toggleMaquette(item.name)}>{item.name === selectedName ? m.close : m.open}</Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {config && (
        <section className="ui-card v10-section">
          <div className="ui-card-header">
            <div>
              <h3>{config.name}</h3>
              <p className="muted">{selectedSummary?.targetPath ?? config.maquette.targetPath}</p>
            </div>
            <div className="button-row end">
              <Button type="button" variant="secondary" onClick={() => void closeMaquette()} disabled={busy}>{m.close}</Button>
              <Button type="button" variant="secondary" onClick={() => void openMaquette(config.name)} disabled={busy}>{m.reload}</Button>
              <Button type="button" onClick={() => void saveCurrent()} disabled={busy}>{m.save}</Button>
              <Button type="button" variant="danger" onClick={() => setConfirmDelete(config.name)} disabled={busy}>{m.delete}</Button>
            </div>
          </div>

          <div className="v10-tabs">
            {tabs.map((tab) => (
              <button type="button" key={tab} className={tab === activeTab ? 'active' : ''} onClick={() => void changeTab(tab)}>
                {tab}
              </button>
            ))}
          </div>

          {activeTab === m.tabs.general && <MaquetteGeneralForm config={config} products={products} defaultTargetPath={defaultTargetPath} onChange={updateConfig} onSelectZip={selectReleaseZip} />}
          {activeTab === m.tabs.gedix && <GedixForm config={config} onChange={updateConfig} />}
          {activeTab === m.tabs.services && <ServicesForm config={config} templates={templates} onChange={updateConfig} />}
          {activeTab === m.tabs.connectors && <ConnectorsForm config={config} onChange={updateConfig} onScanCfg={(file) => void scanCfg(file)} />}
          {activeTab === m.tabs.pipeline && (
            <LocalErrorBoundary>
              <PipelineBuilder config={config} actions={actions} onChange={updateConfig} />
            </LocalErrorBoundary>
          )}
          {activeTab === m.tabs.execution && (
            <ExecutionPanel
              config={config}
              busy={busy}
              runState={runState}
              execution={execution}
              logs={logs}
              selectedLog={selectedLog}
              onConfigChange={updateConfig}
              onCreate={() => void runSystemAction('create-env')}
              onConfigure={() => void runSystemAction('configure-gedix-cfg')}
              onStart={() => void runSystemAction('start-maquette')}
              onRunPipeline={() => void runCurrent()}
              onKill={() => setConfirmKill(true)}
              onRefreshLogs={() => void refreshLogs()}
              onReadLog={(logFile) => void readLog(logFile)}
            />
          )}
          {activeTab === m.tabs.json && (
            <div className="v10-json-panel">
              <textarea value={jsonText} onChange={(event) => setJsonText(event.target.value)} spellCheck={false} />
              {execution?.errors?.length ? <p className="error whitespace">{execution.errors.join('\n')}</p> : null}
              {execution?.status === 'valid' && <p className="info-message">{execution.output || m.validationOk}</p>}
              <div className="button-row end">
                <Button type="button" variant="secondary" onClick={() => void navigator.clipboard?.writeText(jsonText)}>{m.copy}</Button>
                <Button type="button" onClick={() => void saveJSON()}>{m.saveJson}</Button>
                <Button type="button" variant="secondary" onClick={() => void validateCurrent()} disabled={busy}>{m.json.validateConfig}</Button>
              </div>
            </div>
          )}
        </section>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        title={m.deleteTitle}
        message={m.deleteMessage}
        confirmLabel={m.delete}
        onCancel={() => setConfirmDelete(null)}
        onConfirm={() => confirmDelete && void deleteMaquette(confirmDelete)}
      />
      <ConfirmDialog
        open={confirmKill}
        title={m.killTitle}
        message={m.killMessage}
        confirmLabel="Continuer"
        onCancel={() => setConfirmKill(false)}
        onConfirm={() => void killGXProcesses()}
      />
    </div>
  );
}

function MaquetteGeneralForm({ config, products, defaultTargetPath, onChange, onSelectZip, creating = false }: {
  config: V10Config;
  products: V10Product[];
  defaultTargetPath: string;
  onChange: (config: V10Config) => void;
  onSelectZip: (config: V10Config, onChange: (config: V10Config) => void) => void;
  creating?: boolean;
}) {
  return (
    <div className="form-grid v10-form-grid">
      <label>{m.name}
        <input value={config.name} disabled={!creating} onChange={(event) => onChange({ ...config, name: event.target.value })} />
      </label>
      <label>{m.product}
        <select value={config.product} onChange={(event) => onChange({ ...config, product: event.target.value })}>
          {products.map((product) => <option value={product.id} key={product.id}>{product.name}</option>)}
        </select>
      </label>
      <label>{m.releaseZip}
        <div className="v10-file-row">
          <Button type="button" variant="secondary" size="sm" onClick={() => onSelectZip(config, onChange)}>
            {m.selectZip}
          </Button>
          <input placeholder={m.manualZip} value={config.release.zipPath} onChange={(event) => onChange({ ...config, release: { ...config.release, zipPath: event.target.value } })} />
        </div>
      </label>
      <label>{m.targetPath}
        <input placeholder={m.targetPlaceholder.replace('{{path}}', defaultTargetPath)} value={config.maquette.targetPath} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, targetPath: event.target.value } })} />
      </label>
      <label>{m.envName}
        <input value={config.maquette.envName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, envName: event.target.value } })} />
      </label>
      <label>{m.appName}
        <input value={config.maquette.appName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, appName: event.target.value } })} />
      </label>
      <label className="checkbox-row">
        <input type="checkbox" checked={config.release.overwrite} onChange={(event) => onChange({ ...config, release: { ...config.release, overwrite: event.target.checked } })} />
        {m.overwriteLabel}
      </label>
      <p className="muted v10-help-text">{m.overwriteHelp}</p>
      {creating && <GedixForm config={config} onChange={onChange} compact />}
    </div>
  );
}

function DebugTargetsEditor({ config, onChange }: { config: V10Config; onChange: (config: V10Config) => void }) {
  const [selected, setSelected] = useState('');
  const options = [...serviceNames, ...Object.keys(config.gedixConfig.connectors ?? {})]
    .filter((item, index, items) => items.indexOf(item) === index)
    .filter((item) => !config.runtime.debugTargets.includes(item));
  return (
    <div className="v10-debug-targets">
      <div className="v10-file-row">
        <select value={selected} onChange={(event) => setSelected(event.target.value)}>
          <option value="">{m.chooseDebugTarget}</option>
          {options.map((item) => <option key={item} value={item}>{item}</option>)}
        </select>
        <Button
          type="button"
          variant="secondary"
          size="sm"
          disabled={!selected}
          onClick={() => {
            onChange({ ...config, runtime: { ...config.runtime, debugTargets: [...config.runtime.debugTargets, selected] } });
            setSelected('');
          }}
        >
          {m.addDebugTarget}
        </Button>
      </div>
      <div className="button-row">
        {config.runtime.debugTargets.map((target) => (
          <Button key={target} type="button" size="sm" variant="secondary" onClick={() => onChange({ ...config, runtime: { ...config.runtime, debugTargets: config.runtime.debugTargets.filter((item) => item !== target) } })}>
            {target} - {m.removeDebugTarget}
          </Button>
        ))}
      </div>
    </div>
  );
}

function GedixForm({ config, onChange, compact = false }: { config: V10Config; onChange: (config: V10Config) => void; compact?: boolean }) {
  const content = (
    <>
      <label>FQDN
        <input value={config.gedixConfig.fqdn} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, fqdn: event.target.value } })} />
      </label>
      <label>Port
        <input type="number" min={0} max={65535} value={config.gedixConfig.port} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, port: Number(event.target.value) } })} />
      </label>
    </>
  );
  return compact ? content : <div className="form-grid v10-form-grid">{content}</div>;
}

function ServicesForm({ config, templates, onChange }: { config: V10Config; templates: DBTemplate[]; onChange: (config: V10Config) => void }) {
  const updateService = (name: string, service: ServiceDBConfig | null) => {
    const services = { ...config.gedixConfig.services };
    if (service) {
      services[name] = service;
    } else {
      delete services[name];
    }
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, services } });
  };

  return (
    <div className="v10-service-list">
      {serviceNames.map((name) => {
        const disabled = noDatabaseServices.has(name);
        const existingService = config.gedixConfig.services[name];
        const service = existingService ?? { dbType: '', dbDsn: '', extraKeys: {} };
        const enabled = Boolean(existingService?.dbType);
        return (
          <div className="v10-service-row" key={name}>
            <div>
              <strong>{name}</strong>
              {disabled && <p className="muted">{m.noDatabase}</p>}
            </div>
            {!disabled && (
              <>
                <label className="checkbox-row">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(event) => updateService(name, event.target.checked ? { dbType: 'sqlite', dbDsn: '', extraKeys: {} } : null)}
                  />
                  {m.configureDb}
                </label>
                {enabled && (
                  <div className="v10-service-config">
                    <label>{m.dbType}
                      <select value={service?.dbType ?? ''} onChange={(event) => updateService(name, { ...service!, dbType: event.target.value, dbDsn: event.target.value === 'sqlite' ? '' : service?.dbDsn ?? '' })}>
                        {['sqlite', 'mysql', 'postgres', 'mssql', 'oracle'].map((type) => <option key={type} value={type}>{type}</option>)}
                      </select>
                    </label>
                    <label>{service?.dbType === 'sqlite' ? m.sqliteDsn : m.dbDsn}
                      <input placeholder={service?.dbType === 'sqlite' ? m.sqliteDsnPlaceholder : ''} value={service?.dbDsn ?? ''} onChange={(event) => updateService(name, { ...service!, dbDsn: event.target.value })} />
                    </label>
                    <label>{m.dsnTemplate}
                      <select value="" onChange={(event) => updateService(name, { ...service!, dbDsn: event.target.value })}>
                        <option value="">{m.insertTemplate}</option>
                        {templates.filter((template) => template.template).map((template) => <option key={template.type} value={template.template}>{template.type}</option>)}
                      </select>
                    </label>
                  </div>
                )}
              </>
            )}
            <ExtraKeysEditor serviceKey={`${config.name}:${name}`} service={service} onChange={(next) => updateService(name, next)} />
          </div>
        );
      })}
    </div>
  );
}

function ExtraKeysEditor({ serviceKey, service, onChange }: { serviceKey: string; service: ServiceDBConfig; onChange: (service: ServiceDBConfig) => void }) {
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
          <input value={row.key} placeholder={m.extraKeyName} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, key: event.target.value } : item))} />
          <input value={row.value} placeholder={m.extraKeyValue} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, value: event.target.value } : item))} />
          <Button type="button" size="sm" variant="danger" onClick={() => commitRows(rows.filter((item) => item.id !== row.id))}>{m.delete}</Button>
        </div>
      ))}
    </div>
  );
}

function ConnectorsForm({ config, onChange, onScanCfg }: { config: V10Config; onChange: (config: V10Config) => void; onScanCfg: (file: File) => void }) {
  const [rows, setRows] = useState<ConnectorFormRow[]>(() => connectorRowsFromConfig(config));

  useEffect(() => {
    setRows(connectorRowsFromConfig(config));
  }, [config.name]);

  useEffect(() => {
    setRows((current) => {
      const existing = new Set(current.map((row) => row.name));
      const missing = Object.entries(config.gedixConfig.connectors)
        .filter(([name]) => !existing.has(name))
        .map(([name, connector]) => ({ id: makeID(), name, rawConfig: connector.rawConfig }));
      return missing.length > 0 ? [...current, ...missing] : current;
    });
  }, [config.gedixConfig.connectors]);

  const commitRows = (nextRows: ConnectorFormRow[]) => {
    setRows(nextRows);
    const connectors: Record<string, { rawConfig: string }> = {};
    for (const row of nextRows) {
      const name = row.name.trim();
      if (name) {
        connectors[name] = { rawConfig: row.rawConfig };
      }
    }
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, connectors } });
  };

  const duplicate = hasDuplicateConnector(rows);

  return (
    <div className="v10-connector-list">
      <p className="readonly-notice">{m.connectorHelp}</p>
      <p className="muted">{m.scanCfgHelp}</p>
      <div className="button-row">
        <label className="ui-button secondary sm v10-file-button">
          {m.scanCfg}
          <input
            type="file"
            accept=".cfg"
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                onScanCfg(file);
              }
              event.currentTarget.value = '';
            }}
          />
        </label>
      </div>
      {duplicate && <p className="error">{m.duplicateConnector}</p>}
      {rows.map((row) => (
        <div className="v10-connector-row" key={row.id}>
          <input value={row.name} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, name: event.target.value } : item))} />
          <textarea value={row.rawConfig} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, rawConfig: event.target.value } : item))} />
          <Button type="button" variant="danger" size="sm" onClick={() => commitRows(rows.filter((item) => item.id !== row.id))}>{m.delete}</Button>
        </div>
      ))}
      <Button type="button" variant="secondary" onClick={() => commitRows([...rows, { id: makeID(), name: `connector-${rows.length + 1}`, rawConfig: '' }])}>{messages.common.add}</Button>
    </div>
  );
}

function PipelineBuilder({ config, actions, onChange }: { config: V10Config; actions: V10Action[]; onChange: (config: V10Config) => void }) {
  const byID = useMemo<Record<string, V10Action>>(() => Object.fromEntries(actions.map((action) => [action.id, action])), [actions]);
  const legacySteps = (config.pipeline ?? []).filter((step) => systemPipelineActions.has(step.action));
  const apiSteps = (config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action));
  const updateStep = (index: number, step: PipelineStep) => {
    onChange({ ...config, pipeline: apiSteps.map((item, itemIndex) => itemIndex === index ? step : item) });
  };
  const move = (index: number, direction: -1 | 1) => {
    const next = [...apiSteps];
    const target = index + direction;
    if (target < 0 || target >= next.length) {
      return;
    }
    [next[index], next[target]] = [next[target], next[index]];
    onChange({ ...config, pipeline: next });
  };
  return (
    <div className="v10-pipeline">
      <p className="readonly-notice">{m.pipeline.help}</p>
      {legacySteps.length > 0 && (
        <div className="readonly-notice warning">
          <p>{m.pipeline.legacySystemActions}</p>
          <Button type="button" size="sm" variant="secondary" onClick={() => onChange({ ...config, pipeline: apiSteps })}>{m.pipeline.cleanSystemActions}</Button>
        </div>
      )}
      {actions.length === 0 && <p className="muted">{m.pipeline.noApiActions}</p>}
      {apiSteps.map((step, index) => {
        const action = byID[step.action];
        const fields = action?.fields ?? [];
        return (
          <div className="v10-pipeline-step" key={`${step.action}-${index}`}>
            <div className="v10-step-order">{index + 1}</div>
            <div className="v10-step-body">
              <div className="form-grid v10-form-grid">
                <label>{m.action}
                  <select value={step.action} onChange={(event) => {
                    const selected = byID[event.target.value];
                    updateStep(index, { action: event.target.value, label: selected?.label ?? '', params: {} });
                  }}>
                    <option value="">{m.chooseAction}</option>
                    {actions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
                  </select>
                </label>
                <label>{m.label}
                  <input value={step.label} onChange={(event) => updateStep(index, { ...step, label: event.target.value })} />
                </label>
              </div>
              {step.action === 'create-env' && <p className="readonly-notice">{m.actionUsesGeneralSettings}</p>}
              {fields.length > 0 && (
                <div className="form-grid v10-form-grid">
                  {fields.map((field) => (
                    <ActionFieldInput
                      field={field}
                      value={step.params?.[field.name]}
                      key={field.name}
                      onChange={(value) => updateStep(index, { ...step, params: { ...(step.params ?? {}), [field.name]: value } })}
                    />
                  ))}
                </div>
              )}
              <div className="button-row">
                <Button type="button" size="sm" variant="secondary" onClick={() => move(index, -1)}>{m.moveUp}</Button>
                <Button type="button" size="sm" variant="secondary" onClick={() => move(index, 1)}>{m.moveDown}</Button>
                <Button type="button" size="sm" variant="danger" onClick={() => onChange({ ...config, pipeline: apiSteps.filter((_, itemIndex) => itemIndex !== index) })}>{m.delete}</Button>
              </div>
            </div>
          </div>
        );
      })}
      <div className="button-row">
        <Button type="button" variant="secondary" onClick={() => onChange({ ...config, pipeline: [...apiSteps, { action: actions[0]?.id ?? '', label: actions[0]?.label ?? '', params: {} }] })} disabled={actions.length === 0}>{m.addAction}</Button>
      </div>
    </div>
  );
}

function ActionFieldInput({ field, value, onChange }: { field: V10Action['fields'][number]; value: unknown; onChange: (value: unknown) => void }) {
  if (field.type === 'bool') {
    return <label className="checkbox-row"><input type="checkbox" checked={Boolean(value)} onChange={(event) => onChange(event.target.checked)} />{field.label}</label>;
  }
  if (field.type === 'string[]') {
    return <label>{field.label}<input value={Array.isArray(value) ? value.join(',') : ''} onChange={(event) => onChange(event.target.value.split(',').map((item) => item.trim()).filter(Boolean))} /></label>;
  }
  return <label>{field.label}<input value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.target.value)} /></label>;
}

function ExecutionPanel({ config, busy, runState, execution, logs, selectedLog, onConfigChange, onCreate, onConfigure, onStart, onRunPipeline, onKill, onRefreshLogs, onReadLog }: {
  config: V10Config;
  busy: boolean;
  runState: RunState;
  execution: ExecutionResponse | null;
  logs: LogSummary[];
  selectedLog: string;
  onConfigChange: (config: V10Config) => void;
  onCreate: () => void;
  onConfigure: () => void;
  onStart: () => void;
  onRunPipeline: () => void;
  onKill: () => void;
  onRefreshLogs: () => void;
  onReadLog: (logFile: string) => void;
}) {
  const currentLog = execution?.log || execution?.output || execution?.status || '';
  const disabled = busy || runState === 'running';
  return (
    <div className="v10-execution">
      <section className="v10-execution-section">
        <h4>{m.execution.debugTitle}</h4>
        <DebugTargetsEditor config={config} onChange={onConfigChange} />
      </section>
      <section className="v10-execution-section">
        <h4>{m.execution.actionsTitle}</h4>
        <div className="button-row">
          <Button type="button" onClick={onCreate} disabled={disabled}>{m.execution.createMaquette}</Button>
          <Button type="button" variant="secondary" disabled title={m.execution.updateNotImplemented}>{m.execution.updateMaquette}</Button>
          <Button type="button" variant="secondary" onClick={onConfigure} disabled={disabled}>{m.execution.configureCfg}</Button>
          <Button type="button" variant="secondary" onClick={onStart} disabled={disabled}>{m.execution.startMaquette}</Button>
        </div>
      </section>
      <section className="v10-execution-section">
        <h4>{m.execution.apiPipelineTitle}</h4>
        <div className="button-row">
          <Button type="button" onClick={onRunPipeline} disabled={disabled}>{runState === 'running' ? m.running : m.execution.runApiPipeline}</Button>
        </div>
      </section>
      <section className="v10-execution-section">
        <h4>{m.execution.maintenanceTitle}</h4>
        <div className="button-row">
          <Button type="button" variant="danger" onClick={onKill} disabled={disabled}>{m.taskkill}</Button>
          <Button type="button" variant="secondary" onClick={onRefreshLogs} disabled={busy}>{m.execution.refreshLogs}</Button>
        </div>
      </section>
      <h4>{m.currentExecutionLogs}</h4>
      {execution?.errors?.length ? <p className="error whitespace">{execution.errors.join('\n')}</p> : null}
      <pre className="v10-output">{currentLog || m.noLog}</pre>
      <h4>{m.previousLogs}</h4>
      <div className="v10-log-layout">
        <div className="v10-log-list">
          {logs.length === 0 ? <p className="muted">{m.noLog}</p> : logs.map((log) => (
            <button type="button" key={log.name} onClick={() => onReadLog(log.name)}>
              <strong>{log.name}</strong>
              <span>{formatDate(log.modifiedAt)}</span>
            </button>
          ))}
        </div>
        <pre className="v10-output">{selectedLog || m.selectLog}</pre>
      </div>
    </div>
  );
}

function defaultConfig(product = DEFAULT_V10_PRODUCT_ID): V10Config {
  return normalizeConfig({
    name: '',
    product,
    release: { zipPath: '', workDir: '', overwrite: false },
    maquette: { targetPath: '', envName: 'demo', appName: 'prod' },
    gedixConfig: { fqdn: '', port: 80, services: {}, connectors: {} },
    runtime: { debugTargets: [], openConsole: true },
    pipeline: [],
  } as V10Config);
}

function normalizeConfig(config: V10Config): V10Config {
  const release = config.release ?? {};
  const maquette = config.maquette ?? {};
  const gedixConfig = config.gedixConfig ?? {};
  const runtime = config.runtime ?? {};
  return {
    ...config,
    product: config.product || DEFAULT_V10_PRODUCT_ID,
    release: {
      ...release,
      zipPath: release.zipPath ?? '',
      workDir: release.workDir ?? '',
      overwrite: release.overwrite ?? false,
    },
    maquette: {
      ...maquette,
      targetPath: maquette.targetPath ?? '',
      envName: maquette.envName ?? 'demo',
      appName: maquette.appName ?? 'prod',
    },
    gedixConfig: normalizeGedixConfig({
      ...gedixConfig,
      fqdn: gedixConfig.fqdn ?? '',
      port: gedixConfig.port ?? 80,
      services: gedixConfig.services ?? {},
      connectors: gedixConfig.connectors ?? {},
    }),
    runtime: {
      ...runtime,
      debugTargets: runtime.debugTargets ?? [],
      openConsole: runtime.openConsole ?? true,
    },
    pipeline: config.pipeline ?? [],
  };
}

function normalizeGedixConfig(gedixConfig: V10Config['gedixConfig']): V10Config['gedixConfig'] {
  const services = { ...(gedixConfig.services ?? {}) };
  for (const [name, service] of Object.entries(services)) {
    if (service.dbType === 'sqlite' && !service.dbDsn?.trim() && Object.keys(service.extraKeys ?? {}).length === 0) {
      delete services[name];
    }
  }
  return { ...gedixConfig, services };
}

function validateConfig(config: V10Config): string {
  if (!config.name.trim()) {
    return m.errors.nameRequired;
  }
  if (!config.product.trim()) {
    return m.errors.productRequired;
  }
  if (!Number.isFinite(config.gedixConfig.port) || config.gedixConfig.port < 0 || config.gedixConfig.port > 65535) {
    return m.errors.portInvalid;
  }
  if (config.pipeline.some((step) => !step.action.trim())) {
    return m.errors.pipelineActionRequired;
  }
  if (hasDuplicateConnector(Object.keys(config.gedixConfig.connectors ?? {}).map((name) => ({ id: name, name, rawConfig: '' })))) {
    return m.duplicateConnector;
  }
  return '';
}

function formatDate(value: string) {
  return new Date(value).toLocaleString('fr-FR');
}

function extraKeyRowsFromService(service: ServiceDBConfig): ExtraKeyRow[] {
  return Object.entries(service.extraKeys ?? {}).map(([key, value]) => ({
    id: makeID(),
    key,
    value,
  }));
}

function filepathExt(filename: string) {
  const index = filename.lastIndexOf('.');
  return index >= 0 ? filename.slice(index) : '';
}

function stringsEqual(left: string, right: string) {
  return left.localeCompare(right, undefined, { sensitivity: 'accent' }) === 0;
}

function makeID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function connectorRowsFromConfig(config: V10Config): ConnectorFormRow[] {
  return Object.entries(config.gedixConfig.connectors ?? {}).map(([name, connector]) => ({
    id: makeID(),
    name,
    rawConfig: connector.rawConfig,
  }));
}

function hasDuplicateConnector(rows: ConnectorFormRow[]) {
  const seen = new Set<string>();
  for (const row of rows) {
    const name = row.name.trim().toLowerCase();
    if (!name) {
      continue;
    }
    if (seen.has(name)) {
      return true;
    }
    seen.add(name);
  }
  return false;
}

function delay(ms: number) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

class LocalErrorBoundary extends React.Component<{ children: React.ReactNode }, { error: string }> {
  state = { error: '' };

  static getDerivedStateFromError(error: Error) {
    return { error: error.message };
  }

  render() {
    if (this.state.error) {
      return <p className="error">{this.state.error}</p>;
    }
    return this.props.children;
  }
}
