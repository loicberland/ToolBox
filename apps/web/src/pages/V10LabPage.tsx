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
  V10SavedActionPlan,
  V10UnitDefinition,
  UnitKind,
  ExecutableCommandTargetKind,
  v10LabApi,
} from '../api/v10Lab';
import { Button } from '../components/ui/Button';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { Toast } from '../components/ui/Toast';
import { messages } from '../i18n';
import { RequiredDot } from './RequiredDot';
import { isServiceDsnRequired, validateServiceDsns } from './v10LabValidation';

const m = messages.v10Lab;
const tabs = [m.tabs.general, m.tabs.gedix, m.tabs.services, m.tabs.adaptors, m.tabs.connectors, m.tabs.pipeline, m.tabs.execution, m.tabs.json] as const;
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
type ImportableActionPlan = {
  schema?: string;
  version?: number;
  name?: string;
  productId?: string;
  actions?: PipelineStep[];
  pipeline?: PipelineStep[];
};

export function V10LabPage({ onBeforeLeaveChange }: { onBeforeLeaveChange?: (handler: BeforeLeaveHandler | null) => void }) {
  const [products, setProducts] = useState<V10Product[]>([]);
  const [actions, setActions] = useState<V10Action[]>([]);
  const [templates, setTemplates] = useState<DBTemplate[]>([]);
  const [maquettes, setMaquettes] = useState<MaquetteSummary[]>([]);
  const [groups, setGroups] = useState<MaquetteGroup[]>([]);
  const [savedActionPlans, setSavedActionPlans] = useState<V10SavedActionPlan[]>([]);
  const [selectedSavedActionPlanID, setSelectedSavedActionPlanID] = useState('');
  const [showSaveActionPlan, setShowSaveActionPlan] = useState(false);
  const [actionPlanName, setActionPlanName] = useState('');
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});
  const [openUngrouped, setOpenUngrouped] = useState(false);
  const [newGroupName, setNewGroupName] = useState('');
  const [selectedName, setSelectedName] = useState('');
  const [config, setConfig] = useState<V10Config | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>(m.tabs.general);
  const [showCreate, setShowCreate] = useState(false);
  const [showMaquetteSelector, setShowMaquetteSelector] = useState(true);
  const [draft, setDraft] = useState(() => defaultConfig());
  const [jsonText, setJsonText] = useState('');
  const [logs, setLogs] = useState<LogSummary[]>([]);
  const [selectedLog, setSelectedLog] = useState('');
  const [defaultTargetPath, setDefaultTargetPath] = useState('');
  const [busy, setBusy] = useState(false);
  const [runState, setRunState] = useState<RunState>('idle');
  const [isDirty, setIsDirty] = useState(false);
  const [saveAttempted, setSaveAttempted] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [toastInfo, setToastInfo] = useState('');
  const [toastError, setToastError] = useState('');
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [confirmKill, setConfirmKill] = useState(false);
  const [confirmUpdate, setConfirmUpdate] = useState(false);
  const [execution, setExecution] = useState<ExecutionResponse | null>(null);
  const currentMaquetteRef = useRef<HTMLElement | null>(null);
  const maquetteSelectorRef = useRef<HTMLElement | null>(null);
  const actionPlanImportInputRef = useRef<HTMLInputElement | null>(null);
  const scrollToCurrentMaquetteAfterTabChange = useRef(false);
  const scrollToMaquetteSelectorAfterOpen = useRef(false);

  useEffect(() => {
    void loadInitial();
  }, []);

  useEffect(() => {
    if (!toastError) {
      return;
    }
    const timeout = window.setTimeout(() => setToastError(''), 7000);
    return () => window.clearTimeout(timeout);
  }, [toastError]);

  useEffect(() => {
    if (!message) {
      return;
    }
    setToastInfo(message);
  }, [message]);

  useEffect(() => {
    if (!toastInfo) {
      return;
    }
    const timeout = window.setTimeout(() => setToastInfo(''), 7000);
    return () => window.clearTimeout(timeout);
  }, [toastInfo]);

  useEffect(() => {
    if (config) {
      setJsonText(JSON.stringify(config, null, 2));
    }
  }, [config]);

  useEffect(() => {
    if (config) {
      void loadActions(config.product);
      void loadSavedActionPlans(config.product);
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
    if (!showMaquetteSelector || !scrollToMaquetteSelectorAfterOpen.current || !config) {
      return;
    }
    scrollToMaquetteSelectorAfterOpen.current = false;
    window.requestAnimationFrame(() => {
      maquetteSelectorRef.current?.scrollIntoView({ block: 'start', inline: 'nearest', behavior: 'smooth' });
    });
  }, [showMaquetteSelector, config]);

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
      return saved;
    });
    return () => onBeforeLeaveChange(null);
  }, [onBeforeLeaveChange, config, isDirty]);

  const selectedSummary = maquettes.find((item) => item.name === selectedName);
  const currentProduct = productFor(config?.product ?? draft.product, products);
  const visibleTabs = tabs.filter((tab) => {
    if (tab === m.tabs.services) {
      return currentProduct.services.length > 0;
    }
    if (tab === m.tabs.connectors) {
      return productHasUnitKind(currentProduct, 'connector') || productHasUnitKind(currentProduct, 'agent');
    }
    if (tab === m.tabs.adaptors) {
      return productHasUnitKind(currentProduct, 'adaptor');
    }
    return true;
  });

  useEffect(() => {
    if (!visibleTabs.includes(activeTab)) {
      setActiveTab(m.tabs.general);
    }
  }, [activeTab, currentProduct.id, currentProduct.services.length, currentProduct.unitKind, currentProduct.unitCfgSectionName, currentProduct.unitDefinitions]);

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
      await loadSavedActionPlans(product?.id ?? DEFAULT_V10_PRODUCT_ID);
    });
  }

  async function loadActions(product: string) {
    const items = await v10LabApi.actions(product || DEFAULT_V10_PRODUCT_ID);
    setActions(items);
  }

  async function loadSavedActionPlans(product: string) {
    try {
      const items = await v10LabApi.listSavedActionPlans(product || DEFAULT_V10_PRODUCT_ID);
      setSavedActionPlans(items);
      setSelectedSavedActionPlanID((current) => current && items.some((item) => item.id === current) ? current : (items[0]?.id ?? ''));
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Erreur chargement plans d\'actions');
    }
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

  function scrollToCurrentMaquette() {
    window.requestAnimationFrame(() => {
      currentMaquetteRef.current?.scrollIntoView({ block: 'start', inline: 'nearest', behavior: 'auto' });
    });
  }

  async function openMaquette(name: string) {
    await run(async () => {
      const loaded = normalizeConfig(await v10LabApi.getMaquette(name));
      setSelectedName(loaded.name);
      setConfig(loaded);
      setShowMaquetteSelector(false);
      setIsDirty(false);
      setSaveAttempted(false);
      setActiveTab(m.tabs.general);
      setExecution(null);
      setSelectedLog('');
      setLogs([]);
      scrollToCurrentMaquette();
    });
  }

  async function createMaquette(groupName = draft.groupName ?? '') {
    setSaveAttempted(true);
    const validation = validateConfig(draft, products);
    if (validation) {
      if (validateServiceDsns(draft, productFor(draft.product, products))) {
        setActiveTab(m.tabs.services);
      }
      showError(validation);
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
    setSaveAttempted(true);
    const validation = validateConfig(config, products);
    if (validation) {
      setMessage('');
      if (validateServiceDsns(config, productFor(config.product, products))) {
        setActiveTab(m.tabs.services);
      }
      showError(validation);
      return false;
    }
    const requiredFieldsValidation = validatePipelineRequiredFields(config, actions);
    if (requiredFieldsValidation) {
      setMessage('');
      showError(requiredFieldsValidation);
      return false;
    }
    let saved = false;
    await run(async () => {
      const oldName = config.name;
      const next = normalizeConfig(await v10LabApi.updateMaquette(selectedName || oldName, normalizeConfig(config)));
      setConfig(next);
      setSelectedName(next.name);
      setIsDirty(false);
      setSaveAttempted(false);
      await reloadList();
      setMessage(m.saved);
      saved = true;
    });
    return saved;
  }

  async function saveCurrentActionPlan(overwrite = false) {
    if (!config) {
      return;
    }
    const name = actionPlanName.trim();
    if (!name) {
      showError('Nom du plan d\'actions obligatoire.');
      return;
    }
    const apiSteps = (config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action));
    await run(async () => {
      let saved: V10SavedActionPlan;
      try {
        saved = await v10LabApi.saveActionPlan({
          name,
          productId: config.product,
          actions: deepClonePipeline(apiSteps),
          overwrite,
        });
      } catch (err) {
        if (!overwrite && err instanceof Error && err.message.toLowerCase().includes('existe') && window.confirm('Un plan d\'actions avec ce nom existe déjà. Voulez-vous le remplacer ?')) {
          saved = await v10LabApi.saveActionPlan({
            name,
            productId: config.product,
            actions: deepClonePipeline(apiSteps),
            overwrite: true,
          });
        } else {
          throw err;
        }
      }
      await loadSavedActionPlans(config.product);
      setSelectedSavedActionPlanID(saved.id);
      setShowSaveActionPlan(false);
      setActionPlanName('');
      setMessage('Plan d\'actions enregistré.');
    });
  }

  function addSavedActionPlanToCurrent() {
    if (!config) {
      return;
    }
    const plan = savedActionPlans.find((item) => item.id === selectedSavedActionPlanID);
    if (!plan) {
      showError('Sélectionnez un plan d\'actions enregistré.');
      return;
    }
    const apiSteps = (config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action));
    updateConfig({ ...config, pipeline: [...apiSteps, ...deepClonePipeline(plan.actions)] });
    setMessage('Plan d\'actions ajouté au plan actuel.');
  }

  function exportCurrentActionPlan() {
    if (!config) {
      return;
    }
    const apiSteps = (config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action));
    const payload: ImportableActionPlan = {
      schema: 'toolbox-v10-lab-action-plan',
      version: 1,
      name: actionPlanName.trim() || defaultActionPlanName(config.name),
      productId: config.product,
      actions: deepClonePipeline(apiSteps),
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `plan-actions-${safeFileName(actionPlanName.trim() || config.name || config.product)}.json`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
    setMessage(m.pipeline.exported);
  }

  function openActionPlanImport() {
    actionPlanImportInputRef.current?.click();
  }

  async function importActionPlanFile(file: File | null) {
    if (!config || !file) {
      return;
    }
    try {
      const parsed = JSON.parse(await file.text()) as ImportableActionPlan;
      const importedSteps = extractImportedActionPlanSteps(parsed);
      if (parsed.productId && parsed.productId !== config.product) {
        showError(formatMessage(m.pipeline.productMismatch, { product: parsed.productId, currentProduct: config.product }));
        return;
      }
      if (importedSteps.length === 0) {
        showError(m.pipeline.noImportedActions);
        return;
      }
      const availableActionIDs = new Set(actions.map((action) => action.id));
      const invalidStep = importedSteps.find((step) => !availableActionIDs.has(step.action));
      if (invalidStep) {
        showError(formatMessage(m.pipeline.incompatibleAction, { action: invalidStep.action || m.chooseAction }));
        return;
      }
      const legacySteps = (config.pipeline ?? []).filter((step) => systemPipelineActions.has(step.action));
      updateConfig({ ...config, pipeline: [...legacySteps, ...deepClonePipeline(importedSteps)] });
      setActionPlanName(parsed.name ?? '');
      setMessage(m.pipeline.importedUnsaved);
    } catch (err) {
      showError(err instanceof SyntaxError ? m.pipeline.invalidImportFile : (err instanceof Error ? err.message : m.pipeline.invalidImportFile));
    } finally {
      if (actionPlanImportInputRef.current) {
        actionPlanImportInputRef.current.value = '';
      }
    }
  }

  async function deleteSavedActionPlan() {
    if (!config) {
      return;
    }
    const plan = savedActionPlans.find((item) => item.id === selectedSavedActionPlanID);
    if (!plan) {
      showError('Sélectionnez un plan d\'actions enregistré.');
      return;
    }
    if (!window.confirm(`Supprimer le plan d'actions enregistré "${plan.name}" ?`)) {
      return;
    }
    await run(async () => {
      await v10LabApi.deleteSavedActionPlan(plan.id);
      await loadSavedActionPlans(config.product);
      setMessage('Plan d\'actions enregistré supprimé.');
    });
  }

  async function changeTab(nextTab: Tab) {
    if (nextTab === activeTab) {
      return;
    }
    if (config && isDirty) {
      setMessage(m.savingBeforeTabChange);
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    scrollToCurrentMaquetteAfterTabChange.current = false;
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

  async function runExecutableCommand(targetKind: ExecutableCommandTargetKind, targetName: string, command: string, name = selectedName) {
    if (!name || runState === 'running') {
      return;
    }
    if (executableCommandHasUnclosedQuote(command)) {
      showError(m.moduleCommand.unclosedQuote);
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
      const started = await v10LabApi.runExecutableCommand(name, targetKind, targetName, command);
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
      setSaveAttempted(true);
      const validation = validateConfig(parsed, products) || validatePipelineRequiredFields(parsed, actions);
      if (validation) {
        setMessage('');
        if (validateServiceDsns(parsed, productFor(parsed.product, products))) {
          setActiveTab(m.tabs.services);
        }
        showError(validation);
        return;
      }
      const saved = normalizeConfig(await v10LabApi.updateMaquette(selectedName || config.name, parsed));
      setConfig(saved);
      setSelectedName(saved.name);
      setIsDirty(false);
      setSaveAttempted(false);
      await reloadList();
      setMessage(m.jsonSaved);
      setError('');
    } catch (err) {
      showError(err instanceof Error ? err.message : 'JSON invalide');
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

  async function openCurrentMaquetteFolder() {
    if (!config) {
      return;
    }
    if (!config.maquette.targetPath.trim()) {
      showError(m.openMaquetteFolderMissingPath);
      return;
    }
    if (isDirty) {
      const saved = await saveCurrent();
      if (!saved) {
        return;
      }
    }
    await run(async () => {
      await v10LabApi.openMaquetteFolder(config.name);
      setMessage(m.openMaquetteFolderSuccess);
    }, () => showError(m.openMaquetteFolderError));
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
    setShowMaquetteSelector(true);
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
      showError('Nom de groupe obligatoire.');
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

  async function scanCfg(unitKind: UnitKind, file: File, importExistingKeys: boolean, replaceExistingUnits: boolean) {
    if (!config) {
      return;
    }
    if (!stringsEqual(filepathExt(file.name), '.cfg')) {
      showError(m.errors.cfgOnly);
      return;
    }
    await run(async () => {
      const product = productFor(config.product, products);
      const result = await v10LabApi.scanCfg(config.name, file, config.maquette.envName, config.maquette.appName || product.defaultAppName || 'prod', importExistingKeys);
      const scannedUnits = scanUnitsForKind(result, unitKind);
      const unitKey = unitConfigKey(unitKind);
      const definition = unitDefinitionForKind(product, unitKind);
      const units = { ...(config.gedixConfig[unitKey] ?? {}) };
      for (const unit of scannedUnits) {
        const existing = units[unit.name];
        if (existing && !replaceExistingUnits) {
          units[unit.name] = {
            module: existing.module ?? unit.module ?? '',
            rawConfig: existing.rawConfig ?? unit.rawConfig ?? '',
          };
          continue;
        }
        units[unit.name] = {
          module: unit.module ?? '',
          rawConfig: unit.rawConfig ?? '',
        };
      }
      updateConfig({
        ...config,
        maquette: { ...config.maquette, envName: result.envName || config.maquette.envName, appName: result.appName || config.maquette.appName },
        gedixConfig: { ...config.gedixConfig, [unitKey]: units },
      });
      const warnings = result.warnings?.length ? ` ${result.warnings.join(' ')}` : '';
      const replaced = replaceExistingUnits ? ' Les éléments existants ont été remplacés.' : '';
      setMessage(`${scannedUnits.length} ${definition.pluralLabel} detecte(s).${replaced}${warnings}`);
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
      showError(err instanceof Error ? err.message : 'Erreur inconnue');
    } finally {
      setBusy(false);
    }
  }

  function showError(nextError: string) {
    setError(nextError);
    setToastError(nextError);
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

  function toggleMaquetteSelector() {
    setShowMaquetteSelector((current) => {
      const next = !current;
      scrollToMaquetteSelectorAfterOpen.current = next;
      return next;
    });
  }

  const groupedMaquettes = groups.map((group) => ({
    ...group,
    items: maquettes.filter((item) => item.groupName === group.name),
  }));
  const ungroupedMaquettes = maquettes.filter((item) => !item.groupName || !groups.some((group) => group.name === item.groupName));

  function renderMaquetteListSection() {
    return (
      <section ref={maquetteSelectorRef} className="ui-card v10-section">
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
            <div className="v10-group">
              <div
                className="v10-group-header clickable"
                role="button"
                tabIndex={0}
                onClick={() => setOpenUngrouped((value) => !value)}
                onKeyDown={(event) => handleToggleKey(event, () => setOpenUngrouped((value) => !value))}
              >
                <span className="v10-chevron" aria-hidden="true">{openUngrouped ? '▾' : '▸'}</span>
                <strong>Sans groupe</strong>
                <span className="muted">{ungroupedMaquettes.length}</span>
                <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); setDraft({ ...defaultConfig(currentProduct.id, currentProduct), groupName: '' }); setShowCreate(true); setOpenUngrouped(true); }}>Ajouter une maquette</Button>
              </div>
              {openUngrouped && <MaquetteList items={ungroupedMaquettes} selectedName={selectedName} onToggle={toggleMaquette} />}
            </div>
          </div>
        )}
      </section>
    );
  }

  return (
    <div className="workspace v10-lab-workspace">
      <div className="toast-stack" aria-live="polite">
        <Toast message={toastError} type="error" onClose={() => setToastError('')} />
        <Toast message={toastInfo} type="info" onClose={() => setToastInfo('')} />
      </div>
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
      
      {error && <p className="error whitespace">{error}</p>}
      {message && <p className="info-message">{message}</p>}
      
      {config && (
        <section ref={currentMaquetteRef} className="ui-card v10-section">
          <div className="ui-card-header v10-current-maquette-header">
            <div className="button-row end">
              <Button type="button" variant="secondary" onClick={() => void closeMaquette()} disabled={busy}>{m.close}</Button>
              <Button type="button" variant="secondary" onClick={toggleMaquetteSelector}>
                {showMaquetteSelector ? m.maquetteSelector.hide : m.maquetteSelector.show}
              </Button>
              <Button type="button" variant="success" onClick={() => void openCurrentMaquetteFolder()} disabled={busy || !config.maquette.targetPath.trim()}>{m.openMaquetteFolder}</Button>
              <Button type="button" onClick={() => void saveCurrent()} disabled={busy}>{m.save}</Button>
              <Button type="button" variant="danger" onClick={() => setConfirmDelete(config.name)} disabled={busy}>{m.delete}</Button>
            </div>
            <div>
              <h3>{config.name}</h3>
              <h4>{currentProduct.label || currentProduct.name}</h4>
              <p className="muted">{selectedSummary?.targetPath ?? config.maquette.targetPath}</p>
            </div>
          </div>

          <div className="v10-tabs">
            {visibleTabs.map((tab) => (
              <button type="button" key={tab} className={tab === activeTab ? 'active' : ''} onClick={() => void changeTab(tab)}>
                {tab === m.tabs.connectors ? connectorTabLabel(currentProduct) : tab}
              </button>
            ))}
          </div>

          {activeTab === m.tabs.general && <MaquetteGeneralForm config={config} products={products} groups={groups} defaultTargetPath={defaultTargetPath} onChange={updateConfig} onSelectZip={selectReleaseZip} />}
          {activeTab === m.tabs.gedix && <GedixForm config={config} onChange={updateConfig} />}
          {activeTab === m.tabs.services && <ServicesForm config={config} product={currentProduct} templates={templates} saveAttempted={saveAttempted} onChange={updateConfig} />}
          {activeTab === m.tabs.adaptors && productHasUnitKind(currentProduct, 'adaptor') && (
            <UnitsForm config={config} product={currentProduct} unitKind="adaptor" onChange={updateConfig} onScanCfg={(kind, file, importExistingKeys, replaceExistingUnits) => void scanCfg(kind, file, importExistingKeys, replaceExistingUnits)} />
          )}
          {activeTab === m.tabs.connectors && (productHasUnitKind(currentProduct, 'connector') || productHasUnitKind(currentProduct, 'agent')) && (
            <UnitsForm config={config} product={currentProduct} unitKind={currentProduct.unitKind === 'agent' ? 'agent' : 'connector'} onChange={updateConfig} onScanCfg={(kind, file, importExistingKeys, replaceExistingUnits) => void scanCfg(kind, file, importExistingKeys, replaceExistingUnits)} />
          )}
          {activeTab === m.tabs.pipeline && (
            <LocalErrorBoundary>
              <ApiTokenEditor maquetteName={config.name} disabled={busy} />
              <PipelineBuilder
                config={config}
                actions={actions}
                savedActionPlans={savedActionPlans}
                selectedSavedActionPlanID={selectedSavedActionPlanID}
                showSaveActionPlan={showSaveActionPlan}
                actionPlanName={actionPlanName}
                onSelectedSavedActionPlanChange={setSelectedSavedActionPlanID}
                onShowSaveActionPlanChange={setShowSaveActionPlan}
                onActionPlanNameChange={setActionPlanName}
                onSaveActionPlan={() => void saveCurrentActionPlan()}
                onAddSavedActionPlan={addSavedActionPlanToCurrent}
                onExportActionPlan={exportCurrentActionPlan}
                onOpenImportActionPlan={openActionPlanImport}
                onImportActionPlan={(file) => void importActionPlanFile(file)}
                onDeleteSavedActionPlan={() => void deleteSavedActionPlan()}
                importInputRef={actionPlanImportInputRef}
                onChange={updateConfig}
              />
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
              onRunExecutableCommand={(targetKind, targetName, command) => void runExecutableCommand(targetKind, targetName, command)}
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

      {(!config || showMaquetteSelector) && renderMaquetteListSection()}

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
      gedixConfig: materializeProductServices(config.gedixConfig, product),
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
  const [selectedCustomTarget, setSelectedCustomTarget] = useState('');
  const [newCustomArguments, setNewCustomArguments] = useState('');
  const groups = executableCommandGroups(config, product, false);
  const options = groups.flatMap((group) => group.options);
  const customArguments = config.runtime.debugTargetFlags ?? {};
  const customEntries = Object.entries(customArguments).sort(([left], [right]) => left.localeCompare(right));
  const customTargets = new Set(customEntries.map(([target]) => target));
  const addableCustomOptions = options.filter((option) => !customTargets.has(option.name));

  useEffect(() => {
    if (selectedCustomTarget && !addableCustomOptions.some((option) => option.name === selectedCustomTarget)) {
      setSelectedCustomTarget('');
    }
  }, [addableCustomOptions.map((option) => option.name).join('|'), selectedCustomTarget]);

  const updateCustomArguments = (target: string, value: string) => {
    onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: { ...customArguments, [target]: [value.trim()] } } });
  };
  const changeCustomTarget = (currentTarget: string, nextTarget: string) => {
    if (!nextTarget || (nextTarget !== currentTarget && customTargets.has(nextTarget))) {
      return;
    }
    const next = { ...customArguments };
    const currentArguments = next[currentTarget];
    delete next[currentTarget];
    next[nextTarget] = currentArguments;
    onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: next } });
  };
  const removeCustomArguments = (target: string) => {
    const next = { ...customArguments };
    delete next[target];
    onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: next } });
  };
  return (
    <div className="v10-debug-targets">
      <h4>{m.execution.debugModeTitle}</h4>
      <p className="muted">{m.execution.debugModeHelp}</p>
      <div className="v10-file-row">
        <select value={selected} onChange={(event) => setSelected(event.currentTarget.value)}>
          <option value="">{m.chooseDebugTarget}</option>
          <ExecutableCommandOptions groups={groups} excludedNames={config.runtime.debugTargets} />
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
      <h4>{m.execution.customArgumentsTitle}</h4>
      <p className="muted">{m.execution.customArgumentsHelp}</p>
      {customEntries.map(([target, targetArguments]) => {
        const value = customArgumentsForTarget(targetArguments);
        const excludedNames = customEntries.map(([item]) => item).filter((item) => item !== target);
        return (
          <React.Fragment key={target}>
            <div className="v10-startup-argument-row">
              <select value={target} onChange={(event) => changeCustomTarget(target, event.currentTarget.value)}>
                {!options.some((option) => option.name === target) && <option value={target}>{target}</option>}
                <ExecutableCommandOptions groups={groups} excludedNames={excludedNames} />
              </select>
              <input value={value} onChange={(event) => updateCustomArguments(target, event.currentTarget.value)} />
              <Button type="button" variant="secondary" size="sm" onClick={() => removeCustomArguments(target)}>{m.removeDebugTarget}</Button>
            </div>
            {!value && <p className="error">{m.execution.argumentsRequired}</p>}
          </React.Fragment>
        );
      })}
      <div className="v10-startup-argument-row">
        <select value={selectedCustomTarget} onChange={(event) => setSelectedCustomTarget(event.currentTarget.value)}>
          <option value="">{m.execution.executable}</option>
          <ExecutableCommandOptions groups={groups} excludedNames={customEntries.map(([target]) => target)} />
        </select>
        <input value={newCustomArguments} onChange={(event) => setNewCustomArguments(event.currentTarget.value)} />
        <Button type="button" variant="secondary" size="sm" disabled={!selectedCustomTarget || !newCustomArguments.trim() || !addableCustomOptions.length} onClick={() => {
          onChange({ ...config, runtime: { ...config.runtime, debugTargetFlags: { ...customArguments, [selectedCustomTarget]: [newCustomArguments.trim()] } } });
          setSelectedCustomTarget('');
          setNewCustomArguments('');
        }}>{m.execution.addCustomArguments}</Button>
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

function SearchInput({ value, placeholder, onChange }: { value: string; placeholder: string; onChange: (value: string) => void }) {
  return (
    <label className="v10-search-field">
      <input value={value} placeholder={placeholder} onChange={(event) => onChange(event.currentTarget.value)} />
    </label>
  );
}

function ServicesForm({ config, product, templates, saveAttempted, onChange }: { config: V10Config; product: V10Product; templates: DBTemplate[]; saveAttempted: boolean; onChange: (config: V10Config) => void }) {
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

function UnitsForm({ config, product, unitKind, onChange, onScanCfg }: { config: V10Config; product: V10Product; unitKind: UnitKind; onChange: (config: V10Config) => void; onScanCfg: (unitKind: UnitKind, file: File, importExistingKeys: boolean, replaceExistingUnits: boolean) => void }) {
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

function PipelineBuilder({ config, actions, savedActionPlans, selectedSavedActionPlanID, showSaveActionPlan, actionPlanName, onSelectedSavedActionPlanChange, onShowSaveActionPlanChange, onActionPlanNameChange, onSaveActionPlan, onAddSavedActionPlan, onExportActionPlan, onOpenImportActionPlan, onImportActionPlan, onDeleteSavedActionPlan, importInputRef, onChange }: {
  config: V10Config;
  actions: V10Action[];
  savedActionPlans: V10SavedActionPlan[];
  selectedSavedActionPlanID: string;
  showSaveActionPlan: boolean;
  actionPlanName: string;
  onSelectedSavedActionPlanChange: (id: string) => void;
  onShowSaveActionPlanChange: (show: boolean) => void;
  onActionPlanNameChange: (name: string) => void;
  onSaveActionPlan: () => void;
  onAddSavedActionPlan: () => void;
  onExportActionPlan: () => void;
  onOpenImportActionPlan: () => void;
  onImportActionPlan: (file: File | null) => void;
  onDeleteSavedActionPlan: () => void;
  importInputRef: React.RefObject<HTMLInputElement>;
  onChange: (config: V10Config) => void;
}) {
  const byID = useMemo<Record<string, V10Action>>(() => Object.fromEntries(actions.map((action) => [action.id, action])), [actions]);
  const legacySteps = (config.pipeline ?? []).filter((step) => systemPipelineActions.has(step.action));
  const apiSteps = (config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action));
  const [expandedSteps, setExpandedSteps] = useState<Record<number, boolean>>({});
  useEffect(() => {
    setExpandedSteps({});
  }, [config.name]);
  const isExpanded = (index: number) => Boolean(expandedSteps[index]);
  const setAllExpanded = (expanded: boolean) => {
    setExpandedSteps(Object.fromEntries(apiSteps.map((_, index) => [index, expanded])));
  };
  const toggleStep = (index: number) => {
    setExpandedSteps((current) => ({ ...current, [index]: !current[index] }));
  };
  const handleStepHeaderKey = (event: React.KeyboardEvent, index: number) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    toggleStep(index);
  };
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
    setExpandedSteps((current) => ({ ...current, [index]: Boolean(current[target]), [target]: Boolean(current[index]) }));
    onChange({ ...config, pipeline: next });
  };
  return (
    <div className="v10-pipeline">
      <p className="readonly-notice">{m.pipeline.help}</p>
      <section className="v10-saved-plan-panel">
        <div className="v10-saved-plan-header">
          <h4>{m.pipeline.savedPlansTitle}</h4>
          <div className="button-row">
            <Button type="button" size="sm" variant="secondary" onClick={() => {
              onActionPlanNameChange(defaultActionPlanName(config.name));
              onShowSaveActionPlanChange(!showSaveActionPlan);
            }}>{m.pipeline.saveCurrent}</Button>
            <Button type="button" size="sm" variant="secondary" onClick={onExportActionPlan}>{m.pipeline.export}</Button>
            <Button type="button" size="sm" variant="secondary" onClick={onOpenImportActionPlan}>{m.pipeline.import}</Button>
            <input
              ref={importInputRef}
              type="file"
              accept="application/json,.json"
              className="hidden-file-input"
              onChange={(event) => onImportActionPlan(event.currentTarget.files?.[0] ?? null)}
            />
          </div>
        </div>
        {showSaveActionPlan && (
          <div className="v10-saved-plan-save">
            <label>{m.pipeline.planName}
              <input value={actionPlanName} onChange={(event) => onActionPlanNameChange(event.currentTarget.value)} placeholder={m.pipeline.planNamePlaceholder} />
            </label>
            <div className="button-row">
              <Button type="button" size="sm" onClick={onSaveActionPlan} disabled={!actionPlanName.trim()}>{m.save}</Button>
              <Button type="button" size="sm" variant="secondary" onClick={() => onShowSaveActionPlanChange(false)}>{messages.common.cancel}</Button>
            </div>
          </div>
        )}
        <div className="v10-saved-plan-load">
          <label>{m.pipeline.savedPlan}
            <select value={selectedSavedActionPlanID} onChange={(event) => onSelectedSavedActionPlanChange(event.currentTarget.value)} disabled={savedActionPlans.length === 0}>
              <option value="">{m.pipeline.noSavedPlan}</option>
              {savedActionPlans.map((plan) => <option key={plan.id} value={plan.id}>{plan.name}</option>)}
            </select>
          </label>
          <div className="button-row">
            <Button type="button" size="sm" variant="secondary" onClick={onAddSavedActionPlan} disabled={!selectedSavedActionPlanID}>{m.pipeline.addToCurrent}</Button>
            <Button type="button" size="sm" variant="danger" onClick={onDeleteSavedActionPlan} disabled={!selectedSavedActionPlanID}>{m.delete}</Button>
          </div>
        </div>
      </section>
      {legacySteps.length > 0 && (
        <div className="readonly-notice warning">
          <p>{m.pipeline.legacySystemActions}</p>
          <Button type="button" size="sm" variant="secondary" onClick={() => onChange({ ...config, pipeline: apiSteps })}>{m.pipeline.cleanSystemActions}</Button>
        </div>
      )}
      {actions.length === 0 && <p className="muted">{m.actionPlan.noActionsForProduct}</p>}
      {apiSteps.length > 0 && (
        <div className="button-row">
          <Button type="button" size="sm" variant="secondary" onClick={() => setAllExpanded(false)}>Tout réduire</Button>
          <Button type="button" size="sm" variant="secondary" onClick={() => setAllExpanded(true)}>Tout agrandir</Button>
        </div>
      )}
      {apiSteps.map((step, index) => {
        const action = byID[step.action];
        const fields = (action?.fields ?? []).filter((field) => !isActionFieldHidden(field, step.params ?? {}));
        const expanded = isExpanded(index);
        return (
          <div className="v10-pipeline-step" key={`${step.action}-${index}`}>
            <div className="v10-step-order">{index + 1}</div>
            <div className="v10-step-body">
              <div
                className="v10-pipeline-step-header clickable"
                role="button"
                tabIndex={0}
                aria-expanded={expanded}
                onClick={() => toggleStep(index)}
                onKeyDown={(event) => handleStepHeaderKey(event, index)}
              >
                <button type="button" className="v10-chevron" aria-label={expanded ? 'Réduire action' : 'Agrandir action'} aria-expanded={expanded} onClick={(event) => { event.stopPropagation(); toggleStep(index); }}>
                  {expanded ? '▾' : '▸'}
                </button>
                <div className="v10-pipeline-step-summary">
                  <strong>{step.label || action?.label || m.chooseAction}</strong>
                  <span className="muted">{step.action || action?.kind || m.chooseAction}</span>
                </div>
                <div className="button-row">
                  <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); move(index, -1); }}>{m.moveUp}</Button>
                  <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); move(index, 1); }}>{m.moveDown}</Button>
                  <Button type="button" size="sm" variant="danger" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onChange({ ...config, pipeline: apiSteps.filter((_, itemIndex) => itemIndex !== index) }); }}>{m.delete}</Button>
                </div>
              </div>
              {expanded && (
                <>
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
                          config={config}
                          params={step.params ?? {}}
                          key={field.name}
                          onChange={(value) => updateStep(index, { ...step, params: { ...(step.params ?? {}), [field.name]: value } })}
                        />
                      ))}
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        );
      })}
      <div className="button-row">
        <Button type="button" variant="secondary" onClick={() => {
          const action = actions[0];
          setExpandedSteps((current) => ({ ...current, [apiSteps.length]: true }));
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

function ActionFieldInput({ field, value, config, params, disabledOptionValues, onChange }: { field: V10Action['fields'][number]; value: unknown; config: V10Config; params: Record<string, unknown>; disabledOptionValues?: Set<string>; onChange: (value: unknown) => void }) {
  const label = <FieldLabel field={field} />;
  const options = actionFieldOptions(field, config);
  if (options.length > 0) {
    return (
      <label>{label}
        <select value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)}>
          <option value=""></option>
          {options.map((option) => <option key={option.value} value={option.value} disabled={disabledOptionValues?.has(option.value)}>{option.label}</option>)}
        </select>
      </label>
    );
  }
  if (field.type === 'bool') {
    return <label className="checkbox-row"><input type="checkbox" checked={Boolean(value)} onChange={(event) => onChange(event.currentTarget.checked)} />{label}</label>;
  }
  if (field.type === 'string[]') {
    return <label>{label}<input value={Array.isArray(value) ? value.join(',') : ''} onChange={(event) => onChange(event.currentTarget.value.split(',').map((item) => item.trim()).filter(Boolean))} /></label>;
  }
  if (field.type === 'number[]') {
    return <ActionNumberArrayField field={field} value={value} onChange={onChange} />;
  }
  if (field.type === 'object[]') {
    return <ActionObjectArrayField field={field} value={value} config={config} params={params} onChange={onChange} />;
  }
  if (field.type === 'text') {
    return <label>{label}<textarea value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} />{field.description && <span className="muted">{field.description}</span>}</label>;
  }
  if (field.type === 'number') {
    return <label>{label}<input type="number" min={field.min} value={typeof value === 'number' ? value : ''} onChange={(event) => onChange(event.currentTarget.value === '' ? '' : Number(event.currentTarget.value))} /></label>;
  }
  return <label>{label}<input value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} /></label>;
}

