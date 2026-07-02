import React, { useEffect, useState } from 'react';
import { ExecutableCommandTargetKind, ExecutionResponse, LogSummary, V10Config, V10Product } from '../../../api/v10Lab';
import { Button } from '../../../../../shared/components/ui/Button';
import { messages } from '../../../../../i18n';
import { DebugTargetsEditor } from './GeneralTab';
import {
  executableCommandHasUnclosedQuote,
  executableCommandGroups,
  executableCommandOptionValue,
  formatDate,
  type RunState,
} from '../../../utils/v10LabUtils';
const m = messages.v10Lab;
export type ExecutableCommandOption = {
  kind: ExecutableCommandTargetKind;
  name: string;
  label: string;
};

export type ExecutableCommandGroup = {
  label: string;
  options: ExecutableCommandOption[];
};

export function ModuleCommandPanel({ config, product, disabled, onRun, showTitle = true }: { config: V10Config; product: V10Product; disabled: boolean; onRun: (targetKind: ExecutableCommandTargetKind, targetName: string, command: string) => void; showTitle?: boolean }) {
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

export function ExecutableCommandOptions({ groups, excludedNames = [], valueFor = (option: ExecutableCommandOption) => option.name }: { groups: ExecutableCommandGroup[]; excludedNames?: string[]; valueFor?: (option: ExecutableCommandOption) => string }) {
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

export function ExecutionPanel({ config, product, busy, runState, execution, logs, selectedLog, onConfigChange, onCreate, onUpdate, onConfigure, onStart, onOpenMaquette, onRunPipeline, onRunExecutableCommand, onKill, onRefreshLogs, onReadLog }: {
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
          <Button type="button" onClick={onCreate} disabled={disabled}>{m.execution.installMaquette}</Button>
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

export const ExecutionLogsTab = ExecutionPanel;


