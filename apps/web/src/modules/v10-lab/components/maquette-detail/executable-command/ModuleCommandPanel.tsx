import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ExecutableCommandHistoryEntry, ExecutableCommandTargetKind, V10Config, V10Product, v10LabApi } from '../../../api/v10Lab';
import { Button } from '../../../../../shared/components/ui/Button';
import { messages } from '../../../../../i18n';
import {
  executableCommandHasUnclosedQuote,
  executableCommandGroups,
  executableCommandOptionValue,
} from '../../../utils/v10LabUtils';
import { ExecutableCommandOptions } from './ExecutableCommandOptions';
import { CommandHistoryList } from './CommandHistoryList';

const m = messages.v10Lab;

export function ModuleCommandPanel({ config, product, disabled, onRun, showTitle = true }: {
  config: V10Config;
  product: V10Product;
  disabled: boolean;
  onRun: (targetKind: ExecutableCommandTargetKind, targetName: string, command: string) => Promise<void> | void;
  showTitle?: boolean;
}) {
  const groups = useMemo(() => executableCommandGroups(config, product), [config, product]);
  const options = useMemo(() => groups.flatMap((group) => group.options), [groups]);
  const optionValues = useMemo(() => options.map(executableCommandOptionValue).join('|'), [options]);
  const [selectedValue, setSelectedValue] = useState(options[0] ? executableCommandOptionValue(options[0]) : '');
  const [command, setCommand] = useState('');
  const [missingTarget, setMissingTarget] = useState('');
  const [history, setHistory] = useState<ExecutableCommandHistoryEntry[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState('');
  const currentMaquetteName = useRef(config.name);
  const invalid = executableCommandHasUnclosedQuote(command);
  const selectedOption = options.find((option) => executableCommandOptionValue(option) === selectedValue);

  useEffect(() => {
    currentMaquetteName.current = config.name;
    setHistory([]);
    setHistoryError('');
    void loadHistory(config.name);
  }, [config.name]);

  useEffect(() => {
    if (!selectedValue || !options.some((option) => executableCommandOptionValue(option) === selectedValue)) {
      setSelectedValue(options[0] ? executableCommandOptionValue(options[0]) : '');
    }
  }, [optionValues, selectedValue, options]);

  async function loadHistory(name = config.name) {
    setHistoryLoading(true);
    setHistoryError('');
    try {
      const items = await v10LabApi.listExecutableCommandHistory(name);
      if (currentMaquetteName.current === name) {
        setHistory(items);
      }
    } catch (err) {
      if (currentMaquetteName.current === name) {
        setHistoryError(err instanceof Error ? err.message : m.moduleCommand.history.loadError);
      }
    } finally {
      if (currentMaquetteName.current === name) {
        setHistoryLoading(false);
      }
    }
  }

  async function runCommand(targetKind: ExecutableCommandTargetKind, targetName: string, value: string) {
    await onRun(targetKind, targetName, value);
    await loadHistory();
  }

  function reuse(entry: ExecutableCommandHistoryEntry) {
    const option = findOption(entry);
    setCommand(entry.command);
    if (option) {
      setSelectedValue(executableCommandOptionValue(option));
      setMissingTarget('');
      return;
    }
    setMissingTarget(m.moduleCommand.history.missingTarget);
  }

  async function rerun(entry: ExecutableCommandHistoryEntry) {
    if (!findOption(entry)) {
      setCommand(entry.command);
      setMissingTarget(m.moduleCommand.history.missingTarget);
      return;
    }
    await runCommand(entry.targetKind, entry.targetName, entry.command);
  }

  async function toggleFavorite(entry: ExecutableCommandHistoryEntry) {
    try {
      setHistory(await v10LabApi.setExecutableCommandHistoryFavorite(config.name, entry.id, !entry.favorite));
    } catch (err) {
      setHistoryError(err instanceof Error ? err.message : m.moduleCommand.history.saveError);
    }
  }

  async function deleteEntry(entry: ExecutableCommandHistoryEntry) {
    try {
      setHistory(await v10LabApi.deleteExecutableCommandHistoryEntry(config.name, entry.id));
    } catch (err) {
      setHistoryError(err instanceof Error ? err.message : m.moduleCommand.history.saveError);
    }
  }

  async function clearNonFavorites() {
    if (!window.confirm(m.moduleCommand.history.clearConfirm)) {
      return;
    }
    try {
      setHistory(await v10LabApi.clearExecutableCommandHistoryNonFavorites(config.name));
    } catch (err) {
      setHistoryError(err instanceof Error ? err.message : m.moduleCommand.history.saveError);
    }
  }

  function findOption(entry: ExecutableCommandHistoryEntry) {
    return options.find((option) => option.kind === entry.targetKind && option.name === entry.targetName);
  }

  return (
    <div className="v10-module-command">
      {showTitle && <h4>{m.moduleCommand.title}</h4>}
      <p className="muted">{m.moduleCommand.help}</p>
      <div className="form-grid v10-form-grid">
        <label>{m.moduleCommand.target}
          <select value={selectedValue} onChange={(event) => { setSelectedValue(event.currentTarget.value); setMissingTarget(''); }}>
            <ExecutableCommandOptions groups={groups} valueFor={executableCommandOptionValue} />
          </select>
        </label>
        <label>{m.moduleCommand.command}
          <input value={command} placeholder={m.moduleCommand.commandPlaceholder} onChange={(event) => setCommand(event.currentTarget.value)} />
        </label>
      </div>
      {invalid && <p className="error">{m.moduleCommand.unclosedQuote}</p>}
      {missingTarget && <p className="warning-message">{missingTarget}</p>}
      <div className="button-row">
        <Button type="button" variant="secondary" disabled={disabled || !selectedOption || !command.trim() || invalid} onClick={() => selectedOption && void runCommand(selectedOption.kind, selectedOption.name, command)}>
          {m.moduleCommand.run}
        </Button>
      </div>
      <CommandHistoryList
        entries={history}
        loading={historyLoading}
        error={historyError}
        disabled={disabled}
        targetExists={(entry) => Boolean(findOption(entry))}
        onReload={() => void loadHistory()}
        onReuse={reuse}
        onRerun={(entry) => void rerun(entry)}
        onToggleFavorite={(entry) => void toggleFavorite(entry)}
        onDelete={(entry) => void deleteEntry(entry)}
        onClearNonFavorites={() => void clearNonFavorites()}
      />
    </div>
  );
}
