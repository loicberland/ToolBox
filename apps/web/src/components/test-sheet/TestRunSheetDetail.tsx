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
  readOnly?: boolean;
  onSaveSheet: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onSaveStep: (stepId: number, input: RunStepInput) => Promise<void>;
};

const statuses: TestRunStep['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetDetail({ sheet, readOnly = false, onSaveSheet, onSaveStep }: Props) {
  const [openedStepId, setOpenedStepId] = useState<number | undefined>();
  const [stepDrafts, setStepDrafts] = useState<Record<number, RunStepInput>>({});
  const progress = getRunSheetProgress(sheet);

  const getStepDraft = (step: TestRunStep): RunStepInput => stepDrafts[step.id] ?? {
    status: step.status,
    actualResult: step.actualResult,
    comment: step.comment,
  };

  const updateStepDraft = (stepId: number, draft: RunStepInput) => {
    setStepDrafts((current) => ({ ...current, [stepId]: draft }));
  };

  const removeStepDraft = (stepId: number) => {
    setStepDrafts((current) => {
      const next = { ...current };
      delete next[stepId];
      return next;
    });
  };

  const toggleStep = async (step: TestRunStep) => {
    if (openedStepId === step.id) {
      if (!readOnly) {
        await onSaveStep(step.id, getStepDraft(step));
        removeStepDraft(step.id);
      }
      setOpenedStepId(undefined);
      return;
    }
    if (openedStepId) {
      const currentStep = (sheet.steps ?? []).find((item) => item.id === openedStepId);
      if (currentStep && !readOnly) {
        await onSaveStep(currentStep.id, getStepDraft(currentStep));
        removeStepDraft(currentStep.id);
      }
    }
    updateStepDraft(step.id, getStepDraft(step));
    setOpenedStepId(step.id);
  };

  const toggleStepWithKeyboard = (event: React.KeyboardEvent<HTMLDivElement>, step: TestRunStep) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      void toggleStep(step);
    }
  };

  useEffect(() => {
    setOpenedStepId(undefined);
    setStepDrafts({});
  }, [sheet.id]);

  useEffect(() => {
    setStepDrafts((current) => {
      const next: Record<number, RunStepInput> = {};
      for (const step of sheet.steps ?? []) {
        if (current[step.id]) {
          next[step.id] = current[step.id];
        }
      }
      return next;
    });
  }, [sheet.steps]);

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

      {readOnly && <p className="readonly-notice">Execution en lecture seule</p>}

      <RunSheetReadDetails sheet={sheet} />

      {sheet.steps && sheet.steps.length > 0 ? (
        <div className="run-action-list">
          {sheet.steps.map((step) => (
            <React.Fragment key={step.id}>
              <div
                className={`run-action-list-item ${step.id === openedStepId ? 'active' : ''}`}
                role="button"
                tabIndex={0}
                onClick={() => { void toggleStep(step); }}
                onKeyDown={(event) => toggleStepWithKeyboard(event, step)}
              >
                <span className="run-list-order">{step.executionOrder}</span>
                <div className="run-action-title">
                  <SmartEllipsisText text={hasMarkdownContent(step.action) ? step.action : 'Action non renseignee'} />
                </div>
                <StatusBadge status={getStepDraft(step).status} />
              </div>
              {step.id === openedStepId && (
                <TestRunStepDetail
                  draft={getStepDraft(step)}
                  step={step}
                  onDraftChange={(draft) => updateStepDraft(step.id, draft)}
                  readOnly={readOnly}
                  onSave={async (input) => {
                    if (readOnly) {
                      return;
                    }
                    updateStepDraft(step.id, input);
                    await onSaveStep(step.id, input);
                  }}
                />
              )}
            </React.Fragment>
          ))}
        </div>
      ) : (
        <RunSheetResultEditor sheet={sheet} readOnly={readOnly} onSave={onSaveSheet} />
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

function TestRunStepDetail({
  step,
  draft,
  onDraftChange,
  onSave,
  readOnly,
}: {
  step: TestRunStep;
  draft: RunStepInput;
  onDraftChange: (draft: RunStepInput) => void;
  onSave: (input: RunStepInput) => Promise<void>;
  readOnly: boolean;
}) {
  const [isSaving, setIsSaving] = useState(false);

  const setStatusAndSave = async (status: TestRunStep['status']) => {
    if (readOnly) {
      return;
    }
    const nextDraft = { ...draft, status };
    onDraftChange(nextDraft);
    setIsSaving(true);
    try {
      await onSave(nextDraft);
    } finally {
      setIsSaving(false);
    }
  };
  const hasReadDetails = hasMarkdownContent(step.field) || hasMarkdownContent(step.expectedResult);

  return (
    <div className="run-step-detail">
      <div className="run-step-header">
        <div>
          <span className="section-kicker">Action {step.executionOrder}</span>
          {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} /> : <h4>Etape sans action</h4>}
        </div>
        <StatusBadge status={draft.status} />
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
        <textarea readOnly={readOnly} value={draft.actualResult} onChange={(event) => onDraftChange({ ...draft, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea readOnly={readOnly} value={draft.comment} onChange={(event) => onDraftChange({ ...draft, comment: event.target.value })} />
      </label>
      {!readOnly && (
        <div className="status-action-grid" aria-label="Changer le statut de l action">
          <Button type="button" variant="success" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('passed'); }}>Reussi</Button>
          <Button type="button" variant="danger" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('failed'); }}>Echoue</Button>
          <Button type="button" variant="warning" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('blocked'); }}>Bloque</Button>
          <Button type="button" variant="secondary" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('skipped'); }}>Ignore</Button>
        </div>
      )}
    </div>
  );
}

function RunSheetResultEditor({
  sheet,
  readOnly,
  onSave,
}: {
  sheet: TestRunSheet;
  readOnly: boolean;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
}) {
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
    if (readOnly) {
      return;
    }
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
        <select disabled={readOnly} value={value.status} onChange={(event) => setValue({ ...value, status: event.target.value as TestRunSheet['status'] })}>
          {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
        </select>
      </label>
      <label>
        Resultat obtenu
        <textarea readOnly={readOnly} value={value.actualResult} onChange={(event) => setValue({ ...value, actualResult: event.target.value })} />
      </label>
      <label>
        Commentaire
        <textarea readOnly={readOnly} value={value.comment} onChange={(event) => setValue({ ...value, comment: event.target.value })} />
      </label>
      {!readOnly && (
        <>
          <div className="status-action-grid" aria-label="Changer le statut du test">
            <Button type="button" variant="success" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'passed' })}>Reussi</Button>
            <Button type="button" variant="danger" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'failed' })}>Echoue</Button>
            <Button type="button" variant="warning" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'blocked' })}>Bloque</Button>
            <Button type="button" variant="secondary" size="sm" disabled={saving} onClick={() => save({ ...value, status: 'skipped' })}>Ignore</Button>
          </div>
          <Button type="button" disabled={saving} onClick={() => save()}>
            {saving ? 'Sauvegarde...' : 'Sauvegarder'}
          </Button>
        </>
      )}
    </div>
  );
}
