import React, { useEffect, useState } from 'react';
import { RunSheetInput, RunStepInput, TestRunSheet, TestRunStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { StatusBadge } from './StatusBadge';

type Props = {
  sheet: TestRunSheet;
  current?: boolean;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onSaveStep: (stepId: number, input: RunStepInput) => Promise<void>;
};

const statuses: TestRunStep['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetCard({ sheet, current = false, onSave, onSaveStep }: Props) {
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
        <dt>Prerequis</dt>
        <dd>{sheet.prerequisites || '-'}</dd>
        <dt>Configuration</dt>
        <dd>{sheet.config || '-'}</dd>
        <dt>Commande</dt>
        <dd>{sheet.command || '-'}</dd>
      </dl>
      {sheet.steps && sheet.steps.length > 0 ? (
        <div className="run-step-list">
          {sheet.steps.map((step) => (
            <RunStepEditor key={step.id} step={step} onSave={onSaveStep} />
          ))}
        </div>
      ) : (
        <>
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
          <Button type="button" disabled={saving} onClick={() => save()}>
            {saving ? 'Sauvegarde...' : 'Sauvegarder'}
          </Button>
        </>
      )}
    </Card>
  );
}

function RunStepEditor({ step, onSave }: { step: TestRunStep; onSave: (stepId: number, input: RunStepInput) => Promise<void> }) {
  const [value, setValue] = useState<RunStepInput>({
    status: step.status,
    actualResult: step.actualResult,
    comment: step.comment,
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue({
      status: step.status,
      actualResult: step.actualResult,
      comment: step.comment,
    });
  }, [step]);

  const save = async (input: RunStepInput = value) => {
    setSaving(true);
    await onSave(step.id, input);
    setSaving(false);
  };

  return (
    <div className="run-step-editor">
      <div className="run-step-header">
        <div>
          <span className="section-kicker">Etape {step.executionOrder}</span>
          <h4>{step.field || 'Action de test'}</h4>
        </div>
        <StatusBadge status={value.status} />
      </div>
      <dl className="compact-definition-list">
        <dt>Action</dt>
        <dd>{step.action || '-'}</dd>
        <dt>Attendu</dt>
        <dd>{step.expectedResult || '-'}</dd>
      </dl>
      <div className="status-action-grid" aria-label="Changer le statut de l etape">
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
      <Button type="button" disabled={saving} onClick={() => save()}>
        {saving ? 'Sauvegarde...' : 'Sauvegarder l etape'}
      </Button>
    </div>
  );
}
