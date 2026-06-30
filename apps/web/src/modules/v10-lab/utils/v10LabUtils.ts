import {
  ConnectorConfig,
  DEFAULT_V10_PRODUCT_ID,
  ExecutableCommandTargetKind,
  PipelineStep,
  ServiceDBConfig,
  UnitKind,
  V10Action,
  V10Config,
  V10Product,
  V10UnitDefinition,
} from '../api/v10Lab';
import { messages } from '../../../i18n';
import { validateServiceDsns } from '../validation/serviceValidation';

const m = messages.v10Lab;

export type RunState = 'idle' | 'running' | 'success' | 'failed';
export type ConnectorFormRow = {
  id: string;
  name: string;
  module: string;
  rawConfig: string;
};
export type ExtraKeyRow = {
  id: string;
  key: string;
  value: string;
};
export type ImportableActionPlan = {
  schema?: string;
  version?: number;
  name?: string;
  productId?: string;
  actions?: PipelineStep[];
  pipeline?: PipelineStep[];
};
export type ExecutableCommandOption = {
  kind: ExecutableCommandTargetKind;
  name: string;
  label: string;
};
export type ExecutableCommandGroup = {
  label: string;
  options: ExecutableCommandOption[];
};
export function actionFieldOptions(field: V10Action['fields'][number], config: V10Config): Array<{ label: string; value: string }> {
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

export function actionFieldHidden(field: V10Action['fields'][number], params: Record<string, unknown>): boolean {
  if (Object.entries(field.hiddenWhen ?? {}).some(([key, expected]) => actionValuesEqual(params[key], expected))) {
    return true;
  }
  return (field.hiddenWhenAny ?? []).some((group) => actionHiddenGroupMatches(group, params));
}

export function isActionFieldHidden(field: V10Action['fields'][number], params: Record<string, unknown>): boolean {
  return actionFieldHidden(field, params);
}

export function actionHiddenGroupMatches(group: Record<string, unknown>, params: Record<string, unknown>): boolean {
  const entries = Object.entries(group);
  return entries.length > 0 && entries.every(([key, expected]) => actionValuesEqual(params[key], expected));
}

export function actionValuesEqual(left: unknown, right: unknown): boolean {
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

export function normalizePipelineStepsForActionDefinitions(steps: PipelineStep[], actions: V10Action[]): PipelineStep[] {
  const byID = new Map(actions.map((action) => [action.id, action]));
  return steps.map((step) => {
    const action = byID.get(step.action);
    if (!action) {
      return { ...step, params: { ...(step.params ?? {}) } };
    }
    return { ...step, params: normalizeActionParamsForSave(action, step.params ?? {}) };
  });
}

export function normalizeActionParamsForSave(action: V10Action, params: Record<string, unknown>): Record<string, unknown> {
  const next = { ...params };
  pruneHiddenActionFieldValues(action.fields ?? [], next);
  return next;
}

export function pruneHiddenActionFieldValues(fields: V10Action['fields'], params: Record<string, unknown>, parentParams: Record<string, unknown> = {}) {
  const visibilityParams = actionFieldVisibilityParams(fields, params, parentParams);
  for (const field of fields) {
    if (actionFieldHidden(field, visibilityParams)) {
      delete params[field.name];
      continue;
    }
    if (field.type === 'object[]' && field.itemFields?.length) {
      params[field.name] = normalizeObjectArrayFieldValue(params[field.name], field.itemFields, visibilityParams);
    }
  }
}

export function normalizeObjectArrayFieldValue(value: unknown, itemFields: V10Action['fields'], parentParams: Record<string, unknown>): unknown {
  if (!Array.isArray(value)) {
    return value;
  }
  return value.map((row) => {
    if (!isRecord(row)) {
      return row;
    }
    const next = { ...row };
    pruneHiddenActionFieldValues(itemFields, next, parentParams);
    return next;
  });
}

export function actionFieldVisibilityParams(fields: V10Action['fields'], params: Record<string, unknown>, parentParams: Record<string, unknown>): Record<string, unknown> {
  const next = { ...parentParams, ...params };
  for (const field of fields) {
    if (next[field.name] === undefined && field.default !== undefined && field.default !== null) {
      next[field.name] = field.default;
    }
  }
  return next;
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}
export function defaultConfig(product = DEFAULT_V10_PRODUCT_ID, productDefinition?: V10Product): V10Config {
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

export function materializeProductServices(gedixConfig: V10Config['gedixConfig'], product?: V10Product): V10Config['gedixConfig'] {
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

export function normalizeConfig(config: V10Config): V10Config {
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

export function validateConfig(config: V10Config, products: V10Product[]): string {
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

export function validatePipelineRequiredFields(config: V10Config, actions: V10Action[]): string | null {
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
        return `Ã‰tape ${index + 1} - ${step.action} : le champ ${fieldLabel} est obligatoire et ne peut pas Ãªtre vide.`;
      }
    }
  }
  return null;
}

export function isPipelineRequiredValueEmpty(value: unknown): boolean {
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

export function validateNumberArrayMin(value: unknown, min: number): string {
  const values = normalizeNumberArray(value);
  if (!values.valid || values.items.some((item) => item < min)) {
    return `Les IDs groupes machine doivent Ãªtre supÃ©rieurs ou Ã©gaux Ã  ${formatNumberForMessage(min)}.`;
  }
  return '';
}

export function validateNumberMin(value: unknown, min: number, label: string, required = false): string {
  if (value === undefined || value === null || value === '') {
    return required ? `Le champ "${label}" doit Ãªtre supÃ©rieur ou Ã©gal Ã  ${formatNumberForMessage(min)}.` : '';
  }
  const number = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(number) || number < min) {
    return `Le champ "${label}" doit Ãªtre supÃ©rieur ou Ã©gal Ã  ${formatNumberForMessage(min)}.`;
  }
  return '';
}

export function formatNumberForMessage(value: number): string {
  return Number.isInteger(value) ? String(value) : String(value);
}

export function normalizeNumberArray(value: unknown): { items: number[]; valid: boolean } {
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

export function validateUniqueObjectArrayField(field: V10Action['fields'][number], value: unknown): string {
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
      return `Dans "${field.label}", la valeur "${itemValue}" n'est pas autorisÃ©e.`;
    }
    if (seen.has(itemValue)) {
      return `Dans "${field.label}", la valeur "${itemValue}" ne peut Ãªtre utilisÃ©e qu'une seule fois.`;
    }
    seen.add(itemValue);
  }
  return '';
}

export function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

export function formatDate(value: string) {
  return new Date(value).toLocaleString('fr-FR');
}

export function extraKeyRowsFromService(service: ServiceDBConfig): ExtraKeyRow[] {
  return Object.entries(service.extraKeys ?? {}).map(([key, value]) => ({
    id: makeID(),
    key,
    value,
  }));
}

export function filepathExt(filename: string) {
  const index = filename.lastIndexOf('.');
  return index >= 0 ? filename.slice(index) : '';
}

export function stringsEqual(left: string, right: string) {
  return left.localeCompare(right, undefined, { sensitivity: 'accent' }) === 0;
}

export function makeID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function unitRowsFromConfig(config: V10Config, product: V10Product, unitKind: UnitKind): ConnectorFormRow[] {
  return Object.entries(unitsForConfig(config, product, unitKind)).map(([name, connector]) => ({
    id: makeID(),
    name,
    module: connector.module ?? '',
    rawConfig: connector.rawConfig,
  }));
}

export function unitsForConfig(config: V10Config, product: V10Product, unitKind: UnitKind): Record<string, ConnectorConfig> {
  if (!productHasUnitKind(product, unitKind)) {
    return {};
  }
  const genericUnits = config.gedixConfig.units ?? {};
  const typedUnits = config.gedixConfig[unitConfigKey(unitKind)] ?? {};
  return { ...genericUnits, ...typedUnits };
}

export function allUnitsForConfig(config: V10Config, product: V10Product): Record<string, ConnectorConfig> {
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

export function executableCommandUnitGroupLabel(kind: UnitKind) {
  if (kind === 'agent') {
    return m.moduleCommand.groups.agents;
  }
  if (kind === 'adaptor') {
    return m.moduleCommand.groups.adaptors;
  }
  return m.moduleCommand.groups.connectors;
}

export function executableCommandOptionValue(option: ExecutableCommandOption) {
  return `${option.kind}:${option.name}`;
}

export function customArgumentsForTarget(targetArguments: string[]) {
  return targetArguments.map((argument) => argument.trim()).filter(Boolean).join(' ');
}

export function productHasUnitKind(product: V10Product, unitKind: UnitKind): boolean {
  return unitDefinitionsForProduct(product).some((definition) => definition.kind === unitKind && Boolean(definition.cfgSectionName?.trim()));
}

export function unitDefinitionsForProduct(product: V10Product): V10UnitDefinition[] {
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

export function unitDefinitionForKind(product: V10Product, unitKind: UnitKind): V10UnitDefinition {
  return unitDefinitionsForProduct(product).find((definition) => definition.kind === unitKind) ?? {
    kind: unitKind,
    singularLabel: unitKind === 'adaptor' ? m.units.adaptor : unitKind === 'agent' ? m.units.agent : m.units.connector,
    pluralLabel: unitKind === 'adaptor' ? m.units.adaptors : unitKind === 'agent' ? m.units.agents : m.units.connectors,
    cfgSectionName: unitKind === 'adaptor' ? 'adaptors' : unitKind === 'agent' ? 'agents' : 'connectors',
    folderPrefix: unitKind === 'adaptor' ? 'adaptor-' : unitKind === 'agent' ? 'agent-' : 'connector-',
  };
}

export function unitConfigKey(unitKind: UnitKind): 'connectors' | 'agents' | 'adaptors' {
  if (unitKind === 'agent') {
    return 'agents';
  }
  if (unitKind === 'adaptor') {
    return 'adaptors';
  }
  return 'connectors';
}

export function scanUnitsForKind(result: { units?: Array<{ name: string; module?: string; rawConfig: string }>; connectors?: Array<{ name: string; module?: string; rawConfig: string }>; agents?: Array<{ name: string; module?: string; rawConfig: string }>; adaptors?: Array<{ name: string; module?: string; rawConfig: string }> }, unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return result.agents ?? result.units ?? [];
  }
  if (unitKind === 'adaptor') {
    return result.adaptors ?? [];
  }
  return result.connectors ?? result.units ?? [];
}

export function connectorTabLabel(product: V10Product) {
  return productHasUnitKind(product, 'agent') && !productHasUnitKind(product, 'connector') ? m.units.agents : m.units.connectors;
}

export function unitNameLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.agentName;
  }
  if (unitKind === 'adaptor') {
    return m.units.adaptorName;
  }
  return m.units.connectorName;
}

export function unitScanLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.scanAgents;
  }
  if (unitKind === 'adaptor') {
    return m.units.scanAdaptors;
  }
  return m.units.scanConnectors;
}