function ActionNumberArrayField({ field, value, onChange }: { field: V10Action['fields'][number]; value: unknown; onChange: (value: unknown) => void }) {
  const rows = Array.isArray(value) ? value.map((item) => typeof item === 'number' ? item : Number(item)).filter(Number.isFinite) : [];
  const min = field.itemMin;
  const hasInvalidValue = min !== undefined && rows.some((row) => row < min);
  const updateRow = (index: number, nextValue: number) => {
    onChange(rows.map((row, rowIndex) => rowIndex === index ? nextValue : row));
  };
  return (
    <div className="span-2 v10-action-array">
      <FieldLabel field={field} />
      {rows.map((row, index) => (
        <div className="v10-action-number-row" key={index}>
          <input type="number" min={min} value={row} onChange={(event) => updateRow(index, Number(event.currentTarget.value))} />
          <Button type="button" size="sm" variant="danger" onClick={() => onChange(rows.filter((_, rowIndex) => rowIndex !== index))}>{m.delete}</Button>
        </div>
      ))}
      <Button type="button" size="sm" variant="secondary" onClick={() => onChange([...rows, min ?? 0])}>{m.addGroup}</Button>
      {hasInvalidValue && <span className="error">Les IDs groupes machine doivent être supérieurs à 0.</span>}
      {field.description && <span className="muted">{field.description}</span>}
    </div>
  );
}

