import React, { useEffect, useState } from 'react';
import { RunSheetInput, RunStepInput, TestRunSheet, TestRunStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { MarkdownCollapsibleSection } from '../ui/MarkdownCollapsibleSection';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';
import { SmartEllipsisText } from '../ui/SmartEllipsisText';
import { StatusBadge } from './StatusBadge';
import { TestRunStepProgress } from './TestRunStepProgress';
import { getRunSheetProgress } from './runStatus';

type Props = {
  sheet: TestRunSheet;
  onSaveSheet: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onSaveStep: (stepId: number, input: RunStepInput) => Promise<void>;
};

const statuses: TestRunStep['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetDetail({ sheet, onSaveSheet, onSaveStep }: Props) {
  const [openedStepId, setOpenedStepId] = useState<number | undefined>();
  const progress = getRunSheetProgress(sheet);
  const openedStep = (sheet.steps ?? []).find((step) => step.id === openedStepId);

  const toggleStepWithKeyboard = (event: React.KeyboardEvent<HTMLDivElement>, stepId: number) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      setOpenedStepId(stepId === openedStepId ? undefined : stepId);
    }
  };

  useEffect(() => {
    setOpenedStepId(undefined);
  }, [sheet.id]);

  return (
    <Card className="run-sheet-detail">
      <header className="run-card-header">
        <div>
          <div className="card-topline">
            <StatusBadge status={progress.status} />
            <span className="current-marker">{progress.total} action{progress.total > 1 ? 's' : ''}</span>
          </div>
          <h3>{sheet.executionOrder}. {sheet.name}</h3>
        </div>
      </header>

      <TestRunStepProgress steps={sheet.steps ?? []} />

      <RunSheetReadDetails sheet={sheet} />

      {sheet.steps && sheet.steps.length > 0 ? (
        <div className="run-action-list">
          {sheet.steps.map((step) => (
            <div
              className={`run-action-list-item ${step.id === openedStepId ? 'active' : ''}`}
              key={step.id}
              role="button"
              tabIndex={0}
              onClick={() => setOpenedStepId(step.id === openedStepId ? undefined : step.id)}
              onKeyDown={(event) => toggleStepWithKeyboard(event, step.id)}
            >
              <span className="run-list-order">{step.executionOrder}</span>
              <div className="run-action-title">
                <SmartEllipsisText text={hasMarkdownContent(step.action) ? step.action : 'Action non renseignee'} />
              </div>
              <StatusBadge status={step.status} />
            </div>
          ))}
        </div>
      ) : (
        <RunSheetResultEditor sheet={sheet} onSave={onSaveSheet} />
      )}

      {openedStep && (
        <TestRunStepDetail
          step={openedStep}
          onSave={onSaveStep}
          onSaved={() => setOpenedStepId(undefined)}
        />
      )}
    </Card>
  );
}

function RunSheetReadDetails({ sheet }: { sheet: TestRunSheet }) {
  const details = [
    ['Description', sheet.description],
    ['Prerequis', sheet.prerequisites],
    ['Configuration', sheet.config],
    ['Commande', sheet.command],
    ['Notes', sheet.notes],
  ] as const;

  return (
    <div className="run-read-details">
      {details.filter(([, content]) => hasMarkdownContent(content)).map(([label, content]) => (
        <MarkdownCollapsibleSection key={label} title={label} content={content} defaultOpen />
      ))}
    </div>
  );
}

function TestRunStepDetail({ step, onSave, onSaved }: { step: TestRunStep; onSave: (stepId: number, input: RunStepInput) => Promise<void>; onSaved: () => void }) {
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
    try {
      await onSave(step.id, input);
    } catch (error) {
      setSaving(false);
      throw error;
    }
    setSaving(false);
    onSaved();
  };
  const hasReadDetails = hasMarkdownContent(step.field) || hasMarkdownContent(step.expectedResult);

  return (
    <div className="run-step-detail">
      <div className="run-step-header">
        <div>
          <span className="section-kicker">Action {step.executionOrder}</span>
          {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} /> : <h4>Etape sans action</h4>}
        </div>
        <StatusBadge status={value.status} />
      </div>
      {hasReadDetails && (
        <dl className="compact-definition-list">
          {hasMarkdownContent(step.field) && (
            <>
              <dt>Specifique</dt>
              <dd><MarkdownPreview content={step.field} compact /></dd>
            </>
          )}
          {hasMarkdownContent(step.expectedResult) && (
            <>
              <dt>Attendu</dt>
              <dd><MarkdownPreview content={step.expectedResult} compact /></dd>
            </>
          )}
        </dl>
      )}
      <label>
        Resultat obtenu
        <textarea value={value.actualResult} onChange={(event) => setValue({ ...value, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea value={value.comment} onChange={(event) => setValue({ ...value, comment: event.target.value })} />
      </label>
      <div className="status-action-grid" aria-label="Changer le statut de l action">
        <Button type="button" variant="success" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'passed' })}>Reussi</Button>
        <Button type="button" variant="danger" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'failed' })}>Echoue</Button>
        <Button type="button" variant="warning" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'blocked' })}>Bloque</Button>
        <Button type="button" variant="secondary" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'skipped' })}>Ignore</Button>
      </div>
      <Button type="button" disabled={saving} onClick={() => save()}>
        {saving ? 'Sauvegarde...' : 'Sauvegarder'}
      </Button>
    </div>
  );
}

function RunSheetResultEditor({ sheet, onSave }: { sheet: TestRunSheet; onSave: (sheetId: number, input: RunSheetInput) => Promise<void> }) {
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
    try {
      await onSave(sheet.id, input);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="run-step-detail">
      <label>
        Statut
        <select value={value.status} onChange={(event) => setValue({ ...value, status: event.target.value as TestRunSheet['status'] })}>
          {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
        </select>
      </label>
      <label>
        Resultat obtenu
        <textarea value={value.actualResult} onChange={(event) => setValue({ ...value, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea value={value.comment} onChange={(event) => setValue({ ...value, comment: event.target.value })} />
      </label>
      <div className="status-action-grid" aria-label="Changer le statut du test">
        <Button type="button" variant="success" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'passed' })}>Reussi</Button>
        <Button type="button" variant="danger" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'failed' })}>Echoue</Button>
        <Button type="button" variant="warning" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'blocked' })}>Bloque</Button>
        <Button type="button" variant="secondary" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'skipped' })}>Ignore</Button>
      </div>
      <Button type="button" disabled={saving} onClick={() => save()}>
        {saving ? 'Sauvegarde...' : 'Sauvegarder'}
      </Button>
    </div>
  );
}
