import React from 'react';
import { ExecutableCommandHistoryEntry } from '../../../api/v10Lab';
import { Button } from '../../../../../shared/components/ui/Button';
import { messages } from '../../../../../i18n';
import { formatDate } from '../../../utils/v10LabUtils';

const m = messages.v10Lab.moduleCommand.history;

export function CommandHistoryItem({ entry, disabled, missingTarget, onReuse, onRerun, onToggleFavorite, onDelete }: {
  entry: ExecutableCommandHistoryEntry;
  disabled: boolean;
  missingTarget: boolean;
  onReuse: (entry: ExecutableCommandHistoryEntry) => void;
  onRerun: (entry: ExecutableCommandHistoryEntry) => void;
  onToggleFavorite: (entry: ExecutableCommandHistoryEntry) => void;
  onDelete: (entry: ExecutableCommandHistoryEntry) => void;
}) {
  return (
    <li className={missingTarget ? 'v10-command-history-item missing-target' : 'v10-command-history-item'}>
      <div className="v10-command-history-main">
        <div className="v10-command-history-meta">
          <strong>{targetLabel(entry)}</strong>
          <span>{formatDate(entry.lastExecutedAt)}</span>
          {entry.executionCount > 1 && <span>{m.executionCount.replace('{{count}}', String(entry.executionCount))}</span>}
          {missingTarget && <span className="warning-message">{m.missingTarget}</span>}
        </div>
        <code>{entry.command}</code>
      </div>
      <div className="button-row v10-command-history-actions">
        <Button type="button" size="sm" variant="secondary" onClick={() => onReuse(entry)}>{m.reuse}</Button>
        <Button type="button" size="sm" variant="secondary" disabled={disabled || missingTarget} onClick={() => onRerun(entry)}>{m.rerun}</Button>
        <button
          type="button"
          className={entry.favorite ? 'v10-icon-button active' : 'v10-icon-button'}
          title={entry.favorite ? m.removeFavorite : m.addFavorite}
          aria-label={entry.favorite ? m.removeFavorite : m.addFavorite}
          disabled={disabled}
          onClick={() => onToggleFavorite(entry)}
        >
          {entry.favorite ? '★' : '☆'}
        </button>
        <button
          type="button"
          className="v10-icon-button danger"
          title={m.delete}
          aria-label={m.delete}
          disabled={disabled}
          onClick={() => onDelete(entry)}
        >
          x
        </button>
      </div>
    </li>
  );
}

function targetLabel(entry: ExecutableCommandHistoryEntry) {
  return `${entry.targetKind} - ${entry.targetName}`;
}