function ActionObjectArrayField({ field, value, config, params, onChange }: { field: V10Action['fields'][number]; value: unknown; config: V10Config; params: Record<string, unknown>; onChange: (value: unknown) => void }) {
  const rows = Array.isArray(value) ? value.filter(isRecord) : [];
  const itemFields = field.itemFields ?? [];
  const uniqueFieldName = field.uniqueItemField;
  const uniqueField = uniqueFieldName ? itemFields.find((itemField) => itemField.name === uniqueFieldName) : undefined;
  const uniqueOptions = uniqueField ? actionFieldOptions(uniqueField, config) : [];
  const usedUniqueValues = new Set(rows.map((row) => stringValue(row[uniqueFieldName ?? ''])).filter(Boolean));
  const allUniqueValuesUsed = Boolean(uniqueFieldName && uniqueOptions.length > 0 && uniqueOptions.every((option) => usedUniqueValues.has(option.value)));
  const updateRow = (index: number, key: string, nextValue: unknown) => {
    onChange(rows.map((row, rowIndex) => rowIndex === index ? { ...row, [key]: nextValue } : row));
  };
  const addRow = () => {
    const row: Record<string, unknown> = {};
    for (const itemField of itemFields) {
      if (itemField.default !== undefined && itemField.default !== null) {
        row[itemField.name] = itemField.default;
      }
    }
    if (uniqueFieldName && uniqueOptions.length > 0) {
      const available = uniqueOptions.find((option) => !usedUniqueValues.has(option.value));
      if (available) {
        row[uniqueFieldName] = available.value;
      }
    }
    onChange([...rows, row]);
  };
  return (
    <div className="span-2 v10-action-array">
      <FieldLabel field={field} />
      {rows.map((row, index) => (
        <div className="v10-action-array-row" key={index}>
          {itemFields.filter((itemField) => !actionFieldHidden(itemField, { ...params, ...row })).map((itemField) => {
            const disabledOptionValues = itemField.name === uniqueFieldName
              ? new Set(rows
                .filter((_, rowIndex) => rowIndex !== index)
                .map((otherRow) => stringValue(uniqueFieldName ? otherRow[uniqueFieldName] : undefined))
                .filter(Boolean))
              : undefined;
            return (
              <ActionFieldInput
                key={itemField.name}
                field={itemField}
                value={row[itemField.name]}
                config={config}
                params={{ ...params, ...row }}
                disabledOptionValues={disabledOptionValues}
                onChange={(nextValue) => updateRow(index, itemField.name, nextValue)}
              />
            );
          })}
          <Button type="button" size="sm" variant="danger" onClick={() => onChange(rows.filter((_, rowIndex) => rowIndex !== index))}>{m.delete}</Button>
        </div>
      ))}
      <Button type="button" size="sm" variant="secondary" onClick={addRow} disabled={allUniqueValuesUsed}>{m.addStep}</Button>
      {allUniqueValuesUsed && <span className="muted">Toutes les clés de configuration disponibles sont déjà utilisées.</span>}
    </div>
  );
}

