import React, { useEffect, useMemo, useRef, useState } from 'react';
import { PipelineStep, V10Action, V10Config, V10SavedActionPlan, v10LabApi } from '../../../api/v10Lab';
import { Button } from '../../../../../shared/components/ui/Button';
import { messages } from '../../../../../i18n';
import { RequiredDot } from '../../form/RequiredDot';
import {
  actionFieldHidden,
  actionFieldDescription,
  actionFieldLabel,
  actionFieldOptions,
  actionFieldRequired,
  defaultActionPlanName,
  isActionFieldHidden,
  isRecord,
  normalizeActionParamsForSave,
  paramsFromActionDefaults,
  stringValue,
} from '../../../utils/v10LabUtils';

const m = messages.v10Lab;
const systemPipelineActions = new Set(['install-env', 'configure-gedix-cfg', 'start-maquette', 'start-services', 'kill-gx-processes', 'update-env']);
type ComboboxOption = { label: string; value: string };
export function PipelineBuilder({ config, actions, savedActionPlans, selectedSavedActionPlanID, showSaveActionPlan, actionPlanName, onSelectedSavedActionPlanChange, onShowSaveActionPlanChange, onActionPlanNameChange, onSaveActionPlan, onAddSavedActionPlan, onExportActionPlan, onOpenImportActionPlan, onImportActionPlan, onDeleteSavedActionPlan, importInputRef, onChange }: {
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
    const action = byID[step.action];
    const nextStep = action ? { ...step, params: normalizeActionParamsForSave(action, step.params ?? {}) } : step;
    onChange({ ...config, pipeline: apiSteps.map((item, itemIndex) => itemIndex === index ? nextStep : item) });
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
                  {step.action === 'install-env' && <p className="readonly-notice">{m.actionUsesGeneralSettings}</p>}
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

export function ApiTokenEditor({ maquetteName, disabled }: { maquetteName: string; disabled: boolean }) {
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

export function ActionFieldInput({ field, value, config, params, disabledOptionValues, onChange }: { field: V10Action['fields'][number]; value: unknown; config: V10Config; params: Record<string, unknown>; disabledOptionValues?: Set<string>; onChange: (value: unknown) => void }) {
  const label = <FieldLabel field={field} required={actionFieldRequired(field, params)} params={params} />;
  const description = actionFieldDescription(field, params);
  const options = actionFieldOptions(field, config, params);
  if (options.length > 0 || field.optionsSource) {
    const allowCustomValue = field.type === 'string' || field.type === 'text' || field.type === 'number';
    return (
      <label>{label}
        <SearchableSelect
          value={field.type === 'number' && typeof value === 'number' ? String(value) : stringValue(value)}
          options={options}
          disabledOptionValues={disabledOptionValues}
          disabled={options.length === 0 && !allowCustomValue}
          allowCustomValue={allowCustomValue}
          valueType={field.type === 'number' ? 'number' : 'string'}
          emptyLabel={options.length === 0 ? 'Aucune option disponible' : ''}
          onChange={onChange}
        />
        {description && <span className="muted">{description}</span>}
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
    return <label>{label}<textarea value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} />{description && <span className="muted">{description}</span>}</label>;
  }
  if (field.type === 'color') {
    const color = typeof value === 'string' && value.trim() ? value : '#000000';
    return <ActionColorField label={label} value={color} description={description} onChange={onChange} />;
  }
  if (field.type === 'number') {
    return <label>{label}<input type="number" min={field.min} value={typeof value === 'number' ? value : ''} onChange={(event) => onChange(event.currentTarget.value === '' ? '' : Number(event.currentTarget.value))} />{description && <span className="muted">{description}</span>}</label>;
  }
  return <label>{label}<input value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.currentTarget.value)} />{description && <span className="muted">{description}</span>}</label>;
}

export function SearchableSelect({ value, options, disabledOptionValues, disabled = false, allowCustomValue = false, valueType = 'string', emptyLabel = '', onChange }: { value: string; options: ComboboxOption[]; disabledOptionValues?: Set<string>; disabled?: boolean; allowCustomValue?: boolean; valueType?: 'string' | 'number'; emptyLabel?: string; onChange: (value: unknown) => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [inputValue, setInputValue] = useState(displayValueForOption(value, options));
  const [filterValue, setFilterValue] = useState('');
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const optionsKey = options.map((option) => `${option.value}\u0000${option.label}`).join('\u0001');
  const enabledOptions = options.filter((option) => !disabledOptionValues?.has(option.value));
  const query = filterValue.trim().toLowerCase();
  const filteredOptions = enabledOptions.filter((option) => option.label.toLowerCase().includes(query) || option.value.toLowerCase().includes(query));
  const visibleOptions = query === '' ? enabledOptions : filteredOptions;
  const openAndSelectInput = () => {
    setFilterValue('');
    setOpen(true);
    window.requestAnimationFrame(() => inputRef.current?.select());
  };

  useEffect(() => {
    setInputValue(displayValueForOption(value, options));
    setFilterValue('');
  }, [value, optionsKey]);

  useEffect(() => {
    setActiveIndex(0);
  }, [filterValue, optionsKey]);

  const commitCustomValue = (nextInput: string) => {
    if (!allowCustomValue) {
      return;
    }
    if (nextInput === displayValueForOption(value, options)) {
      return;
    }
    if (valueType === 'number') {
      const trimmed = nextInput.trim();
      if (trimmed === '') {
        onChange('');
        return;
      }
      const numberValue = Number(trimmed);
      if (Number.isFinite(numberValue)) {
        onChange(numberValue);
      }
      return;
    }
    onChange(nextInput);
  };
  const selectOption = (option: ComboboxOption) => {
    setInputValue(option.label);
    setFilterValue('');
    setOpen(false);
    if (valueType === 'number') {
      const numberValue = Number(option.value);
      onChange(Number.isFinite(numberValue) ? numberValue : '');
      return;
    }
    onChange(option.value);
  };
  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => Math.min(current + 1, Math.max(visibleOptions.length - 1, 0)));
      return;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => Math.max(current - 1, 0));
      return;
    }
    if (event.key === 'Enter' && open && visibleOptions[activeIndex]) {
      event.preventDefault();
      selectOption(visibleOptions[activeIndex]);
      return;
    }
    if (event.key === 'Enter') {
      commitCustomValue(inputValue);
      setOpen(false);
      return;
    }
    if (event.key === 'Escape') {
      setOpen(false);
    }
  };

  return (
    <div className="v10-combobox">
      <input
        ref={inputRef}
        role="combobox"
        aria-expanded={open}
        aria-autocomplete="list"
        disabled={disabled}
        value={inputValue}
        placeholder={emptyLabel}
        onFocus={() => {
          setFilterValue('');
          setOpen(true);
        }}
        onClick={openAndSelectInput}
        onChange={(event) => {
          const nextInput = event.currentTarget.value;
          setInputValue(nextInput);
          setFilterValue(nextInput);
          setOpen(true);
        }}
        onBlur={() => {
          commitCustomValue(inputValue);
          window.setTimeout(() => setOpen(false), 120);
        }}
        onKeyDown={handleKeyDown}
      />
      {open && !disabled && (
        <div className="v10-combobox-options" role="listbox">
          {visibleOptions.length === 0 && <div className="v10-combobox-empty">{emptyLabel || 'Aucune option'}</div>}
          {visibleOptions.map((option, index) => (
            <button
              type="button"
              key={option.value}
              className={`v10-combobox-option${index === activeIndex ? ' active' : ''}`}
              role="option"
              aria-selected={option.value === value}
              onMouseDown={(event) => {
                event.preventDefault();
                selectOption(option);
              }}
            >
              <span>{option.label}</span>
              <small>{option.value}</small>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function displayValueForOption(value: string, options: ComboboxOption[]): string {
  const selected = options.find((option) => option.value === value);
  return selected?.label ?? value;
}

export function ActionNumberArrayField({ field, value, onChange }: { field: V10Action['fields'][number]; value: unknown; onChange: (value: unknown) => void }) {
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

export function ActionObjectArrayField({ field, value, config, params, onChange }: { field: V10Action['fields'][number]; value: unknown; config: V10Config; params: Record<string, unknown>; onChange: (value: unknown) => void }) {
  const rows = Array.isArray(value) ? value.filter(isRecord) : [];
  const itemFields = field.itemFields ?? [];
  const uniqueFieldName = field.uniqueItemField;
  const uniqueField = uniqueFieldName ? itemFields.find((itemField) => itemField.name === uniqueFieldName) : undefined;
  const uniqueOptions = uniqueField ? actionFieldOptions(uniqueField, config, params) : [];
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

export function ActionColorField({ label, value, description, onChange }: { label: React.ReactNode; value: string; description: string; onChange: (value: unknown) => void }) {
  const normalized = /^#[0-9a-fA-F]{6}$/.test(value) ? value : '#000000';
  return (
    <label>{label}
      <div className="v10-color-field">
        <input type="color" value={normalized} onChange={(event) => onChange(event.currentTarget.value.toUpperCase())} />
        <input value={value} placeholder="#D0021B" onChange={(event) => onChange(event.currentTarget.value)} />
      </div>
      {description && <span className="muted">{description}</span>}
    </label>
  );
}

export function FieldLabel({ field, required = field.required, params = {} }: { field: V10Action['fields'][number]; required?: boolean; params?: Record<string, unknown> }) {
  return (
    <span className="v10-field-label">
      {actionFieldLabel(field, params)}
      {required && <RequiredDot />}
    </span>
  );
}


export const ActionPlanTab = PipelineBuilder;

