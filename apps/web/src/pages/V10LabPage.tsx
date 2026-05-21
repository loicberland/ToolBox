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

const serviceNames = ['webserver', 'auth', 'filestore', 'entreprise', 'etl', 'dnc', 'reactor', 'config'];
const noDatabaseServices = new Set(['webserver', 'reactor']);
const tabs = ['Général', 'Configuration Gedix', 'Services', 'Connecteurs', 'Pipeline', 'Exécution / logs', 'JSON'] as const;
type Tab = typeof tabs[number];

export function V10LabPage() {
  const [products, setProducts] = useState<V10Product[]>([]);
  const [actions, setActions] = useState<V10Action[]>([]);
  const [templates, setTemplates] = useState<DBTemplate[]>([]);
  const [maquettes, setMaquettes] = useState<MaquetteSummary[]>([]);
  const [selectedName, setSelectedName] = useState('');
  const [config, setConfig] = useState<V10Config | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('Général');
  const [showCreate, setShowCreate] = useState(false);
  const [draft, setDraft] = useState(() => defaultConfig());
  const [jsonText, setJsonText] = useState('');
  const [logs, setLogs] = useState<LogSummary[]>([]);
  const [selectedLog, setSelectedLog] = useState('');
  const [busy, setBusy] = useState(false);
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
      void loadActions(config.product);
    }
  }, [config?.name, config?.product]);

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

  async function reloadList() {
    const items = await v10LabApi.listMaquettes();
    setMaquettes(items);
  }

  async function openMaquette(name: string) {
    await run(async () => {
      const loaded = normalizeConfig(await v10LabApi.getMaquette(name));
      setSelectedName(loaded.name);
      setConfig(loaded);
      setActiveTab('Général');
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
      setMessage('Maquette enregistrée.');
    });
  }

  async function saveCurrent() {
    if (!config) {
      return;
    }
    const validation = validateConfig(config);
    if (validation) {
      setError(validation);
      return;
    }
    await run(async () => {
      await v10LabApi.updateMaquette(config.name, normalizeConfig(config));
      await reloadList();
      setMessage('Sauvegarde effectuée.');
    });
  }

  async function validateCurrent(name = selectedName) {
    if (!name) {
      return;
    }
    await run(async () => {
      const result = await v10LabApi.validateMaquette(name);
      setExecution(result);
      setMessage('Validation OK.');
    });
  }

  async function runCurrent(name = selectedName) {
    if (!name) {
      return;
    }
    await run(async () => {
      const result = await v10LabApi.runMaquette(name);
      setExecution(result);
      await reloadList();
      await refreshLogs(name);
      setMessage('Lancement terminé.');
    });
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
      setMessage('Enregistrement supprimé. Le dossier Gedix physique n’a pas été supprimé.');
    });
  }

  async function killGXProcesses() {
    await run(async () => {
      const result = await v10LabApi.killGXProcesses();
      setConfirmKill(false);
      setExecution(result);
      setMessage('Taskkill gx-* terminé.');
    });
  }

  async function saveJSON() {
    if (!config) {
      return;
    }
    try {
      const parsed = normalizeConfig(JSON.parse(jsonText) as V10Config);
      setConfig(parsed);
      await v10LabApi.updateMaquette(config.name, parsed);
      await reloadList();
      setMessage('JSON sauvegardé.');
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'JSON invalide');
    }
  }

  async function run(task: () => Promise<void>) {
    setBusy(true);
    setError('');
    setMessage('');
    try {
      await task();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur inconnue');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="workspace v10-lab-workspace">
      <header className="page-header">
        <div className="page-title-group">
          <p className="page-eyebrow">V10 Lab</p>
          <h2>Générateur de maquettes V10</h2>
          <p>Gérez les maquettes, leur configuration Gedix et les pipelines système existants.</p>
        </div>
        <div className="page-actions">
          <Button type="button" onClick={() => setShowCreate((value) => !value)}>Nouvelle maquette</Button>
        </div>
      </header>

      {error && <p className="error whitespace">{error}</p>}
      {message && <p className="info-message">{message}</p>}

      {showCreate && (
        <section className="ui-card v10-section">
          <div className="ui-card-header">
            <h3>Nouvelle maquette</h3>
          </div>
          <MaquetteGeneralForm config={draft} products={products} onChange={setDraft} creating />
          <div className="button-row end">
            <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>Annuler</Button>
            <Button type="button" onClick={() => void createMaquette()} disabled={busy}>Créer</Button>
          </div>
        </section>
      )}

      <section className="ui-card v10-section">
        <div className="ui-card-header">
          <h3>Maquettes enregistrées</h3>
          <Button type="button" variant="secondary" size="sm" onClick={() => void reloadList()} disabled={busy}>Rafraîchir</Button>
        </div>
        {maquettes.length === 0 ? (
          <div className="empty-state">
            <h3>Aucune maquette enregistrée.</h3>
          </div>
        ) : (
          <div className="v10-table">
            <div className="v10-table-head">
              <span>Nom</span>
              <span>Produit</span>
              <span>Cible</span>
              <span>Disque</span>
              <span>Dernier lancement</span>
              <span>Actions</span>
            </div>
            {maquettes.map((item) => (
              <div className={`v10-table-row ${item.name === selectedName ? 'active' : ''}`} key={item.name}>
                <strong>{item.name}</strong>
                <span>{item.product}</span>
                <span className="truncate">{item.targetPath || '-'}</span>
                <span>{item.existsOnDisk ? 'Présent' : 'Absent'}</span>
                <span>{item.lastRunAt ? `${formatDate(item.lastRunAt)} (${item.lastStatus ?? 'unknown'})` : '-'}</span>
                <div className="button-row">
                  <Button type="button" size="sm" variant="secondary" onClick={() => void openMaquette(item.name)}>Ouvrir</Button>
                  <Button type="button" size="sm" variant="secondary" onClick={() => void validateCurrent(item.name)}>Valider</Button>
                  <Button type="button" size="sm" onClick={() => void runCurrent(item.name)}>Lancer</Button>
                  <Button type="button" size="sm" variant="danger" onClick={() => setConfirmDelete(item.name)}>Supprimer</Button>
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
              <Button type="button" variant="secondary" onClick={() => void openMaquette(config.name)} disabled={busy}>Recharger</Button>
              <Button type="button" onClick={() => void saveCurrent()} disabled={busy}>Sauvegarder</Button>
            </div>
          </div>

          <div className="v10-tabs">
            {tabs.map((tab) => (
              <button type="button" key={tab} className={tab === activeTab ? 'active' : ''} onClick={() => setActiveTab(tab)}>
                {tab}
              </button>
            ))}
          </div>

          {activeTab === 'Général' && <MaquetteGeneralForm config={config} products={products} onChange={setConfig} />}
          {activeTab === 'Configuration Gedix' && <GedixForm config={config} onChange={setConfig} />}
          {activeTab === 'Services' && <ServicesForm config={config} templates={templates} onChange={setConfig} />}
          {activeTab === 'Connecteurs' && <ConnectorsForm config={config} onChange={setConfig} />}
          {activeTab === 'Pipeline' && <PipelineBuilder config={config} actions={actions} onChange={setConfig} />}
          {activeTab === 'Exécution / logs' && (
            <ExecutionPanel
              busy={busy}
              execution={execution}
              logs={logs}
              selectedLog={selectedLog}
              onValidate={() => void validateCurrent()}
              onRun={() => void runCurrent()}
              onKill={() => setConfirmKill(true)}
              onRefreshLogs={() => void refreshLogs()}
              onReadLog={(logFile) => void readLog(logFile)}
            />
          )}
          {activeTab === 'JSON' && (
            <div className="v10-json-panel">
              <textarea value={jsonText} onChange={(event) => setJsonText(event.target.value)} spellCheck={false} />
              <div className="button-row end">
                <Button type="button" variant="secondary" onClick={() => void navigator.clipboard?.writeText(jsonText)}>Copier</Button>
                <Button type="button" onClick={() => void saveJSON()}>Sauvegarder JSON</Button>
              </div>
            </div>
          )}
        </section>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        title="Supprimer la maquette"
        message="Cette suppression retire seulement l’enregistrement V10 Lab. Le dossier Gedix physique ne sera pas supprimé."
        confirmLabel="Supprimer"
        onCancel={() => setConfirmDelete(null)}
        onConfirm={() => confirmDelete && void deleteMaquette(confirmDelete)}
      />
      <ConfirmDialog
        open={confirmKill}
        title="Taskkill gx-*"
        message="Cette action tue tous les processus gx-* sur la machine. Continuer ?"
        confirmLabel="Continuer"
        onCancel={() => setConfirmKill(false)}
        onConfirm={() => void killGXProcesses()}
      />
    </div>
  );
}

function MaquetteGeneralForm({ config, products, onChange, creating = false }: {
  config: V10Config;
  products: V10Product[];
  onChange: (config: V10Config) => void;
  creating?: boolean;
}) {
  return (
    <div className="form-grid v10-form-grid">
      <label>Nom
        <input value={config.name} disabled={!creating} onChange={(event) => onChange({ ...config, name: event.target.value })} />
      </label>
      <label>Produit
        <select value={config.product} onChange={(event) => onChange({ ...config, product: event.target.value })}>
          {products.map((product) => <option value={product.id} key={product.id}>{product.name}</option>)}
        </select>
      </label>
      <label>Chemin ZIP release
        <input value={config.release.zipPath} onChange={(event) => onChange({ ...config, release: { ...config.release, zipPath: event.target.value } })} />
      </label>
      <label>Dossier cible
        <input value={config.maquette.targetPath} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, targetPath: event.target.value } })} />
      </label>
      <label>Environnement
        <input value={config.maquette.envName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, envName: event.target.value } })} />
      </label>
      <label>Application
        <input value={config.maquette.appName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, appName: event.target.value } })} />
      </label>
      <label className="checkbox-row">
        <input type="checkbox" checked={config.release.overwrite} onChange={(event) => onChange({ ...config, release: { ...config.release, overwrite: event.target.checked } })} />
        Overwrite
      </label>
      {creating && <GedixForm config={config} onChange={onChange} compact />}
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
        const service = config.gedixConfig.services[name];
        const enabled = Boolean(service);
        return (
          <div className="v10-service-row" key={name}>
            <div>
              <strong>{name}</strong>
              {disabled && <p className="muted">Pas de base de données pour ce service.</p>}
            </div>
            {!disabled && (
              <>
                <label className="checkbox-row">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(event) => updateService(name, event.target.checked ? { dbType: 'sqlite', dbDsn: '', extraKeys: {} } : null)}
                  />
                  Configurer DB
                </label>
                {enabled && (
                  <div className="v10-service-config">
                    <label>Type DB
                      <select value={service?.dbType ?? ''} onChange={(event) => updateService(name, { ...service!, dbType: event.target.value, dbDsn: event.target.value === 'sqlite' ? '' : service?.dbDsn ?? '' })}>
                        {['sqlite', 'mysql', 'postgres', 'mssql', 'oracle'].map((type) => <option key={type} value={type}>{type}</option>)}
                      </select>
                    </label>
                    {service?.dbType !== 'sqlite' && (
                      <label>DSN
                        <input value={service?.dbDsn ?? ''} onChange={(event) => updateService(name, { ...service!, dbDsn: event.target.value })} />
                      </label>
                    )}
                    <label>Template DSN
                      <select value="" onChange={(event) => updateService(name, { ...service!, dbDsn: event.target.value })}>
                        <option value="">Insérer un template</option>
                        {templates.filter((template) => template.template).map((template) => <option key={template.type} value={template.template}>{template.type}</option>)}
                      </select>
                    </label>
                    <ExtraKeysEditor service={service!} onChange={(next) => updateService(name, next)} />
                  </div>
                )}
              </>
            )}
          </div>
        );
      })}
    </div>
  );
}