function FieldLabel({ field }: { field: V10Action['fields'][number] }) {
  return (
    <span className="v10-field-label">
      {field.label}
      {field.required && <RequiredDot />}
    </span>
  );
}

function actionFieldOptions(field: V10Action['fields'][number], config: V10Config): Array<{ label: string; value: string }> {
  if (field.options?.length) {
    return field.options;
  }
  if (field.optionsSource === 'connectors') {
    return Object.keys(config.gedixConfig.connectors ?? {})
      .sort((left, right) => left.localeCompare(right))
      .map((name) => ({ label: name, value: name }));
  }
  return [];
}

function actionFieldHidden(field: V10Action['fields'][number], params: Record<string, unknown>): boolean {
  if (Object.entries(field.hiddenWhen ?? {}).some(([key, expected]) => actionValuesEqual(params[key], expected))) {
    return true;
  }
  return (field.hiddenWhenAny ?? []).some((group) => actionHiddenGroupMatches(group, params));
}

function isActionFieldHidden(field: V10Action['fields'][number], params: Record<string, unknown>): boolean {
  return actionFieldHidden(field, params);
}

function actionHiddenGroupMatches(group: Record<string, unknown>, params: Record<string, unknown>): boolean {
  const entries = Object.entries(group);
  return entries.length > 0 && entries.every(([key, expected]) => actionValuesEqual(params[key], expected));
}

