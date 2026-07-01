import React from 'react';
import { MaquetteSummary } from '../api/v10Lab';
import { Button } from '../../../shared/components/ui/Button';
import { messages } from '../../../i18n';

const m = messages.v10Lab;
function MaquetteList({ items, selectedName, onToggle, onDuplicate }: { items: MaquetteSummary[]; selectedName: string; onToggle: (name: string) => Promise<void>; onDuplicate: (item: MaquetteSummary) => void }) {
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
            <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onDuplicate(item); }}>{m.duplicate}</Button>
          </div>
        </div>
      ))}
    </div>
  );
}
function handleToggleKey(event: React.KeyboardEvent, action: () => void) {
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }
  event.preventDefault();
  action();
}

export function MaquetteListView({
  maquettes,
  groups,
  groupedMaquettes,
  ungroupedMaquettes,
  selectedName,
  busy,
  openGroups,
  openUngrouped,
  newGroupName,
  onNewGroupNameChange,
  onReload,
  onCreateGroup,
  onToggleGroup,
  onToggleUngrouped,
  onToggleMaquette,
  onDuplicate,
  onAddToGroup,
  onAddUngrouped,
  onDeleteGroup,
}: {
  maquettes: MaquetteSummary[];
  groups: Array<{ name: string }>;
  groupedMaquettes: Array<{ name: string; items: MaquetteSummary[] }>;
  ungroupedMaquettes: MaquetteSummary[];
  selectedName: string;
  busy: boolean;
  openGroups: Record<string, boolean>;
  openUngrouped: boolean;
  newGroupName: string;
  onNewGroupNameChange: (value: string) => void;
  onReload: () => void;
  onCreateGroup: () => void;
  onToggleGroup: (name: string) => void;
  onToggleUngrouped: () => void;
  onToggleMaquette: (name: string) => Promise<void>;
  onDuplicate: (item: MaquetteSummary) => void;
  onAddToGroup: (groupName: string) => void;
  onAddUngrouped: () => void;
  onDeleteGroup: (groupName: string) => void;
}) {
  return (
    <section className="ui-card v10-section">
      <div className="ui-card-header">
        <h3>{m.registeredMaquettes}</h3>
        <Button type="button" variant="secondary" size="sm" onClick={onReload} disabled={busy}>{m.refreshLogs}</Button>
      </div>
      <div className="v10-file-row">
        <input placeholder="Nom du groupe" value={newGroupName} onChange={(event) => onNewGroupNameChange(event.currentTarget.value)} />
        <Button type="button" variant="secondary" size="sm" onClick={onCreateGroup} disabled={busy || !newGroupName.trim()}>Créer un groupe</Button>
      </div>
      {maquettes.length === 0 && groups.length === 0 ? (
        <div className="empty-state"><h3>{m.noMaquette}</h3></div>
      ) : (
        <div className="v10-group-list">
          {groupedMaquettes.map((group) => (
            <div className="v10-group" key={group.name}>
              <div className="v10-group-header clickable" role="button" tabIndex={0} onClick={() => onToggleGroup(group.name)} onKeyDown={(event) => handleToggleKey(event, () => onToggleGroup(group.name))}>
                <span className="v10-chevron" aria-hidden="true">{openGroups[group.name] ? '▾' : '▸'}</span>
                <strong>{group.name}</strong>
                <span className="muted">{group.items.length}</span>
                <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onAddToGroup(group.name); }}>Ajouter une maquette</Button>
                <Button type="button" size="sm" variant="danger" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onDeleteGroup(group.name); }} disabled={group.items.length > 0}>Supprimer</Button>
              </div>
              {openGroups[group.name] && <MaquetteList items={group.items} selectedName={selectedName} onToggle={onToggleMaquette} onDuplicate={onDuplicate} />}
            </div>
          ))}
          <div className="v10-group">
            <div className="v10-group-header clickable" role="button" tabIndex={0} onClick={onToggleUngrouped} onKeyDown={(event) => handleToggleKey(event, onToggleUngrouped)}>
              <span className="v10-chevron" aria-hidden="true">{openUngrouped ? '▾' : '▸'}</span>
              <strong>Sans groupe</strong>
              <span className="muted">{ungroupedMaquettes.length}</span>
              <Button type="button" size="sm" variant="secondary" onKeyDown={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onAddUngrouped(); }}>Ajouter une maquette</Button>
            </div>
            {openUngrouped && <MaquetteList items={ungroupedMaquettes} selectedName={selectedName} onToggle={onToggleMaquette} onDuplicate={onDuplicate} />}
          </div>
        </div>
      )}
    </section>
  );
}

