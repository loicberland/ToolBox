import React, { useEffect, useRef, useState } from 'react';
import {
  DBTemplate,
  ExecutionResponse,
  LogSummary,
  MaquetteGroup,
  MaquetteSummary,
  DEFAULT_V10_PRODUCT_ID,
  V10Action,
  V10Config,
  V10Product,
  V10SavedActionPlan,
  UnitKind,
  ExecutableCommandTargetKind,
  v10LabApi,
} from './api/v10Lab';
import { Button } from '../../shared/components/ui/Button';
import { ConfirmDialog } from '../../shared/components/ui/ConfirmDialog';
import { Toast } from '../../shared/components/ui/Toast';
import { messages } from '../../i18n';
import { RequiredDot } from './components/form/RequiredDot';
import { validateServiceDsns } from './validation/serviceValidation';
import { DuplicateMaquetteDialog } from './components/dialogs/DuplicateMaquetteDialog';
import { LocalErrorBoundary } from './components/LocalErrorBoundary';
import { MaquetteGeneralForm } from './components/maquette-detail/tabs/GeneralTab';
import { GedixForm } from './components/maquette-detail/tabs/GedixConfigTab';
import { ServicesForm } from './components/maquette-detail/tabs/ServicesTab';
import { AdaptorsTab } from './components/maquette-detail/tabs/AdaptorsTab';
import { ConnectorsTab } from './components/maquette-detail/tabs/ConnectorsTab';
import { ApiTokenEditor, PipelineBuilder } from './components/maquette-detail/tabs/ActionPlanTab';
import { ExecutionPanel } from './components/maquette-detail/tabs/ExecutionLogsTab';
import { JsonTab } from './components/maquette-detail/tabs/JsonTab';
import { MaquetteListView } from './views/MaquetteListView';
import {
  connectorTabLabel,
  defaultActionPlanName,
  defaultConfig,
  delay,
  deepClonePipeline,
  executableCommandHasUnclosedQuote,
  extractImportedActionPlanSteps,
  filepathExt,
  formatMessage,
  maquetteJSONFileName,
  normalizeConfig,
  normalizePipelineStepsForActionDefinitions,
  parentPath,
  prettyJSONForDownload,
  productFor,
  productHasUnitKind,
  safeFileName,
  scanUnitsForKind,
  stringsEqual,
  unitConfigKey,
  unitDefinitionForKind,
  validateConfig,
  validatePipelineRequiredFields,
  type ImportableActionPlan,
  type RunState,
} from './utils/v10LabUtils';
export { executableCommandGroups, maquetteJSONFileName, normalizeActionParamsForSave, normalizePipelineStepsForActionDefinitions, prettyJSONForDownload } from './utils/v10LabUtils';

const m = messages.v10Lab;
const tabs = [m.tabs.general, m.tabs.gedix, m.tabs.services, m.tabs.adaptors, m.tabs.connectors, m.tabs.pipeline, m.tabs.execution, m.tabs.json] as const;
const systemPipelineActions = new Set(['install-env', 'configure-gedix-cfg', 'start-maquette', 'start-services', 'kill-gx-processes', 'update-env']);
type Tab = typeof tabs[number];
type BeforeLeaveHandler = () => Promise<boolean>;

