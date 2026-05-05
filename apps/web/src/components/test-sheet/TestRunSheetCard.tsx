import React, { useState } from 'react';
import { RunSheetInput, TestRunSheet } from '../../api/testSheet';

type Props = {
  sheet: TestRunSheet;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
};

const statuses: TestRunSheet['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetCard({ sheet, onSave }: Props) {
  const [value, setValue] = useState<RunSheetInput>({
    status: sheet.status,
    actualResult: sheet.actualResult,
    comment: sheet.comment,
  });
  const [saving, setSaving] = useState(false);

  return (
    <article className="run-card">
      <header>
        <span className={`status-pill ${value.status}`}>{value.status}</span>
        <h3>{sheet.executionOrder}. {sheet.name}</h3>
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
      <label>
        Resultat reel
        <textarea value={value.actualResult} onChange={(event) => setValue({ ...value, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea value={value.comment} onChange={(event) => setValue({ ...value, comment: event.target.value })} />
      </label>
      <button
        type="button"
        disabled={saving}
        onClick={async () => {
          setSaving(true);
          await onSave(sheet.id, value);
          setSaving(false);
        }}
      >
        {saving ? 'Sauvegarde...' : 'Sauvegarder'}
      </button>
    </article>
  );
}