function actionValuesEqual(left: unknown, right: unknown): boolean {
  if (left === right) {
    return true;
  }
  if (typeof left === 'boolean' || typeof right === 'boolean') {
    return String(left).toLowerCase() === String(right).toLowerCase();
  }
  const leftNumber = typeof left === 'number' ? left : Number(left);
  const rightNumber = typeof right === 'number' ? right : Number(right);
  if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) {
    return leftNumber === rightNumber;
  }
  return String(left) === String(right);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

type ExecutableCommandOption = {
  kind: ExecutableCommandTargetKind;
  name: string;
  label: string;
};

type ExecutableCommandGroup = {
  label: string;
  options: ExecutableCommandOption[];
};

function ModuleCommandPanel({ config, product, disabled, onRun, showTitle = true }: { config: V10Config; product: V10Product; disabled: boolean; onRun: (targetKind: ExecutableCommandTargetKind, targetName: string, command: string) => void; showTitle?: boolean }) {
  const groups = executableCommandGroups(config, product);
  const options = groups.flatMap((group) => group.options);
  const [selectedValue, setSelectedValue] = useState(options[0] ? executableCommandOptionValue(options[0]) : '');
  const [command, setCommand] = useState('');
  const invalid = executableCommandHasUnclosedQuote(command);
  const selectedOption = options.find((option) => executableCommandOptionValue(option) === selectedValue);

  useEffect(() => {
    if (!selectedValue || !options.some((option) => executableCommandOptionValue(option) === selectedValue)) {
      setSelectedValue(options[0] ? executableCommandOptionValue(options[0]) : '');
    }
  }, [options.map(executableCommandOptionValue).join('|'), selectedValue]);

  return (
    <div className="v10-module-command">
      {showTitle && <h4>{m.moduleCommand.title}</h4>}
      <p className="muted">{m.moduleCommand.help}</p>
      <div className="form-grid v10-form-grid">
        <label>{m.moduleCommand.target}
          <select value={selectedValue} onChange={(event) => setSelectedValue(event.currentTarget.value)}>
            <ExecutableCommandOptions groups={groups} valueFor={executableCommandOptionValue} />
          </select>
        </label>
        <label>{m.moduleCommand.command}
          <input value={command} placeholder={m.moduleCommand.commandPlaceholder} onChange={(event) => setCommand(event.currentTarget.value)} />
        </label>
      </div>
      {invalid && <p className="error">{m.moduleCommand.unclosedQuote}</p>}
      <div className="button-row">
        <Button type="button" variant="secondary" disabled={disabled || !selectedOption || !command.trim() || invalid} onClick={() => selectedOption && onRun(selectedOption.kind, selectedOption.name, command)}>
          {m.moduleCommand.run}
        </Button>
      </div>
    </div>
  );
}