export function unitAddLabel(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.units.addAgent;
  }
  if (unitKind === 'adaptor') {
    return m.units.addAdaptor;
  }
  return m.units.addConnector;
}

export function unitSearchPlaceholder(unitKind: UnitKind) {
  if (unitKind === 'agent') {
    return m.search.agentPlaceholder;
  }
  if (unitKind === 'adaptor') {
    return m.search.adaptorPlaceholder;
  }
  return m.search.connectorPlaceholder;
}

export function normalizeSearch(value: string) {
  return value.trim().toLowerCase();
}

export function matchesSearch(normalizedQuery: string, values: Array<string | undefined>) {
  if (!normalizedQuery) {
    return true;
  }
  return values
    .filter((value): value is string => Boolean(value))
    .some((value) => value.toLowerCase().includes(normalizedQuery));
}

export function productFor(productId: string | undefined, products: V10Product[]): V10Product {
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

export function unitHelp(definition: V10UnitDefinition) {
  return m.unitHelp
    .replace('{{unit}}', definition.singularLabel)
    .replace('{{section}}', definition.cfgSectionName)
    .replace('{{unitPlaceholder}}', definition.singularLabel);
}

export function executableCommandHasUnclosedQuote(command: string) {
  return [...command].filter((char) => char === '"').length % 2 === 1;
}

export function normalizeModuleType(value: string) {
  return value.trim().replace(/^["']|["']$/g, '').trim().replace(/^module-/i, '');
}

export function hasDuplicateConnector(rows: ConnectorFormRow[]) {
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

export function paramsFromActionDefaults(action: V10Action): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  for (const field of action.fields) {
    if (field.default !== undefined && field.default !== null) {
      params[field.name] = field.default;
    }
  }
  return params;
}

export function deepClonePipeline(steps: PipelineStep[]): PipelineStep[] {
  return JSON.parse(JSON.stringify(steps)) as PipelineStep[];
}

export function extractImportedActionPlanSteps(payload: ImportableActionPlan): PipelineStep[] {
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

export function isPipelineStep(value: unknown): value is PipelineStep {
  return Boolean(value && typeof value === 'object' && typeof (value as PipelineStep).action === 'string');
}

export function safeFileName(value: string): string {
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

export function prettyJSONForDownload(value: string): string {
  return JSON.stringify(JSON.parse(value), null, 2);
}

export function maquetteJSONFileName(value: string): string {
  const base = value.replace(/(?:\.json)+$/i, '').replace(/[\u0000-\u001F<>:"/\\|?*]/g, '-');
  return `${base || 'maquette'}.json`;
}

export function defaultActionPlanName(maquetteName: string): string {
  const name = maquetteName.trim();
  return name ? formatMessage(m.pipeline.defaultPlanName, { name }) : '';
}

export function parentPath(path: string): string {
  const normalized = path.replace(/[\\/]+$/, '');
  const index = Math.max(normalized.lastIndexOf('/'), normalized.lastIndexOf('\\'));
  return index > 0 ? normalized.slice(0, index) : '';
}

export function formatMessage(template: string, values: Record<string, string>): string {
  return Object.entries(values).reduce((message, [key, value]) => message.split(`{{${key}}}`).join(value), template);
}

export function delay(ms: number) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