export function V10LabModule({ onBeforeLeaveChange }: { onBeforeLeaveChange?: (handler: BeforeLeaveHandler | null) => void }) {
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
  const [importPath, setImportPath] = useState('');
  const [importConfig, setImportConfig] = useState<V10Config | null>(null);
  const [importName, setImportName] = useState('');
  const [importGroupName, setImportGroupName] = useState('');
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
  const [deleteDirectory, setDeleteDirectory] = useState(false);
  const [confirmKill, setConfirmKill] = useState(false);
  const [confirmUpdate, setConfirmUpdate] = useState(false);
  const [duplicateSource, setDuplicateSource] = useState<MaquetteSummary | null>(null);
  const [duplicateName, setDuplicateName] = useState('');
  const [duplicateParentPath, setDuplicateParentPath] = useState('');
  const [duplicateCopyData, setDuplicateCopyData] = useState(true);
  const [duplicating, setDuplicating] = useState(false);
  const [execution, setExecution] = useState<ExecutionResponse | null>(null);
  const currentMaquetteRef = useRef<HTMLElement | null>(null);
  const maquetteSelectorRef = useRef<HTMLDivElement | null>(null);
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
      const nextConfig = normalizeConfig({ ...config, pipeline: normalizePipelineStepsForActionDefinitions(config.pipeline ?? [], actions) });
      const next = normalizeConfig(await v10LabApi.updateMaquette(selectedName || oldName, nextConfig));
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
    const apiSteps = normalizePipelineStepsForActionDefinitions((config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action)), actions);
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
    updateConfig({ ...config, pipeline: [...apiSteps, ...normalizePipelineStepsForActionDefinitions(deepClonePipeline(plan.actions), actions)] });
    setMessage('Plan d\'actions ajouté au plan actuel.');
  }

  function exportCurrentActionPlan() {
    if (!config) {
      return;
    }
    const apiSteps = normalizePipelineStepsForActionDefinitions((config.pipeline ?? []).filter((step) => !systemPipelineActions.has(step.action)), actions);
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
      updateConfig({ ...config, pipeline: [...legacySteps, ...normalizePipelineStepsForActionDefinitions(deepClonePipeline(importedSteps), actions)] });
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
      await v10LabApi.deleteMaquette(name, deleteDirectory);
      setConfirmDelete(null);
	  setDeleteDirectory(false);
      if (selectedName === name) {
        setSelectedName('');
        setConfig(null);
      }
      await reloadList();
	  setMessage(deleteDirectory ? m.deletedWithDirectory : m.deleted);
    });
  }

  function openDeleteConfirmation(name: string) {
    setDeleteDirectory(false);
    setConfirmDelete(name);
  }

  function cancelDeleteConfirmation() {
    setDeleteDirectory(false);
    setConfirmDelete(null);
  }

  function startDuplicate(source: MaquetteSummary) {
    const existing = new Set(maquettes.map((item) => item.name.toLocaleLowerCase()));
    const base = `${source.name}_copie`;
    let name = base;
    for (let index = 2; existing.has(name.toLocaleLowerCase()); index += 1) name = `${base}_${index}`;
    setDuplicateSource(source); setDuplicateName(name); setDuplicateParentPath(parentPath(source.targetPath)); setDuplicateCopyData(true);
  }

  async function duplicateMaquette() {
    if (!duplicateSource || duplicating) return;
    if (config && isDirty && !(await saveCurrent())) return;
    setDuplicating(true); setError('');
    try {
      const created = normalizeConfig(await v10LabApi.duplicateMaquette(duplicateSource.name, { name: duplicateName, parentPath: duplicateParentPath, copyData: duplicateCopyData }));
      setDuplicateSource(null); await reloadList();
      if (created.groupName) setOpenGroups((current) => ({ ...current, [created.groupName!]: true }));
      await openMaquette(created.name); setMessage(m.duplicateSuccess);
    } catch (err) { showError(err instanceof Error ? err.message : m.duplicateError); } finally { setDuplicating(false); }
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
      setMessage(m.jsonChangesApplied);
      setError('');
    } catch (err) {
      showError(err instanceof Error ? err.message : m.invalidJson);
    }
  }

  function downloadJSON() {
    if (!config) {
      return;
    }
    try {
      const content = prettyJSONForDownload(jsonText);
      const blob = new Blob([content], { type: 'application/json;charset=utf-8' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = maquetteJSONFileName(config.name);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      setMessage(m.jsonDownloaded);
    } catch {
      showError(m.invalidJson);
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

  function resetImportJSON() {
    setImportPath('');
    setImportConfig(null);
    setImportName('');
    setImportGroupName('');
  }

  function matchingGroupName(name: string | undefined) {
    const normalized = name?.trim().replace(/\s+/g, ' ').toLocaleLowerCase();
    return normalized ? groups.find((group) => group.name.trim().replace(/\s+/g, ' ').toLocaleLowerCase() === normalized)?.name ?? '' : '';
  }

  async function selectImportJSON() {
    await run(async () => {
      const selected = await v10LabApi.selectImportJSONPath();
      if (selected.cancelled || !selected.path) {
        return;
      }
      const preview = await v10LabApi.previewImportJSON(selected.path);
      const next = normalizeConfig(preview.config);
      setImportPath(preview.path);
      setImportConfig(next);
      setImportName(next.name);
      setImportGroupName(matchingGroupName(next.groupName));
    });
  }

  async function confirmImportJSON() {
    if (!importConfig || !importName.trim()) {
      return;
    }
    await run(async () => {
      const name = importName.trim();
      await v10LabApi.importJSON(importPath, name, importGroupName);
      resetImportJSON();
      await reloadList();
      if (importGroupName) {
        setOpenGroups((current) => ({ ...current, [importGroupName]: true }));
      } else {
        setOpenUngrouped(true);
      }
      await openMaquette(name);
      setMessage(m.importJSON.imported);
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
      <div ref={maquetteSelectorRef}>
        <MaquetteListView
          maquettes={maquettes}
          groups={groups}
          groupedMaquettes={groupedMaquettes}
          ungroupedMaquettes={ungroupedMaquettes}
          selectedName={selectedName}
          busy={busy}
          openGroups={openGroups}
          openUngrouped={openUngrouped}
          newGroupName={newGroupName}
          onNewGroupNameChange={setNewGroupName}
          onReload={() => void reloadList()}
          onCreateGroup={() => void createGroup()}
          onToggleGroup={toggleGroup}
          onToggleUngrouped={() => setOpenUngrouped((value) => !value)}
          onToggleMaquette={toggleMaquette}
          onDuplicate={startDuplicate}
          onAddToGroup={(groupName) => {
            setDraft({ ...defaultConfig(currentProduct.id, currentProduct), groupName });
            setShowCreate(true);
            setOpenGroups((current) => ({ ...current, [groupName]: true }));
          }}
          onAddUngrouped={() => {
            setDraft({ ...defaultConfig(currentProduct.id, currentProduct), groupName: '' });
            setShowCreate(true);
            setOpenUngrouped(true);
          }}
          onDeleteGroup={(groupName) => void deleteGroup(groupName)}
        />
      </div>
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
          <Button type="button" variant="secondary" onClick={() => void importExistingMaquettes()} disabled={busy}>Scanner</Button>
          <Button type="button" variant="secondary" onClick={() => void selectImportJSON()} disabled={busy}>Importer</Button>
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

      {importConfig && (
        <section className="ui-card v10-section">
          <div className="ui-card-header">
            <h3>Importer une maquette JSON</h3>
          </div>
          <label>Fichier JSON
            <input value={importPath} readOnly />
          </label>
          <label>Nom de la maquette <RequiredDot />
            <input value={importName} onChange={(event) => setImportName(event.currentTarget.value)} />
          </label>
          <label>Groupe
            <select value={importGroupName} onChange={(event) => setImportGroupName(event.currentTarget.value)}>
              <option value="">Sans groupe</option>
              {groups.map((group) => <option key={group.name} value={group.name}>{group.name}</option>)}
            </select>
          </label>
          <div className="button-row end">
            <Button type="button" variant="secondary" onClick={resetImportJSON} disabled={busy}>{messages.common.cancel}</Button>
            <Button type="button" onClick={() => void confirmImportJSON()} disabled={busy || !importName.trim()}>Importer</Button>
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
              <Button type="button" variant="secondary" onClick={() => startDuplicate({ name: config.name, product: config.product, targetPath: config.maquette.targetPath, appName: config.maquette.appName, existsOnDisk: false, groupName: config.groupName })} disabled={busy}>{m.duplicate}</Button>
              <Button type="button" onClick={() => void saveCurrent()} disabled={busy}>{m.save}</Button>
              <Button type="button" variant="danger" onClick={() => openDeleteConfirmation(config.name)} disabled={busy}>{m.delete}</Button>
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
            <AdaptorsTab config={config} product={currentProduct} onChange={updateConfig} onScanCfg={(kind, file, importExistingKeys, replaceExistingUnits) => void scanCfg(kind, file, importExistingKeys, replaceExistingUnits)} />
          )}
          {activeTab === m.tabs.connectors && (productHasUnitKind(currentProduct, 'connector') || productHasUnitKind(currentProduct, 'agent')) && (
            <ConnectorsTab config={config} product={currentProduct} onChange={updateConfig} onScanCfg={(kind, file, importExistingKeys, replaceExistingUnits) => void scanCfg(kind, file, importExistingKeys, replaceExistingUnits)} />
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
              onCreate={() => void runSystemAction('install-env')}
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
            <JsonTab
              jsonText={jsonText}
              execution={execution}
              busy={busy}
              onJsonTextChange={setJsonText}
              onCopy={() => void navigator.clipboard?.writeText(jsonText)}
              onApply={() => void saveJSON()}
              onValidate={() => void validateCurrent()}
              onDownload={downloadJSON}
            />
          )}
        </section>
      )}

      {(!config || showMaquetteSelector) && renderMaquetteListSection()}

      <ConfirmDialog
        open={confirmDelete !== null}
        title={m.deleteTitle}
        message={m.deleteMessage}
        confirmLabel={m.delete}
        onCancel={cancelDeleteConfirmation}
        onConfirm={() => confirmDelete && void deleteMaquette(confirmDelete)}
      >
        <label className="duplicate-copy-data"><input type="checkbox" checked={deleteDirectory} disabled={busy} onChange={(event) => setDeleteDirectory(event.currentTarget.checked)} /><span>{m.deleteDirectory}</span></label>
        {deleteDirectory && config?.name === confirmDelete && <p className="warning-message">{formatMessage(m.deleteDirectoryWarning, { path: config.maquette.targetPath })}</p>}
      </ConfirmDialog>
      <DuplicateMaquetteDialog open={duplicateSource !== null} name={duplicateName} parentPath={duplicateParentPath} copyData={duplicateCopyData} busy={duplicating} error={error} onNameChange={setDuplicateName} onParentPathChange={setDuplicateParentPath} onCopyDataChange={setDuplicateCopyData} onCancel={() => !duplicating && setDuplicateSource(null)} onConfirm={() => void duplicateMaquette()} />
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
