import React from 'react';
import { ExecutableCommandHistoryEntry } from '../../../api/v10Lab';
import { Button } from '../../../../../shared/components/ui/Button';
import { messages } from '../../../../../i18n';
import { CommandHistoryItem } from './CommandHistoryItem';

const m = messages.v10Lab.moduleCommand.history;

export function CommandHistoryList({ entries, loading, error, disabled, targetExists, onReload, onReuse, onRerun, onToggleFavorite, onDelete, onClearNonFavorites }: {
  entries: ExecutableCommandHistoryEntry[];
  loading: boolean;
  error: string;
  disabled: boolean;
  targetExists: (entry: ExecutableCommandHistoryEntry) => boolean;
  onReload: () => void;
  onReuse: (entry: ExecutableCommandHistoryEntry) => void;
  onRerun: (entry: ExecutableCommandHistoryEntry) => void;
  onToggleFavorite: (entry: ExecutableCommandHistoryEntry) => void;
  onDelete: (entry: ExecutableCommandHistoryEntry) => void;
  onClearNonFavorites: () => void;
}) {
  const favorites = entries.filter((entry) => entry.favorite);
  const recent = entries.filter((entry) => !entry.favorite);
  const hasNonFavorites = recent.length > 0;
  return (
    <div className="v10-command-history">
      <div className="v10-command-history-header">
        <h5>{m.title}</h5>
        <div className="button-row">
          <Button type="button" size="sm" variant="secondary" disabled={disabled || loading || !hasNonFavorites} onClick={onClearNonFavorites}>{m.clear}</Button>
          <Button type="button" size="sm" variant="secondary" disabled={disabled || loading} onClick={onReload}>{m.reload}</Button>
        </div>
      </div>
      {loading && <p className="muted">{m.loading}</p>}
      {error && <p className="error">{error}</p>}
      {!loading && !error && entries.length === 0 && <p className="muted">{m.empty}</p>}
      {!loading && !error && favorites.length > 0 && (
        <CommandHistorySection
          title={m.favorites}
          entries={favorites}
          disabled={disabled}
          targetExists={targetExists}
          onReuse={onReuse}
          onRerun={onRerun}
          onToggleFavorite={onToggleFavorite}
          onDelete={onDelete}
        />
      )}
      {!loading && !error && (
        <CommandHistorySection
          title={m.recent}
          entries={recent}
          disabled={disabled}
          targetExists={targetExists}
          emptyText={entries.length ? m.emptyRecent : undefined}
          onReuse={onReuse}
          onRerun={onRerun}
          onToggleFavorite={onToggleFavorite}
          onDelete={onDelete}
        />
      )}
    </div>
  );
}

function CommandHistorySection({ title, entries, disabled, targetExists, emptyText, onReuse, onRerun, onToggleFavorite, onDelete }: {
  title: string;
  entries: ExecutableCommandHistoryEntry[];
  disabled: boolean;
  targetExists: (entry: ExecutableCommandHistoryEntry) => boolean;
  emptyText?: string;
  onReuse: (entry: ExecutableCommandHistoryEntry) => void;
  onRerun: (entry: ExecutableCommandHistoryEntry) => void;
  onToggleFavorite: (entry: ExecutableCommandHistoryEntry) => void;
  onDelete: (entry: ExecutableCommandHistoryEntry) => void;
}) {
  return (
    <section className="v10-command-history-section">
      <h6>{title}</h6>
      {entries.length === 0 ? (
        emptyText ? <p className="muted">{emptyText}</p> : null
      ) : (
        <ul>
          {entries.map((entry) => (
            <CommandHistoryItem
              key={entry.id}
              entry={entry}
              disabled={disabled}
              missingTarget={!targetExists(entry)}
              onReuse={onReuse}
              onRerun={onRerun}
              onToggleFavorite={onToggleFavorite}
              onDelete={onDelete}
            />
          ))}
        </ul>
      )}
    </section>
  );
}
