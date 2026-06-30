import React, { useEffect, useState } from 'react';
import { MaquetteGroup, V10Config, V10Product } from '../../../api/v10Lab';
import { messages } from '../../../../../i18n';
import { Button } from '../../../../../shared/components/ui/Button';
import { RequiredDot } from '../../form/RequiredDot';
import { GedixForm } from './GedixConfigTab';
import {
  customArgumentsForTarget,
  executableCommandGroups,
  executableCommandOptionValue,
  materializeProductServices,
  productFor,
} from '../../../utils/v10LabUtils';

const m = messages.v10Lab;

function ExecutableCommandOptions({ groups, excludedNames = [], valueFor = executableCommandOptionValue }: { groups: ReturnType<typeof executableCommandGroups>; excludedNames?: string[]; valueFor?: (option: ReturnType<typeof executableCommandGroups>[number]['options'][number]) => string }) {
  const excluded = new Set(excludedNames);
  return <>
    {groups.map((group) => {
      const options = group.options.filter((option) => !excluded.has(option.name));
      if (options.length === 0) return null;
      return <optgroup key={group.label} label={group.label}>{options.map((option) => <option key={`${option.kind}:${option.name}`} value={valueFor(option)}>{option.label}</option>)}</optgroup>;
    })}
  </>;
}
export function MaquetteGeneralForm({ config, products, groups, defaultTargetPath, onChange, onSelectZip, creating = false }: {
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

export function DebugTargetsEditor({ config, product, onChange }: { config: V10Config; product: V10Product; onChange: (config: V10Config) => void }) {
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

export const GeneralTab = MaquetteGeneralForm;


