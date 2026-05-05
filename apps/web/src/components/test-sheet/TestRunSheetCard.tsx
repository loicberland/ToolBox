import React, { useEffect, useState } from 'react';
import { RunSheetInput, TestRunSheet } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { StatusBadge } from './StatusBadge';

type Props = {
  sheet: TestRunSheet;
  current?: boolean;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
};

const statuses: TestRunSheet['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetCard({ sheet, current = false, onSave }: Props) {
  const [value, setValue] = useState<RunSheetInput>({
    status: sheet.status,
    actualResult: sheet.actualResult,
    comment: sheet.comment,
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue({
      status: sheet.status,
      actualResult: sheet.actualResult,
      comment: sheet.comment,
    });
  }, [sheet]);

  const save = async (input: RunSheetInput = value) => {
    setSaving(true);
    await onSave(sheet.id, input);
    setSaving(false);
  };

  return (
    <Card className={`run-card ${current ? 'current' : ''}`}>
      <header className="run-card-header">
        <div>
          <div className="card-topline">
            <StatusBadge status={value.status} />
            {current && <span className="current-marker">Test courant</span>}
          </div>
          <h3>{sheet.executionOrder}. {sheet.name}</h3>
        </div>
      </header>
      <dl>
        <dt>Action</dt>
        <dd>{sheet.action || '-'}</dd>
        <dt>Resultat attendu</dt>
        <dd>{sheet.expectedResult || '-'}</dd>
      </dl>
      <label>
        Statut
        <select value={value.status} onChange={(event) => setValue({ ...value, status: event.target.value as TestRunSheet['status'] })}>
          {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
        </select>
      </label>
      <div className="status-action-grid" aria-label="Changer le statut">
        <Button type="button" variant="success" size="sm" onClick={() => save({ ...value, status: 'passed' })}>Reussi</Button>
        <Button type="button" variant="danger" size="sm" onClick={() => save({ ...value, status: 'failed' })}>Echoue</Button>
        <Button type="button" variant="warning" size="sm" onClick={() => save({ ...value, status: 'blocked' })}>Bloque</Button>
        <Button type="button" variant="secondary" size="sm" onClick={() => save({ ...value, status: 'skipped' })}>Ignore</Button>
      </div>
      <label>
        Resultat obtenu
        <textarea value={value.actualResult} onChange={(event) => setValue({ ...value, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea value={value.comment} onChange={(event) => setValue({ ...value, comment: event.target.value })} />
      </label>
      <Button
        type="button"
        disabled={saving}
        onClick={() => save()}
      >
        {saving ? 'Sauvegarde...' : 'Sauvegarder'}
      </Button>
    </Card>
  );
}