function ExecutableCommandOptions({ groups, excludedNames = [], valueFor = (option: ExecutableCommandOption) => option.name }: { groups: ExecutableCommandGroup[]; excludedNames?: string[]; valueFor?: (option: ExecutableCommandOption) => string }) {
  const excluded = new Set(excludedNames);
  return <>
    {groups.map((group) => {
      const options = group.options.filter((option) => !excluded.has(option.name));
      if (!options.length) {
        return null;
      }
      return (
        <optgroup key={group.label} label={group.label}>
          {options.map((option) => <option key={valueFor(option)} value={valueFor(option)}>{option.label}</option>)}
        </optgroup>
      );
    })}
  </>;
}

function ExecutionPanel({ config, product, busy, runState, execution, logs, selectedLog, onConfigChange, onCreate, onUpdate, onConfigure, onStart, onOpenMaquette, onRunPipeline, onRunExecutableCommand, onKill, onRefreshLogs, onReadLog }: {
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
  onRunExecutableCommand: (targetKind: ExecutableCommandTargetKind, targetName: string, command: string) => void;
  onKill: () => void;
  onRefreshLogs: () => void;
  onReadLog: (logFile: string) => void;
}) {
  const currentLog = execution?.log || execution?.output || execution?.status || '';
  const disabled = busy || runState === 'running';
  return (
    <div className="v10-execution">
      <details className="v10-execution-section v10-collapsible-section">
        <summary>{m.execution.startupOptionsTitle}</summary>
        <DebugTargetsEditor config={config} product={product} onChange={onConfigChange} />
      </details>
      <details className="v10-execution-section v10-collapsible-section">
        <summary>{m.moduleCommand.title}</summary>
        <ModuleCommandPanel config={config} product={product} disabled={disabled} onRun={onRunExecutableCommand} showTitle={false} />
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
    gedixConfig: materializeProductServices({ fqdn: '', port: 80, services: {}, connectors: {}, agents: {}, adaptors: {} }, productDefinition),
    runtime: { debugTargets: [], openConsole: true },
    groupName: '',
    pipeline: [],
  } as V10Config);
}