function ExtraKeysEditor({ service, onChange }: { service: ServiceDBConfig; onChange: (service: ServiceDBConfig) => void }) {
  const entries = Object.entries(service.extraKeys ?? {});
  return (
    <div className="v10-extra-keys">
      <div className="section-header compact">
        <h4>Extra keys</h4>
        <Button type="button" size="sm" variant="secondary" onClick={() => onChange({ ...service, extraKeys: { ...(service.extraKeys ?? {}), '': '' } })}>Ajouter</Button>
      </div>
      {entries.map(([key, value], index) => (
        <div className="v10-key-row" key={`${key}-${index}`}>
          <input value={key} placeholder="clé" onChange={(event) => replaceExtraKey(service, key, event.target.value, value, onChange)} />
          <input value={value} placeholder="valeur" onChange={(event) => onChange({ ...service, extraKeys: { ...(service.extraKeys ?? {}), [key]: event.target.value } })} />
          <Button type="button" size="sm" variant="danger" onClick={() => removeExtraKey(service, key, onChange)}>Supprimer</Button>
        </div>
      ))}
    </div>
  );
}

function ConnectorsForm({ config, onChange }: { config: V10Config; onChange: (config: V10Config) => void }) {
  const connectors = config.gedixConfig.connectors;
  const update = (name: string, nextName: string, rawConfig: string) => {
    const next = { ...connectors };
    delete next[name];
    if (nextName.trim()) {
      next[nextName.trim()] = { rawConfig };
    }
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, connectors: next } });
  };
  return (
    <div className="v10-connector-list">
      <p className="readonly-notice">Le nom du connecteur doit correspondre exactement à la section du gedix.cfg : [environments.&lt;env&gt;.applications.&lt;app&gt;.connectors.&lt;nomConnecteur&gt;]</p>
      {Object.keys(connectors).map((name) => (
        <div className="v10-connector-row" key={name}>
          <input value={name} onChange={(event) => update(name, event.target.value, connectors[name].rawConfig)} />
          <textarea value={connectors[name].rawConfig} onChange={(event) => update(name, name, event.target.value)} />
          <Button type="button" variant="danger" size="sm" onClick={() => update(name, '', '')}>Supprimer</Button>
        </div>
      ))}
      <Button type="button" variant="secondary" onClick={() => update('', `connector-${Object.keys(connectors).length + 1}`, '')}>Ajouter connecteur</Button>
    </div>
  );
}

