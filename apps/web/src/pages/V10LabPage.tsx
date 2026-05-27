import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  DBTemplate,
  ExecutionResponse,
  ConnectorConfig,
  LogSummary,
  MaquetteGroup,
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

const m = messages.v10Lab;
const tabs = [m.tabs.general, m.tabs.gedix, m.tabs.services, m.tabs.connectors, m.tabs.pipeline, m.tabs.execution, m.tabs.json] as const;
const systemPipelineActions = new Set(['create-env', 'configure-gedix-cfg', 'start-maquette', 'stop-maquette', 'start-services', 'stop-services', 'kill-gx-processes', 'update-env']);
type Tab = typeof tabs[number];
type RunState = 'idle' | 'running' | 'success' | 'failed';
type ConnectorFormRow = {
  id: string;
  name: string;
  module: string;
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
  const [groups, setGroups] = useState<MaquetteGroup[]>([]);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});
  const [newGroupName, setNewGroupName] = useState('');
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
  const [confirmUpdate, setConfirmUpdate] = useState(false);
  const [execution, setExecution] = useState<ExecutionResponse | null>(null);
  const currentMaquetteRef = useRef<HTMLElement | null>(null);
  const scrollToCurrentMaquetteAfterTabChange = useRef(false);

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
    if (!scrollToCurrentMaquetteAfterTabChange.current) {
      return;
    }
    scrollToCurrentMaquetteAfterTabChange.current = false;
    window.requestAnimationFrame(() => {
      currentMaquetteRef.current?.scrollIntoView({ block: 'start', inline: 'nearest', behavior: 'auto' });
    });
  }, [activeTab]);

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
  const currentProduct = productFor(config?.product ?? draft.product, products);

  async function loadInitial() {
    await run(async () => {
      const [productItems, templateItems, maquetteItems, groupItems] = await Promise.all([
        v10LabApi.products(),
        v10LabApi.dbTemplates(),
        v10LabApi.listMaquettes(),
        v10LabApi.listMaquetteGroups(),
      ]);
      setProducts(productItems);
      setTemplates(templateItems);
      setMaquettes(maquetteItems);
      setGroups(groupItems);
      const product = productItems.find((item) => item.id === DEFAULT_V10_PRODUCT_ID) ?? productItems[0];
      setDraft(defaultConfig(product?.id, product));
      await loadActions(product?.id ?? DEFAULT_V10_PRODUCT_ID);
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
    const [items, groupItems] = await Promise.all([v10LabApi.listMaquettes(), v10LabApi.listMaquetteGroups()]);
    setMaquettes(items);
    setGroups(groupItems);
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

  async function createMaquette(groupName = draft.groupName ?? '') {
    const validation = validateConfig(draft);
    if (validation) {
      setError(validation);
      return;
    }
    await run(async () => {
      const next = normalizeConfig({ ...draft, groupName });
      await v10LabApi.createMaquette(next);
      setShowCreate(false);
      const product = productFor(DEFAULT_V10_PRODUCT_ID, products);
      setDraft(defaultConfig(product.id, product));
      await reloadList();
      if (next.groupName) {
        setOpenGroups((current) => ({ ...current, [next.groupName!]: true }));
      }
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
      const oldName = config.name;
      const next = normalizeConfig(await v10LabApi.updateMaquette(selectedName || oldName, normalizeConfig(config)));
      setConfig(next);
      setSelectedName(next.name);
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
    scrollToCurrentMaquetteAfterTabChange.current = true;
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
    const isUpdate = actionId === 'update-env';
    setExecution({ status: 'running', running: true, output: isUpdate ? m.execution.updateRunning : m.executionRunning });
    await run(async () => {
      const started = await v10LabApi.runAction(name, actionId);
      setExecution(started);
      const result = await pollCurrentRun(name);
      await reloadList();
      await refreshLogs(name);
      setRunState(result.status === 'success' ? 'success' : 'failed');
      setMessage(result.status === 'success' ? (isUpdate ? m.execution.updateFinished : m.executionFinished) : m.executionFailed);
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

  async function runModuleCommand(unitName: string, command: string, name = selectedName) {
    if (!name || runState === 'running') {
      return;
    }
    if (moduleCommandHasUnsafeCharacters(command)) {
      setError(m.moduleCommand.invalidCommand);
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
      const started = await v10LabApi.runModuleCommand(name, unitName, command);
      setExecution(started);
      const result = await pollCurrentRun(name);
      await reloadList();
      await refreshLogs(name);
      setRunState(result.status === 'success' ? 'success' : 'failed');
      setMessage(result.status === 'success' ? m.moduleCommand.consoleOpened : m.executionFailed);
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

  async function updateExistingMaquette() {
    setConfirmUpdate(false);
    await runSystemAction('update-env');
  }

  async function saveJSON() {
    if (!config) {
      return;
    }
    try {
      const parsed = normalizeConfig(JSON.parse(jsonText) as V10Config);
      const saved = normalizeConfig(await v10LabApi.updateMaquette(selectedName || config.name, parsed));
      setConfig(saved);
      setSelectedName(saved.name);
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

  async function importExistingMaquettes() {
    await run(async () => {
      const selected = await v10LabApi.selectFolderPath();
      if (selected.cancelled || !selected.path) {
        return;
      }
      const result = await v10LabApi.importExistingMaquettes(selected.path);
      await reloadList();
      setMessage(`${result.imported.length} maquette(s) importée(s), ${result.skipped.length} ignorée(s).${result.warnings.length ? ` ${result.warnings.join(' ')}` : ''}`);
    });
  }

  async function openCurrentMaquetteURL(name = selectedName) {
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
      const result = await v10LabApi.getMaquetteOpenUrl(name);
      window.open(result.url, '_blank', 'noopener,noreferrer');
      setMessage(`Ouverture de la maquette : ${result.url}`);
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

  async function createGroup() {
    const name = newGroupName.trim();
    if (!name) {
      setError('Nom de groupe obligatoire.');
      return;
    }
    await run(async () => {
      const group = await v10LabApi.createMaquetteGroup(name);
      setNewGroupName('');
      await reloadList();
      setOpenGroups((current) => ({ ...current, [group.name]: true }));
      setMessage('Groupe créé.');
    });
  }

  async function deleteGroup(name: string) {
    await run(async () => {
      await v10LabApi.deleteMaquetteGroup(name);
      await reloadList();
      setMessage('Groupe supprimé.');
    });
  }

  async function scanCfg(file: File, importExistingKeys: boolean) {
    if (!config) {
      return;
    }
    if (!stringsEqual(filepathExt(file.name), '.cfg')) {
      setError(m.errors.cfgOnly);
      return;
    }
    await run(async () => {
      const product = productFor(config.product, products);
      const result = await v10LabApi.scanCfg(config.name, file, config.maquette.envName, config.maquette.appName || product.defaultAppName || 'prod', importExistingKeys);
      const scannedUnits = result.units ?? result.connectors ?? [];
      const unitKey = product.unitKind === 'agent' ? 'agents' : 'connectors';
      const units = { ...(config.gedixConfig[unitKey] ?? {}) };
      for (const unit of scannedUnits) {
        const existing = units[unit.name];
        units[unit.name] = {
          module: unit.module ? unit.module : (existing?.module ?? ''),
          rawConfig: importExistingKeys ? unit.rawConfig : (existing?.rawConfig ?? unit.rawConfig),
        };
      }
      updateConfig({
        ...config,
        maquette: { ...config.maquette, envName: result.envName || config.maquette.envName, appName: result.appName || config.maquette.appName },
        gedixConfig: { ...config.gedixConfig, [unitKey]: units },
      });
      const warnings = result.warnings?.length ? ` ${result.warnings.join(' ')}` : '';
      setMessage(`${scannedUnits.length} ${product.unitPluralLabel} détecté(s).${warnings}`);
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

  function toggleGroup(name: string) {
    setOpenGroups((current) => ({ ...current, [name]: !current[name] }));
  }

  function handleToggleKey(event: React.KeyboardEvent, action: () => void) {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    action();
  }

  const groupedMaquettes = groups.map((group) => ({
    ...group,
    items: maquettes.filter((item) => item.groupName === group.name),
  }));
  const ungroupedMaquettes = maquettes.filter((item) => !item.groupName || !groups.some((group) => group.name === item.groupName));

  return (
    <div className="workspace v10-lab-workspace">
      <header className="page-header">
        <div className="page-title-group">
          <p className="page-eyebrow">{m.title}</p>
          <h2>{m.subtitle}</h2>
          <p>{m.description}</p>
        </div>
        <div className="page-actions">
          <Button type="button" variant="secondary" onClick={() => void importExistingMaquettes()} disabled={busy}>Scanner maquettes existantes</Button>
          <Button type="button" onClick={() => setShowCreate((value) => !value)}>{m.newMaquette}</Button>
        </div>
      </header>


      {showCreate && (
        <section className="ui-card v10-section">
          <div className="ui-card-header">
            <h3>{m.newMaquette}</h3>
          </div>
          <MaquetteGeneralForm config={draft} products={products} groups={groups} defaultTargetPath={defaultTargetPath} onChange={setDraft} onSelectZip={selectReleaseZip} creating />
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
        <div className="v10-file-row">
          <input placeholder="Nom du groupe" value={newGroupName} onChange={(event) => setNewGroupName(event.currentTarget.value)} />
          <Button type="button" variant="secondary" size="sm" onClick={() => void createGroup()} disabled={busy || !newGroupName.trim()}>Créer un groupe</Button>
        </div>
        {maquettes.length === 0 && groups.length === 0 ? (
          <div className="empty-state">
            <h3>{m.noMaquette}</h3>
          </div>
        ) : (
          <>
            <div className="v10-group-list">
              {groupedMaquettes.map((group) => (
                <div className="v10-group" key={group.name}>
                  <div
                    className="v10-group-header clickable"
                    role="button"
                    tabIndex={0}
                    onClick={() => toggleGroup(group.name)}
                    onKeyDown={(event) => handleToggleKey(event, () => toggleGroup(group.name))}
                  >
                    <span className="v10-chevron" aria-hidden="true">{openGroups[group.name] ? '▾' : '▸'}</span>
                    <strong>{group.name}</strong>
                    <span className="muted">{group.items.length}</span>
                    <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); setDraft({ ...defaultConfig(currentProduct.id, currentProduct), groupName: group.name }); setShowCreate(true); setOpenGroups((current) => ({ ...current, [group.name]: true })); }}>Ajouter une maquette</Button>
                    <Button type="button" size="sm" variant="danger" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); void deleteGroup(group.name); }} disabled={group.items.length > 0}>Supprimer</Button>
                  </div>
                  {openGroups[group.name] && <MaquetteList items={group.items} selectedName={selectedName} onToggle={toggleMaquette} />}
                </div>
              ))}
            </div>
            <h4>Sans groupe</h4>
            <MaquetteList items={ungroupedMaquettes} selectedName={selectedName} onToggle={toggleMaquette} />
          </>
        )}
      </section>

      {error && <p className="error whitespace">{error}</p>}
      {message && <p className="info-message">{message}</p>}
      
      {config && (
        <section ref={currentMaquetteRef} className="ui-card v10-section">
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
                {tab === m.tabs.connectors ? (currentProduct.unitKind === 'agent' ? m.units.agents : m.units.connectors) : tab}
              </button>
            ))}
          </div>

          {activeTab === m.tabs.general && <MaquetteGeneralForm config={config} products={products} groups={groups} defaultTargetPath={defaultTargetPath} onChange={updateConfig} onSelectZip={selectReleaseZip} />}
          {activeTab === m.tabs.gedix && <GedixForm config={config} onChange={updateConfig} />}
          {activeTab === m.tabs.services && <ServicesForm config={config} product={currentProduct} templates={templates} onChange={updateConfig} />}
          {activeTab === m.tabs.connectors && <ConnectorsForm config={config} product={currentProduct} onChange={updateConfig} onScanCfg={(file, importExistingKeys) => void scanCfg(file, importExistingKeys)} />}
          {activeTab === m.tabs.pipeline && (
            <LocalErrorBoundary>
              <ApiTokenEditor maquetteName={config.name} disabled={busy} />
              <PipelineBuilder config={config} actions={actions} onChange={updateConfig} />
            </LocalErrorBoundary>
          )}
          {activeTab === m.tabs.execution && (
            <ExecutionPanel
              config={config}
              product={currentProduct}
              busy={busy}
              runState={runState}
              execution={execution}
              logs={logs}
              selectedLog={selectedLog}
              onConfigChange={updateConfig}
              onCreate={() => void runSystemAction('create-env')}
              onUpdate={() => setConfirmUpdate(true)}
              onConfigure={() => void runSystemAction('configure-gedix-cfg')}
              onStart={() => void runSystemAction('start-maquette')}
              onOpenMaquette={() => void openCurrentMaquetteURL()}
              onRunPipeline={() => void runCurrent()}
              onRunModuleCommand={(unitName, command) => void runModuleCommand(unitName, command)}
              onKill={() => setConfirmKill(true)}
              onRefreshLogs={() => void refreshLogs()}
              onReadLog={(logFile) => void readLog(logFile)}
            />
          )}
          {activeTab === m.tabs.json && (
            <div className="v10-json-panel">
              <textarea value={jsonText} onChange={(event) => setJsonText(event.currentTarget.value)} spellCheck={false} />
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
      <ConfirmDialog
        open={confirmUpdate}
        title={m.execution.updateMaquette}
        message={m.execution.updateMaquetteConfirm}
        confirmLabel="Continuer"
        onCancel={() => setConfirmUpdate(false)}
        onConfirm={() => void updateExistingMaquette()}
      />
    </div>
  );
}

function MaquetteList({ items, selectedName, onToggle }: { items: MaquetteSummary[]; selectedName: string; onToggle: (name: string) => Promise<void> }) {
  const handleKeyDown = (event: React.KeyboardEvent, name: string) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    void onToggle(name);
  };

  if (items.length === 0) {
    return <p className="muted">Aucune maquette.</p>;
  }
  return (
    <div className="v10-table">
      <div className="v10-table-head">
        <span>{m.name}</span>
        <span>{m.product}</span>
        <span>{m.installed}</span>
        <span>{m.actions}</span>
      </div>
      {items.map((item) => (
        <div
          className={`v10-table-row clickable ${item.name === selectedName ? 'active' : ''}`}
          key={item.name}
          role="button"
          tabIndex={0}
          onClick={() => void onToggle(item.name)}
          onKeyDown={(event) => handleKeyDown(event, item.name)}
        >
          <strong>{item.name}</strong>
          <span>{item.product}</span>
          <span>{item.existsOnDisk ? m.yes : m.no}</span>
          <div className="button-row">
            <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); void onToggle(item.name); }}>{item.name === selectedName ? m.close : m.open}</Button>
          </div>
        </div>
      ))}
    </div>
  );
}

function MaquetteGeneralForm({ config, products, groups, defaultTargetPath, onChange, onSelectZip, creating = false }: {
  config: V10Config;
  products: V10Product[];
  groups: MaquetteGroup[];
  defaultTargetPath: string;
  onChange: (config: V10Config) => void;
  onSelectZip: (config: V10Config, onChange: (config: V10Config) => void) => void;
  creating?: boolean;
}) {
  const changeProduct = (productId: string) => {
    if (!creating && productId !== config.product && !window.confirm(m.productChangeWarning)) {
      return;
    }
    const product = productFor(productId, products);
    const shouldSetDefaultApp = !config.maquette.appName.trim() || (creating && config.maquette.appName === productFor(config.product, products).defaultAppName);
    onChange({
      ...config,
      product: product.id,
      maquette: {
        ...config.maquette,
        appName: shouldSetDefaultApp ? product.defaultAppName : config.maquette.appName,
      },
    });
  };
  return (
    <div className="form-grid v10-form-grid">
      <label className="span-2">{m.product}
        <select value={config.product} onChange={(event) => changeProduct(event.currentTarget.value)}>
          {products.map((product) => <option value={product.id} key={product.id}>{product.label || product.name}</option>)}
        </select>
      </label>
      <label>{m.name}
        <input value={config.name} onChange={(event) => onChange({ ...config, name: event.currentTarget.value })} />
      </label>
      <label>Groupe
        <select value={config.groupName ?? ''} onChange={(event) => onChange({ ...config, groupName: event.currentTarget.value })}>
          <option value="">Sans groupe</option>
          {groups.map((group) => <option value={group.name} key={group.name}>{group.name}</option>)}
        </select>
      </label>
      <label>{m.releaseZip}
        <div className="v10-file-row">
          <Button type="button" variant="secondary" size="sm" onClick={() => onSelectZip(config, onChange)}>
            {m.selectZip}
          </Button>
          <input placeholder={m.manualZip} value={config.release.zipPath} onChange={(event) => onChange({ ...config, release: { ...config.release, zipPath: event.currentTarget.value } })} />
        </div>
      </label>
      <label>{m.targetPath}
        <input placeholder={m.targetPlaceholder.replace('{{path}}', defaultTargetPath)} value={config.maquette.targetPath} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, targetPath: event.currentTarget.value } })} />
      </label>
      <label>{m.appName}
        <input value={config.maquette.appName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, appName: event.currentTarget.value } })} />
      </label>
      <label>{m.envName}
        <input value={config.maquette.envName} onChange={(event) => onChange({ ...config, maquette: { ...config.maquette, envName: event.currentTarget.value } })} />
      </label>
      {!creating && (
        <>
          <label className="checkbox-row span-2">
            <input type="checkbox" checked={config.release.overwrite} onChange={(event) => onChange({ ...config, release: { ...config.release, overwrite: event.currentTarget.checked } })} />
            {m.overwriteLabel}
          </label>
          <p className="muted v10-help-text span-2">{m.overwriteHelp}</p>
        </>
      )}
      {creating && <GedixForm config={config} onChange={onChange} compact />}
    </div>
  );
}

function DebugTargetsEditor({ config, product, onChange }: { config: V10Config; product: V10Product; onChange: (config: V10Config) => void }) {
  const [selected, setSelected] = useState('');
  const [customTarget, setCustomTarget] = useState('');
  const [customFlag, setCustomFlag] = useState('');
  const [customError, setCustomError] = useState('');
  const options = [...product.services.map((service) => service.name), ...Object.keys(unitsForConfig(config, product))]
    .filter((item, index, items) => items.indexOf(item) === index)
    .filter((item) => !config.runtime.debugTargets.includes(item));
  const allTargets = [...product.services.map((service) => service.name), ...Object.keys(unitsForConfig(config, product))]
    .filter((item, index, items) => items.indexOf(item) === index);
  const debugTargetFlags: Record<string, string[]> = config.runtime.debugTargetFlags ?? {};
  const addCustomFlag = () => {
    const target = customTarget.trim();
    const flag = normalizeDebugFlag(customFlag);
    if (!target || !flag) {
      setCustomError('Cible et flag requis.');
      return;
    }
    if (debugFlagHasUnsafeCharacters(customFlag)) {
      setCustomError('Flag invalide.');
      return;
    }
    const current = debugTargetFlags[target] ?? [];
    if (!current.includes(flag)) {
      onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: { ...debugTargetFlags, [target]: [...current, flag] } } });
    }
    setCustomFlag('');
    setCustomError('');
  };
  const removeCustomFlag = (target: string, flag: string) => {
    const next = { ...debugTargetFlags };
    const flags = (next[target] ?? []).filter((item) => item !== flag);
    if (flags.length > 0) {
      next[target] = flags;
    } else {
      delete next[target];
    }
    onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: next } });
  };
  return (
    <div className="v10-debug-targets">
      <div className="v10-file-row">
        <select value={selected} onChange={(event) => setSelected(event.currentTarget.value)}>
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
      <div className="v10-file-row">
        <select value={customTarget} onChange={(event) => setCustomTarget(event.currentTarget.value)}>
          <option value="">Cible debug sur mesure</option>
          {allTargets.map((item) => <option key={item} value={item}>{item}</option>)}
        </select>
        <input value={customFlag} placeholder="clef ou --clef" onChange={(event) => setCustomFlag(event.currentTarget.value)} />
        <Button type="button" variant="secondary" size="sm" disabled={!customTarget || !customFlag.trim()} onClick={addCustomFlag}>Ajouter</Button>
      </div>
      <p className="muted">Ces clés lancent la cible séparément. Le mode debug --debug -v2 est ajouté uniquement si la cible est aussi sélectionnée en debug.</p>
      {customError && <p className="error">{customError}</p>}
      <div className="button-row">
        {debugFlagButtons(debugTargetFlags, removeCustomFlag)}
      </div>
    </div>
  );
}