function materializeProductServices(gedixConfig: V10Config['gedixConfig'], product?: V10Product): V10Config['gedixConfig'] {
  const services = { ...(gedixConfig.services ?? {}) };
  for (const service of product?.services ?? []) {
    services[service.name] = {
      dbType: services[service.name]?.dbType || 'sqlite',
      dbDsn: services[service.name]?.dbDsn ?? '',
      extraKeys: services[service.name]?.extraKeys ?? {},
    };
  }
  return { ...gedixConfig, services };
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
      agents: gedixConfig.agents ?? {},
      adaptors: gedixConfig.adaptors ?? {},
      units: gedixConfig.units ?? {},
    },
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

function validateConfig(config: V10Config, products: V10Product[]): string {
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
  if (hasDuplicateConnector(Object.keys(allUnitsForConfig(config, productFor(config.product, []))).map((name) => ({ id: name, name, module: '', rawConfig: '' })))) {
    return m.duplicateConnector;
  }
  const serviceDsnValidation = validateServiceDsns(config, productFor(config.product, products));
  if (serviceDsnValidation) {
    return serviceDsnValidation;
  }
  return '';
}

function validatePipelineRequiredFields(config: V10Config, actions: V10Action[]): string | null {
  if (actions.length === 0) {
    return null;
  }
  const byID = new Map(actions.map((action) => [action.id, action]));
  for (const [index, step] of (config.pipeline ?? []).entries()) {
    const action = byID.get(step.action);
    if (!action) {
      continue;
    }
    const params = step.params ?? {};
    for (const field of action.fields ?? []) {
      if (actionFieldHidden(field, params)) {
        continue;
      }
      const value = params[field.name] ?? field.default;
      if (field.type === 'number' && field.min !== undefined) {
        const validation = validateNumberMin(value, field.min, field.label?.trim() || field.name, field.required);
        if (validation) {
          return validation;
        }
      }
      if (field.type === 'number[]' && field.itemMin !== undefined) {
        const validation = validateNumberArrayMin(value, field.itemMin);
        if (validation) {
          return validation;
        }
      }
      if (field.type === 'object[]' && field.uniqueItemField) {
        const validation = validateUniqueObjectArrayField(field, value);
        if (validation) {
          return validation;
        }
      }
      if (!field.required) {
        continue;
      }
      if (isPipelineRequiredValueEmpty(value)) {
        const fieldLabel = field.label?.trim() || field.name;
        return `Étape ${index + 1} - ${step.action} : le champ ${fieldLabel} est obligatoire et ne peut pas être vide.`;
      }
    }
  }
  return null;
}

function isPipelineRequiredValueEmpty(value: unknown): boolean {
  if (value === undefined || value === null) {
    return true;
  }
  if (typeof value === 'string') {
    return value.trim() === '';
  }
  if (Array.isArray(value)) {
    return value.length === 0;
  }
  return false;
}

function validateNumberArrayMin(value: unknown, min: number): string {
  const values = normalizeNumberArray(value);
  if (!values.valid || values.items.some((item) => item < min)) {
    return `Les IDs groupes machine doivent être supérieurs ou égaux à ${formatNumberForMessage(min)}.`;
  }
  return '';
}

function validateNumberMin(value: unknown, min: number, label: string, required = false): string {
  if (value === undefined || value === null || value === '') {
    return required ? `Le champ "${label}" doit être supérieur ou égal à ${formatNumberForMessage(min)}.` : '';
  }
  const number = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(number) || number < min) {
    return `Le champ "${label}" doit être supérieur ou égal à ${formatNumberForMessage(min)}.`;
  }
  return '';
}

function formatNumberForMessage(value: number): string {
  return Number.isInteger(value) ? String(value) : String(value);
}

function normalizeNumberArray(value: unknown): { items: number[]; valid: boolean } {
  if (value === undefined || value === null || value === '') {
    return { items: [], valid: true };
  }
  if (Array.isArray(value)) {
    const items = value.map((item) => typeof item === 'number' ? item : Number(item));
    return { items, valid: items.every((item) => Number.isInteger(item)) };
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return { items: [], valid: true };
    }
    try {
      if (trimmed.startsWith('[')) {
        return normalizeNumberArray(JSON.parse(trimmed) as unknown);
      }
    } catch {
      return { items: [], valid: false };
    }
    const items = trimmed.split(',').map((item) => Number(item.trim()));
    return { items, valid: items.every((item) => Number.isInteger(item)) };
  }
  if (typeof value === 'number') {
    return { items: [value], valid: Number.isInteger(value) };
  }
  return { items: [], valid: false };
}