function PipelineBuilder({ config, actions, onChange }: { config: V10Config; actions: V10Action[]; onChange: (config: V10Config) => void }) {
  const byID = useMemo(() => Object.fromEntries(actions.map((action) => [action.id, action])), [actions]);
  const updateStep = (index: number, step: PipelineStep) => {
    onChange({ ...config, pipeline: config.pipeline.map((item, itemIndex) => itemIndex === index ? step : item) });
  };
  const move = (index: number, direction: -1 | 1) => {
    const next = [...config.pipeline];
    const target = index + direction;
    if (target < 0 || target >= next.length) {
      return;
    }
    [next[index], next[target]] = [next[target], next[index]];
    onChange({ ...config, pipeline: next });
  };
  return (
    <div className="v10-pipeline">
      <div className="button-row">
        <Button type="button" variant="secondary" onClick={() => onChange({ ...config, pipeline: [...config.pipeline, { action: actions[0]?.id ?? '', label: actions[0]?.label ?? '', params: {} }] })}>Ajouter une étape</Button>
      </div>
      {config.pipeline.map((step, index) => {
        const action = byID[step.action];
        return (
          <div className="v10-pipeline-step" key={`${step.action}-${index}`}>
            <div className="v10-step-order">{index + 1}</div>
            <div className="v10-step-body">
              <div className="form-grid v10-form-grid">
                <label>Action
                  <select value={step.action} onChange={(event) => {
                    const selected = byID[event.target.value];
                    updateStep(index, { action: event.target.value, label: selected?.label ?? '', params: {} });
                  }}>
                    <option value="">Choisir une action</option>
                    {actions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
                  </select>
                </label>
                <label>Label
                  <input value={step.label} onChange={(event) => updateStep(index, { ...step, label: event.target.value })} />
                </label>
              </div>
              {action && action.fields.length > 0 && (
                <div className="form-grid v10-form-grid">
                  {action.fields.map((field) => (
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
                <Button type="button" size="sm" variant="secondary" onClick={() => move(index, -1)}>Monter</Button>
                <Button type="button" size="sm" variant="secondary" onClick={() => move(index, 1)}>Descendre</Button>
                <Button type="button" size="sm" variant="danger" onClick={() => onChange({ ...config, pipeline: config.pipeline.filter((_, itemIndex) => itemIndex !== index) })}>Supprimer</Button>
              </div>
            </div>
          </div>
        );
      })}
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

function ExecutionPanel({ busy, execution, logs, selectedLog, onValidate, onRun, onKill, onRefreshLogs, onReadLog }: {
  busy: boolean;
  execution: ExecutionResponse | null;
  logs: LogSummary[];
  selectedLog: string;
  onValidate: () => void;
  onRun: () => void;
  onKill: () => void;
  onRefreshLogs: () => void;
  onReadLog: (logFile: string) => void;
}) {
  return (
    <div className="v10-execution">
      <div className="button-row">
        <Button type="button" variant="secondary" onClick={onValidate} disabled={busy}>Valider</Button>
        <Button type="button" onClick={onRun} disabled={busy}>Lancer</Button>
        <Button type="button" variant="danger" onClick={onKill} disabled={busy}>Taskkill gx-*</Button>
        <Button type="button" variant="secondary" onClick={onRefreshLogs} disabled={busy}>Rafraîchir logs</Button>
      </div>
      {execution && (
        <pre className="v10-output">{execution.errors?.length ? execution.errors.join('\n') : execution.output || execution.status}</pre>
      )}
      <div className="v10-log-layout">
        <div className="v10-log-list">
          {logs.length === 0 ? <p className="muted">Aucun log disponible.</p> : logs.map((log) => (
            <button type="button" key={log.name} onClick={() => onReadLog(log.name)}>
              <strong>{log.name}</strong>
              <span>{formatDate(log.modifiedAt)}</span>
            </button>
          ))}
        </div>
        <pre className="v10-output">{selectedLog || 'Sélectionnez un log.'}</pre>
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
    pipeline: [
      { action: 'create-env', label: 'Créer la maquette', params: {} },
      { action: 'configure-gedix-cfg', label: 'Configurer gedix.cfg', params: {} },
    ],
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
    gedixConfig: {
      ...gedixConfig,
      fqdn: gedixConfig.fqdn ?? '',
      port: gedixConfig.port ?? 80,
      services: gedixConfig.services ?? {},
      connectors: gedixConfig.connectors ?? {},
    },
    runtime: {
      ...runtime,
      debugTargets: runtime.debugTargets ?? [],
      openConsole: runtime.openConsole ?? true,
    },
    pipeline: config.pipeline ?? [],
  };
}

function validateConfig(config: V10Config): string {
  if (!config.name.trim()) {
    return 'Nom obligatoire.';
  }
  if (!config.product.trim()) {
    return 'Produit obligatoire.';
  }
  if (!Number.isFinite(config.gedixConfig.port) || config.gedixConfig.port < 0 || config.gedixConfig.port > 65535) {
    return 'Port numérique invalide.';
  }
  if (config.pipeline.some((step) => !step.action.trim())) {
    return 'Chaque étape de pipeline doit avoir une action.';
  }
  return '';
}

function replaceExtraKey(service: ServiceDBConfig, oldKey: string, nextKey: string, value: string, onChange: (service: ServiceDBConfig) => void) {
  const next = { ...(service.extraKeys ?? {}) };
  delete next[oldKey];
  next[nextKey] = value;
  onChange({ ...service, extraKeys: next });
}

function removeExtraKey(service: ServiceDBConfig, key: string, onChange: (service: ServiceDBConfig) => void) {
  const next = { ...(service.extraKeys ?? {}) };
  delete next[key];
  onChange({ ...service, extraKeys: next });
}

function formatDate(value: string) {
  return new Date(value).toLocaleString('fr-FR');
}