function GedixForm({ config, onChange, compact = false }: { config: V10Config; onChange: (config: V10Config) => void; compact?: boolean }) {
  const content = (
    <>
      <label>FQDN
        <input value={config.gedixConfig.fqdn} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, fqdn: event.currentTarget.value } })} />
      </label>
      <label>Port
        <input type="number" min={0} max={65535} value={config.gedixConfig.port} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, port: Number(event.currentTarget.value) } })} />
      </label>
    </>
  );
  return compact ? content : <div className="form-grid v10-form-grid">{content}</div>;
}

function ServicesForm({ config, product, templates, onChange }: { config: V10Config; product: V10Product; templates: DBTemplate[]; onChange: (config: V10Config) => void }) {
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
      {product.services.length === 0 && <p className="muted">{m.noServicesForProduct}</p>}
      {product.services.map((serviceDefinition) => {
        const name = serviceDefinition.name;
        const disabled = !serviceDefinition.hasDatabase;
        const existingService = config.gedixConfig.services[name];
        const service = existingService ?? { dbType: '', dbDsn: '', extraKeys: {} };
        const enabled = Boolean(existingService?.dbType);
        return (
          <div className="v10-service-row" key={name}>
            <div>
              <strong>{serviceDefinition.label || name}</strong>
              {disabled && <p className="muted">{m.noDatabase}</p>}
            </div>
            {!disabled && (
              <>
                <label className="checkbox-row">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(event) => updateService(name, event.currentTarget.checked ? { dbType: 'sqlite', dbDsn: '', extraKeys: {} } : null)}
                  />
                  {m.configureDb}
                </label>
                {enabled && (
                  <div className="v10-service-config">
                    <label>{m.dbType}
                      <select value={service?.dbType ?? ''} onChange={(event) => updateService(name, { ...service!, dbType: event.currentTarget.value, dbDsn: event.currentTarget.value === 'sqlite' ? '' : service?.dbDsn ?? '' })}>
                        {['sqlite', 'mysql', 'postgres', 'mssql', 'oracle'].map((type) => <option key={type} value={type}>{type}</option>)}
                      </select>
                    </label>
                    <label>{service?.dbType === 'sqlite' ? m.sqliteDsn : m.dbDsn}
                      <input placeholder={service?.dbType === 'sqlite' ? m.sqliteDsnPlaceholder : ''} value={service?.dbDsn ?? ''} onChange={(event) => updateService(name, { ...service!, dbDsn: event.currentTarget.value })} />
                    </label>
                    <label>{m.dsnTemplate}
                      <select value="" onChange={(event) => updateService(name, { ...service!, dbDsn: event.currentTarget.value })}>
                        <option value="">{m.insertTemplate}</option>
                        {templates.filter((template) => template.template).map((template) => <option key={template.type} value={template.template}>{template.type}</option>)}
                      </select>
                    </label>
                  </div>
                )}
              </>
            )}
            {serviceDefinition.supportsExtraKeys && <ExtraKeysEditor serviceKey={`${config.name}:${name}`} service={service} onChange={(next) => updateService(name, next)} />}
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
          <input value={row.key} placeholder={m.extraKeyName} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, key: event.currentTarget.value } : item))} />
          <input value={row.value} placeholder={m.extraKeyValue} onChange={(event) => commitRows(rows.map((item) => item.id === row.id ? { ...item, value: event.currentTarget.value } : item))} />
          <Button type="button" size="sm" variant="danger" onClick={() => commitRows(rows.filter((item) => item.id !== row.id))}>{m.delete}</Button>
        </div>
      ))}
    </div>
  );
}