function validateUniqueObjectArrayField(field: V10Action['fields'][number], value: unknown): string {
  const rows = Array.isArray(value) ? value.filter(isRecord) : [];
  const uniqueFieldName = field.uniqueItemField;
  if (!uniqueFieldName) {
    return '';
  }
  const uniqueField = field.itemFields?.find((itemField) => itemField.name === uniqueFieldName);
  const options = uniqueField?.options ?? [];
  const allowedValues = new Set(options.map((option) => option.value));
  const seen = new Set<string>();
  for (const row of rows) {
    const itemValue = stringValue(row[uniqueFieldName]);
    if (!itemValue) {
      continue;
    }
    if (allowedValues.size > 0 && !allowedValues.has(itemValue)) {
      return `Dans "${field.label}", la valeur "${itemValue}" n'est pas autorisée.`;
    }
    if (seen.has(itemValue)) {
      return `Dans "${field.label}", la valeur "${itemValue}" ne peut être utilisée qu'une seule fois.`;
    }
    seen.add(itemValue);
  }
  return '';
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
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

function unitRowsFromConfig(config: V10Config, product: V10Product, unitKind: UnitKind): ConnectorFormRow[] {
  return Object.entries(unitsForConfig(config, product, unitKind)).map(([name, connector]) => ({
    id: makeID(),
    name,
    module: connector.module ?? '',
    rawConfig: connector.rawConfig,
  }));
}

function unitsForConfig(config: V10Config, product: V10Product, unitKind: UnitKind): Record<string, ConnectorConfig> {
  if (!productHasUnitKind(product, unitKind)) {
    return {};
  }
  const genericUnits = config.gedixConfig.units ?? {};
  const typedUnits = config.gedixConfig[unitConfigKey(unitKind)] ?? {};
  return { ...genericUnits, ...typedUnits };
}

function allUnitsForConfig(config: V10Config, product: V10Product): Record<string, ConnectorConfig> {
  return unitDefinitionsForProduct(product).reduce<Record<string, ConnectorConfig>>((items, definition) => ({ ...items, ...unitsForConfig(config, product, definition.kind) }), {});
}

export function executableCommandGroups(config: V10Config, product: V10Product, includeRoot = true): ExecutableCommandGroup[] {
  const groups: ExecutableCommandGroup[] = [];
  if (includeRoot) {
    groups.push({
      label: m.moduleCommand.groups.general,
      options: [
        { kind: 'root', name: 'gx.exe', label: 'gx.exe' },
        { kind: 'root', name: 'gx-front.exe', label: 'gx-front.exe' },
      ],
    });
  }
  const configuredServices = config.gedixConfig.services ?? {};
  const serviceOptions = product.services
    .filter((service) => Boolean(configuredServices[service.name]))
    .map((service) => ({ kind: 'service' as ExecutableCommandTargetKind, name: service.name, label: service.label || service.name }))
    .sort((left, right) => left.label.localeCompare(right.label));
  if (serviceOptions.length) {
    groups.push({ label: m.moduleCommand.groups.services, options: serviceOptions });
  }
  for (const definition of unitDefinitionsForProduct(product)) {
    const units = unitsForConfig(config, product, definition.kind);
    const options = Object.keys(units)
      .sort((left, right) => left.localeCompare(right))
      .map((name) => ({ kind: definition.kind as ExecutableCommandTargetKind, name, label: name }));
    if (options.length) {
      groups.push({ label: executableCommandUnitGroupLabel(definition.kind), options });
    }
  }
  return groups;
}

function executableCommandUnitGroupLabel(kind: UnitKind) {
  if (kind === 'agent') {
    return m.moduleCommand.groups.agents;
  }
  if (kind === 'adaptor') {
    return m.moduleCommand.groups.adaptors;
  }
  return m.moduleCommand.groups.connectors;
}

function executableCommandOptionValue(option: ExecutableCommandOption) {
  return `${option.kind}:${option.name}`;
}

function customArgumentsForTarget(targetArguments: string[]) {
  return targetArguments.map((argument) => argument.trim()).filter(Boolean).join(' ');
}

function productHasUnitKind(product: V10Product, unitKind: UnitKind): boolean {
  return unitDefinitionsForProduct(product).some((definition) => definition.kind === unitKind && Boolean(definition.cfgSectionName?.trim()));
}

function unitDefinitionsForProduct(product: V10Product): V10UnitDefinition[] {
  if (product.unitDefinitions?.length) {
    return product.unitDefinitions;
  }
  if ((product.unitKind === 'connector' || product.unitKind === 'agent' || product.unitKind === 'adaptor') && product.unitCfgSectionName?.trim()) {
    return [{
      kind: product.unitKind,
      singularLabel: product.unitSingularLabel,
      pluralLabel: product.unitPluralLabel,
      cfgSectionName: product.unitCfgSectionName,
      folderPrefix: product.unitFolderPrefix,
      runtimeExecutablePattern: product.unitRuntimeExecutablePattern ?? product.unitExecutableName,
      moduleExecutablePattern: product.unitModuleExecutablePattern,
    }];
  }
  return [];
}

function unitDefinitionForKind(product: V10Product, unitKind: UnitKind): V10UnitDefinition {
  return unitDefinitionsForProduct(product).find((definition) => definition.kind === unitKind) ?? {
    kind: unitKind,
    singularLabel: unitKind === 'adaptor' ? m.units.adaptor : unitKind === 'agent' ? m.units.agent : m.units.connector,
    pluralLabel: unitKind === 'adaptor' ? m.units.adaptors : unitKind === 'agent' ? m.units.agents : m.units.connectors,
    cfgSectionName: unitKind === 'adaptor' ? 'adaptors' : unitKind === 'agent' ? 'agents' : 'connectors',
    folderPrefix: unitKind === 'adaptor' ? 'adaptor-' : unitKind === 'agent' ? 'agent-' : 'connector-',
  };
}

function unitConfigKey(unitKind: UnitKind): 'connectors' | 'agents' | 'adaptors' {
  if (unitKind === 'agent') {
    return 'agents';
  }
  if (unitKind === 'adaptor') {
    return 'adaptors';
  }
  return 'connectors';
}

function scanUnitsForKind(result: { units?: Array<{ name: string; module?: string; rawConfig: string }>; connectors?: Array<{ name: string; module?: string; rawConfig: string }>; agents?: Array<{ name: string; module?: string; rawConfig: string }>; adaptors?: Array<{ name: string; module?: string; rawConfig: string }> }, unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return result.agents ?? result.units ?? [];
  }
  if (unitKind === 'adaptor') {
    return result.adaptors ?? [];
  }
  return result.connectors ?? result.units ?? [];
}

function connectorTabLabel(product: V10Product) {
  return productHasUnitKind(product, 'agent') && !productHasUnitKind(product, 'connector') ? m.units.agents : m.units.connectors;
}

function unitNameLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.agentName;
  }
  if (unitKind === 'adaptor') {
    return m.units.adaptorName;
  }
  return m.units.connectorName;
}

function unitScanLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.scanAgents;
  }
  if (unitKind === 'adaptor') {
    return m.units.scanAdaptors;
  }
  return m.units.scanConnectors;
}

function unitAddLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.addAgent;
  }
  if (unitKind === 'adaptor') {
    return m.units.addAdaptor;
  }
  return m.units.addConnector;
}

function unitSearchPlaceholder(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.search.agentPlaceholder;
  }
  if (unitKind === 'adaptor') {
    return m.search.adaptorPlaceholder;
  }
  return m.search.connectorPlaceholder;
}

function normalizeSearch(value: string) {
  return value.trim().toLowerCase();
}

function matchesSearch(normalizedQuery: string, values: Array<string | undefined>) {
  if (!normalizedQuery) {
    return true;
  }
  return values
    .filter((value): value is string => Boolean(value))
    .some((value) => value.toLowerCase().includes(normalizedQuery));
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
    unitSingularLabel: 'connector',
    unitPluralLabel: 'connectors',
    unitCfgSectionName: 'connectors',
    unitFolderPrefix: 'connector-',
    unitExecutableName: 'gx-connector.exe',
    unitModuleExecutablePattern: 'gx-module-<unitName>.exe',
  };
}

function unitHelp(definition: V10UnitDefinition) {
  return m.unitHelp
    .replace('{{unit}}', definition.singularLabel)
    .replace('{{section}}', definition.cfgSectionName)
    .replace('{{unitPlaceholder}}', definition.singularLabel);
}

function executableCommandHasUnclosedQuote(command: string) {
  return [...command].filter((char) => char === '"').length % 2 === 1;
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

function deepClonePipeline(steps: PipelineStep[]): PipelineStep[] {
  return JSON.parse(JSON.stringify(steps)) as PipelineStep[];
}

function extractImportedActionPlanSteps(payload: ImportableActionPlan): PipelineStep[] {
  if (!payload || typeof payload !== 'object') {
    throw new Error(m.pipeline.invalidImportFile);
  }
  const steps = Array.isArray(payload.actions) ? payload.actions : (Array.isArray(payload.pipeline) ? payload.pipeline : []);
  return steps.filter(isPipelineStep).map((step) => ({
    action: step.action,
    label: typeof step.label === 'string' ? step.label : '',
    params: step.params && typeof step.params === 'object' && !Array.isArray(step.params) ? step.params : {},
  }));
}

function isPipelineStep(value: unknown): value is PipelineStep {
  return Boolean(value && typeof value === 'object' && typeof (value as PipelineStep).action === 'string');
}

function safeFileName(value: string): string {
  const invalidFileNameChars = /[<>:"/\\|?*]/;
  const cleaned = Array.from(value.trim())
    .map((char) => {
      const code = char.charCodeAt(0);
      return code <= 31 || invalidFileNameChars.test(char) ? '-' : char;
    })
    .join('')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^\.+$/, '');
  return cleaned || 'plan-actions';
}

function defaultActionPlanName(maquetteName: string): string {
  const name = maquetteName.trim();
  return name ? formatMessage(m.pipeline.defaultPlanName, { name }) : '';
}

function formatMessage(template: string, values: Record<string, string>): string {
  return Object.entries(values).reduce((message, [key, value]) => message.split(`{{${key}}}`).join(value), template);
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