function ConnectorsForm({ config, product, onChange, onScanCfg }: { config: V10Config; product: V10Product; onChange: (config: V10Config) => void; onScanCfg: (file: File, importExistingKeys: boolean) => void }) {
  const [rows, setRows] = useState<ConnectorFormRow[]>(() => unitRowsFromConfig(config, product));
  const [importExistingKeys, setImportExistingKeys] = useState(false);

  useEffect(() => {
    setRows(unitRowsFromConfig(config, product));
  }, [config.name, product.id]);

  useEffect(() => {
    setRows((current) => {
      const existing = new Set(current.map((row) => row.name));
      const missing = Object.entries(unitsForConfig(config, product))
        .filter(([name]) => !existing.has(name))
        .map(([name, connector]) => ({ id: makeID(), name, module: connector.module ?? '', rawConfig: connector.rawConfig }));
      return missing.length > 0 ? [...current, ...missing] : current;
    });
  }, [config.gedixConfig.connectors, config.gedixConfig.agents, config.gedixConfig.units, product.id]);

  const commitRows = (nextRows: ConnectorFormRow[]) => {
    setRows(nextRows);
    const units: Record<string, { module: string; rawConfig: string }> = {};
    for (const row of nextRows) {
      const name = row.name.trim();
      if (name) {
        units[name] = { module: normalizeModuleType(row.module), rawConfig: row.rawConfig };
      }
    }
    const unitKey = product.unitKind === 'agent' ? 'agents' : 'connectors';
    onChange({ ...config, gedixConfig: { ...config.gedixConfig, [unitKey]: units } });
  };

  const duplicate = hasDuplicateConnector(rows);

  return (
    <div className="v10-connector-list">
      <p className="readonly-notice">{unitHelp(product)}</p>
      <p className="readonly-notice">{m.units.moduleHelp}</p>
      <p className="muted">{m.scanCfgHelp}</p>
      <div className="button-row">
        <label className="ui-button secondary sm v10-file-button">
          {product.unitKind === 'agent' ? m.units.scanAgents : m.units.scanConnectors}
          <input
            type="file"
            accept=".cfg"
            onChange={(event) => {
              const file = event.currentTarget.files?.[0];
              if (file) {
                onScanCfg(file, importExistingKeys);
              }
              event.currentTarget.value = '';
            }}
          />
        </label>
        <label className="checkbox-row v10-inline-checkbox">
          <input type="checkbox" checked={importExistingKeys} onChange={(event) => setImportExistingKeys(event.currentTarget.checked)} />
          Importer clés existantes
        </label>
      </div>
      {duplicate && <p className="error">{m.duplicateConnector}</p>}
      {rows.map((row) => (
        <div className="v10-connector-row" key={row.id}>
          <div>
            <label>{product.unitKind === 'agent' ? m.units.agentName : m.units.connectorName}
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
      <Button type="button" variant="secondary" onClick={() => commitRows([...rows, { id: makeID(), name: `${product.unitFolderPrefix || 'connector-'}${rows.length + 1}`, module: '', rawConfig: '' }])}>{product.unitKind === 'agent' ? m.units.addAgent : m.units.addConnector}</Button>
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
      {actions.length === 0 && <p className="muted">{m.actionPlan.noActionsForProduct}</p>}
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
                    const selected = byID[event.currentTarget.value];
                    updateStep(index, { action: event.currentTarget.value, label: selected?.label ?? '', params: selected ? paramsFromActionDefaults(selected) : {} });
                  }}>
                    <option value="">{m.chooseAction}</option>
                    {actions.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
                  </select>
                </label>
                <label>{m.label}
                  <input value={step.label} onChange={(event) => updateStep(index, { ...step, label: event.currentTarget.value })} />
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
        <Button type="button" variant="secondary" onClick={() => {
          const action = actions[0];
          onChange({ ...config, pipeline: [...apiSteps, { action: action?.id ?? '', label: action?.label ?? '', params: action ? paramsFromActionDefaults(action) : {} }] });
        }} disabled={actions.length === 0}>{m.addAction}</Button>
      </div>
    </div>
  );
}

function ApiTokenEditor({ maquetteName, disabled }: { maquetteName: string; disabled: boolean }) {
  const [hasToken, setHasToken] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draftToken, setDraftToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    setDraftToken('');
    setEditing(false);
    v10LabApi.getApiTokenStatus(maquetteName)
      .then((status) => {
        if (!cancelled) {
          setHasToken(status.hasToken);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Erreur inconnue');
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [maquetteName]);

  const save = async () => {
    const token = draftToken.trim();
    if (!token) {
      setError(m.apiToken.required);
      return;
    }
    setLoading(true);
    setError('');
    try {
      const status = await v10LabApi.saveApiToken(maquetteName, token);
      setHasToken(status.hasToken);
      setEditing(false);
      setDraftToken('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur inconnue');
    } finally {
      setLoading(false);
    }
  };

  const remove = async () => {
    setLoading(true);
    setError('');
    try {
      await v10LabApi.deleteApiToken(maquetteName);
      setHasToken(false);
      setEditing(false);
      setDraftToken('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur inconnue');
    } finally {
      setLoading(false);
    }
  };

  const editingToken = editing || !hasToken;
  return (
    <div className="v10-api-token">
      <label>{m.apiToken.label}
        <input
          type="password"
          value={editingToken ? draftToken : '************'}
          placeholder={m.apiToken.placeholder}
          disabled={disabled || loading || !editingToken}
          className={!editingToken ? 'masked-token' : undefined}
          onChange={(event) => setDraftToken(event.currentTarget.value)}
        />
      </label>
      <div className="button-row">
        {editingToken ? (
          <>
            <Button type="button" size="sm" onClick={() => void save()} disabled={disabled || loading || !draftToken.trim()}>{m.apiToken.save}</Button>
            {hasToken && <Button type="button" size="sm" variant="secondary" onClick={() => { setEditing(false); setDraftToken(''); setError(''); }} disabled={disabled || loading}>{m.apiToken.cancel}</Button>}
          </>
        ) : (
          <>
            <Button type="button" size="sm" variant="secondary" onClick={() => { setEditing(true); setDraftToken(''); setError(''); }} disabled={disabled || loading}>{m.apiToken.edit}</Button>
            <Button type="button" size="sm" variant="danger" onClick={() => void remove()} disabled={disabled || loading}>{m.apiToken.delete}</Button>
          </>
        )}
      </div>
      {hasToken && !editingToken && <p className="muted">{m.apiToken.saved}</p>}
      {error && <p className="error">{error}</p>}
    </div>
  );
}

function ActionFieldInput({ field, value, onChange }: { field: V10Action['fields'][number]; value: unknown; onChange: (value: unknown) => void }) {
  const label = <FieldLabel field={field} />;
  if (field.type === 'bool') {
    return <label className="checkbox-row"><input type="checkbox" checked={Boolean(value)} onChange={(event) => onChange(event.currentTarget.checked)} />{label}</label>;
  }
  if (field.type === 'string[]') {
    return <label>{label}<input value={Array.isArray(value) ? value.join(',') : ''} onChange={(event) => onChange(event.currentTarget.value.split(',').map((item) => item.trim()).filter(Boolean))} /></label>;
  }
  if (field.type === 'text') {
    return <label>{label}<textarea value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} />{field.description && <span className="muted">{field.description}</span>}</label>;
  }
  if (field.type === 'number') {
    return <label>{label}<input type="number" value={typeof value === 'number' ? value : ''} onChange={(event) => onChange(Number(event.currentTarget.value))} /></label>;
  }
  return <label>{label}<input value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} /></label>;
}

function FieldLabel({ field }: { field: V10Action['fields'][number] }) {
  return (
    <span className="v10-field-label">
      {field.label}
      {field.required && <span className="v10-required-dot" title="Champ obligatoire" aria-label="Champ obligatoire" role="img" />}
    </span>
  );
}

function ModuleCommandPanel({ config, product, disabled, onRun, showTitle = true }: { config: V10Config; product: V10Product; disabled: boolean; onRun: (unitName: string, command: string) => void; showTitle?: boolean }) {
  const unitNames = Object.keys(unitsForConfig(config, product)).sort((left, right) => left.localeCompare(right));
  const [unitName, setUnitName] = useState(unitNames[0] ?? '');
  const [command, setCommand] = useState('');
  const invalid = moduleCommandHasUnsafeCharacters(command);
  const isAgent = product.unitKind === 'agent';

  useEffect(() => {
    if (!unitName || !unitNames.includes(unitName)) {
      setUnitName(unitNames[0] ?? '');
    }
  }, [unitNames.join('|'), unitName]);

  return (
    <div className="v10-module-command">
      {showTitle && <h4>{isAgent ? m.moduleCommand.titleAgent : m.moduleCommand.titleConnector}</h4>}
      <p className="muted">{m.moduleCommand.help}</p>
      {unitNames.length === 0 ? (
        <p className="muted">{isAgent ? m.moduleCommand.noAgent : m.moduleCommand.noConnector}</p>
      ) : (
        <div className="form-grid v10-form-grid">
          <label>{isAgent ? m.moduleCommand.agent : m.moduleCommand.connector}
            <select value={unitName} onChange={(event) => setUnitName(event.currentTarget.value)}>
              {unitNames.map((name) => <option key={name} value={name}>{name}</option>)}
            </select>
          </label>
          <label>{m.moduleCommand.command}
            <input value={command} placeholder={m.moduleCommand.commandPlaceholder} onChange={(event) => setCommand(event.currentTarget.value)} />
          </label>
        </div>
      )}
      {invalid && <p className="error">{m.moduleCommand.invalidCommand}</p>}
      <div className="button-row">
        <Button type="button" variant="secondary" disabled={disabled || !unitName || !command.trim() || invalid} onClick={() => onRun(unitName, command)}>
          {m.moduleCommand.run}
        </Button>
      </div>
    </div>
  );
}

function ExecutionPanel({ config, product, busy, runState, execution, logs, selectedLog, onConfigChange, onCreate, onUpdate, onConfigure, onStart, onOpenMaquette, onRunPipeline, onRunModuleCommand, onKill, onRefreshLogs, onReadLog }: {
  config: V10Config;
  product: V10Product;
  busy: boolean;
  runState: RunState;
  execution: ExecutionResponse | null;
  logs: LogSummary[];
  selectedLog: string;
  onConfigChange: (config: V10Config) => void;
  onCreate: () => void;
  onUpdate: () => void;
  onConfigure: () => void;
  onStart: () => void;
  onOpenMaquette: () => void;
  onRunPipeline: () => void;
  onRunModuleCommand: (unitName: string, command: string) => void;
  onKill: () => void;
  onRefreshLogs: () => void;
  onReadLog: (logFile: string) => void;
}) {
  const currentLog = execution?.log || execution?.output || execution?.status || '';
  const disabled = busy || runState === 'running';
  return (
    <div className="v10-execution">
      <details className="v10-execution-section v10-collapsible-section">
        <summary>{m.execution.debugTitle.replace('connecteurs', product.unitPluralLabel)}</summary>
        <DebugTargetsEditor config={config} product={product} onChange={onConfigChange} />
      </details>
      <details className="v10-execution-section v10-collapsible-section">
        <summary>{product.unitKind === 'agent' ? m.moduleCommand.titleAgent : m.moduleCommand.titleConnector}</summary>
        <ModuleCommandPanel config={config} product={product} disabled={disabled} onRun={onRunModuleCommand} showTitle={false} />
      </details>
      <section className="v10-execution-section">
        <h4>{m.execution.actionsTitle}</h4>
        <div className="button-row">
          <Button type="button" onClick={onCreate} disabled={disabled}>{m.execution.createMaquette}</Button>
          <Button type="button" variant="secondary" onClick={onUpdate} disabled={disabled}>{m.execution.updateMaquette}</Button>
          <Button type="button" variant="secondary" onClick={onConfigure} disabled={disabled}>{m.execution.configureCfg}</Button>
          <Button type="button" variant="success" onClick={onStart} disabled={disabled}>{m.execution.startMaquette}</Button>
          <Button type="button" variant="secondary" onClick={onOpenMaquette} disabled={disabled}>Ouvrir maquette</Button>
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

function defaultConfig(product = DEFAULT_V10_PRODUCT_ID, productDefinition?: V10Product): V10Config {
  return normalizeConfig({
    name: '',
    product,
    release: { zipPath: '', workDir: '', overwrite: false },
    maquette: { targetPath: '', envName: 'live', appName: productDefinition?.defaultAppName ?? 'prod' },
    gedixConfig: { fqdn: '', port: 80, services: {}, connectors: {}, agents: {} },
    runtime: { debugTargets: [], openConsole: true },
    groupName: '',
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
      agents: gedixConfig.agents ?? {},
      units: gedixConfig.units ?? {},
    }),
    runtime: {
      ...runtime,
      debugTargets: runtime.debugTargets ?? [],
      debugTargetFlags: runtime.debugTargetFlags ?? {},
      openConsole: runtime.openConsole ?? true,
    },
    groupName: config.groupName ?? '',
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
  if (hasDuplicateConnector(Object.keys(unitsForConfig(config, productFor(config.product, []))).map((name) => ({ id: name, name, module: '', rawConfig: '' })))) {
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

function unitRowsFromConfig(config: V10Config, product: V10Product): ConnectorFormRow[] {
  return Object.entries(unitsForConfig(config, product)).map(([name, connector]) => ({
    id: makeID(),
    name,
    module: connector.module ?? '',
    rawConfig: connector.rawConfig,
  }));
}

function unitsForConfig(config: V10Config, product: V10Product): Record<string, ConnectorConfig> {
  const genericUnits = config.gedixConfig.units ?? {};
  const typedUnits = product.unitKind === 'agent' ? (config.gedixConfig.agents ?? {}) : (config.gedixConfig.connectors ?? {});
  return { ...genericUnits, ...typedUnits };
}

function productFor(productId: string | undefined, products: V10Product[]): V10Product {
  return products.find((product) => product.id === productId) ?? products.find((product) => product.id === DEFAULT_V10_PRODUCT_ID) ?? {
    id: DEFAULT_V10_PRODUCT_ID,
    name: 'Gedix Prod V10',
    label: 'Gedix Prod V10',
    description: '',
    defaultAppName: 'prod',
    services: [
      { name: 'webserver', label: 'webserver', hasDatabase: false, supportsExtraKeys: true },
      { name: 'auth', label: 'auth', hasDatabase: true, supportsExtraKeys: true },
      { name: 'filestore', label: 'filestore', hasDatabase: true, supportsExtraKeys: true },
      { name: 'entreprise', label: 'entreprise', hasDatabase: true, supportsExtraKeys: true },
      { name: 'etl', label: 'etl', hasDatabase: true, supportsExtraKeys: true },
      { name: 'dnc', label: 'dnc', hasDatabase: true, supportsExtraKeys: true },
      { name: 'reactor', label: 'reactor', hasDatabase: false, supportsExtraKeys: true },
      { name: 'config', label: 'config', hasDatabase: true, supportsExtraKeys: true },
    ],
    unitKind: 'connector',
    unitSingularLabel: 'connecteur',
    unitPluralLabel: 'connecteurs',
    unitCfgSectionName: 'connectors',
    unitFolderPrefix: 'connector-',
    unitExecutableName: 'gx-connector.exe',
  };
}

function unitHelp(product: V10Product) {
  return `Le nom du ${product.unitSingularLabel} doit correspondre exactement à la section du gedix.cfg : [environments.<env>.applications.<app>.${product.unitCfgSectionName}.<nom${product.unitSingularLabel}>]`;
}

function moduleCommandHasUnsafeCharacters(command: string) {
  return /[&|><]/.test(command);
}

function normalizeDebugFlag(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }
  return trimmed.startsWith('--') ? trimmed : `--${trimmed}`;
}

function debugFlagHasUnsafeCharacters(value: string) {
  const normalized = normalizeDebugFlag(value);
  return !normalized || normalized === '--' || normalized === '---' || /[\s"'&|;<>]/.test(normalized) || !/^--[A-Za-z0-9._=-]+$/.test(normalized);
}

function normalizeModuleType(value: string) {
  return value.trim().replace(/^["']|["']$/g, '').trim().replace(/^module-/i, '');
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

function paramsFromActionDefaults(action: V10Action): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  for (const field of action.fields) {
    if (field.default !== undefined && field.default !== null) {
      params[field.name] = field.default;
    }
  }
  return params;
}

function debugFlagButtons(debugTargetFlags: Record<string, string[]>, removeCustomFlag: (target: string, flag: string) => void) {
  const buttons: React.ReactNode[] = [];
  for (const [target, flags] of Object.entries(debugTargetFlags)) {
    for (const flag of flags) {
      buttons.push(
        <Button key={`${target}:${flag}`} type="button" size="sm" variant="secondary" onClick={() => removeCustomFlag(target, flag)}>
          {target} {flag} - {m.removeDebugTarget}
        </Button>,
      );
    }
  }
  return buttons;
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

