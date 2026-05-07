import React, { useEffect, useRef, useState } from 'react';
import { Evidence, RunSheetInput, RunStepInput, TestRunSheet, TestRunStep, testSheetApi } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { MarkdownCollapsibleSection } from '../ui/MarkdownCollapsibleSection';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';
import { SmartEllipsisText } from '../ui/SmartEllipsisText';
import { DocumentFilePicker, DocumentList, formatBytes } from './DocumentList';
import { StatusBadge } from './StatusBadge';
import { TestRunStepProgress } from './TestRunStepProgress';
import { getRunSheetProgress } from './runStatus';

type Props = {
  sheet: TestRunSheet;
  readOnly?: boolean;
  onSaveSheet: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onSaveStep: (stepId: number, input: RunStepInput) => Promise<void>;
  onEvidenceChanged: () => Promise<void>;
};

const statuses: TestRunStep['status'][] = ['pending', 'passed', 'failed', 'blocked', 'skipped'];

export function TestRunSheetDetail({ sheet, readOnly = false, onSaveSheet, onSaveStep, onEvidenceChanged }: Props) {
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
          {hasMarkdownContent(sheet.description) && (
            <div className="run-sheet-intro">
              <MarkdownPreview content={sheet.description} compact />
            </div>
          )}
        </div>
      </header>

      <TestRunStepProgress steps={sheet.steps ?? []} />

      {readOnly && <p className="readonly-notice">Execution en lecture seule</p>}

      {sheet.documents && sheet.documents.length > 0 && (
        <section className="run-read-details">
          <h4>Documents de la fiche</h4>
          <DocumentList documents={sheet.documents} />
        </section>
      )}

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
                    if (input.status !== 'passed' && input.status !== 'skipped') {
                      return;
                    }
                    const steps = sheet.steps ?? [];
                    const currentIndex = steps.findIndex((item) => item.id === step.id);
                    const nextStep = currentIndex >= 0 ? steps[currentIndex + 1] : undefined;
                    if (!nextStep) {
                      return;
                    }
                    removeStepDraft(step.id);
                    updateStepDraft(nextStep.id, getStepDraft(nextStep));
                    setOpenedStepId(nextStep.id);
                  }}
                />
              )}
            </React.Fragment>
          ))}
        </div>
      ) : (
        <p className="muted">Aucune action dans cette fiche</p>
      )}

      <RunSheetCommentEditor sheet={sheet} readOnly={readOnly} onSave={onSaveSheet} />
      <RunSheetEvidenceList
        runId={sheet.runId}
        runSheetId={sheet.id}
        evidences={sheet.evidences}
        readOnly={readOnly}
        onChanged={onEvidenceChanged}
      />
    </Card>
  );
}

function RunSheetReadDetails({ sheet }: { sheet: TestRunSheet }) {
  const details = [
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
  const hasField = hasMarkdownContent(step.field);

  return (
    <div className="run-step-detail">
      <div className="run-step-header">
        <div>
          <span className="section-kicker">Action {step.executionOrder}</span>
          {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} /> : <h4>Etape sans action</h4>}
        </div>
        <StatusBadge status={draft.status} />
      </div>
      {hasField && (
        <dl className="compact-definition-list">
          <dt>Specifique</dt>
          <dd><MarkdownPreview content={step.field} compact /></dd>
        </dl>
      )}
      {hasMarkdownContent(step.expectedResult) && (
        <section className="expected-result-block">
          <h4 className="expected-result-title">Resultat attendu</h4>
          <div className="expected-result-content">
            <MarkdownPreview content={step.expectedResult} compact />
          </div>
        </section>
      )}
      {step.documents && step.documents.length > 0 && (
        <section className="run-read-details">
          <h4>Documents de l action</h4>
          <DocumentList documents={step.documents} />
        </section>
      )}
      {readOnly ? (
        <RunResultReadDetails actualResult={step.actualResult} comment={step.comment} />
      ) : (
        <>
          <label>
            Resultat obtenu
            <textarea value={draft.actualResult} onChange={(event) => onDraftChange({ ...draft, actualResult: event.target.value })} />
          </label>
          <label>
            Commentaire
            <textarea value={draft.comment} onChange={(event) => onDraftChange({ ...draft, comment: event.target.value })} />
          </label>
        </>
      )}
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

function RunSheetCommentEditor({
  sheet,
  readOnly,
  onSave,
}: {
  sheet: TestRunSheet;
  readOnly: boolean;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
}) {
  const [commentDraft, setCommentDraft] = useState(sheet.comment);
  const [savedComment, setSavedComment] = useState(sheet.comment);
  const [saveState, setSaveState] = useState<'saved' | 'dirty' | 'saving' | 'success' | 'error'>('saved');
  const [error, setError] = useState('');
  const progress = getRunSheetProgress(sheet);
  const isDirty = commentDraft !== savedComment;

  useEffect(() => {
    const nextComment = sheet.comment ?? '';
    setCommentDraft(nextComment);
    setSavedComment(nextComment);
    setSaveState('saved');
    setError('');
  }, [sheet.id, sheet.comment]);

  useEffect(() => {
    if (saveState !== 'success') {
      return undefined;
    }
    const timeout = window.setTimeout(() => setSaveState('saved'), 1500);
    return () => window.clearTimeout(timeout);
  }, [saveState]);

  const updateComment = (nextComment: string) => {
    setCommentDraft(nextComment);
    setError('');
    setSaveState(nextComment === savedComment ? 'saved' : 'dirty');
  };

  const save = async () => {
    if (readOnly || !isDirty || saveState === 'saving') {
      return;
    }
    setSaveState('saving');
    setError('');
    try {
      await onSave(sheet.id, {
        status: progress.status,
        actualResult: sheet.actualResult,
        comment: commentDraft,
      });
      setSavedComment(commentDraft);
      setSaveState('success');
    } catch (err) {
      setError((err as Error).message);
      setSaveState('error');
    }
  };

  const buttonLabel = () => {
    if (saveState === 'saving') {
      return 'Enregistrement...';
    }
    if (saveState === 'error') {
      return 'Erreur, reessayer';
    }
    if (saveState === 'dirty') {
      return 'Enregistrer le commentaire';
    }
    return 'Commentaire enregistre';
  };

  const buttonVariant = (): 'primary' | 'secondary' | 'danger' | 'success' => {
    if (saveState === 'success') {
      return 'success';
    }
    if (saveState === 'error') {
      return 'danger';
    }
    if (saveState === 'dirty') {
      return 'primary';
    }
    return 'secondary';
  };

  return (
    <div className="run-step-detail">
      <h4>Commentaire de la fiche</h4>
      {readOnly ? (
        hasMarkdownContent(sheet.comment)
          ? <MarkdownPreview content={sheet.comment} />
          : <p className="muted">Aucun commentaire</p>
      ) : (
        <label>
          Commentaire
          <textarea value={commentDraft} onChange={(event) => updateComment(event.target.value)} />
        </label>
      )}
      {!readOnly && (
        <Button
          type="button"
          variant={buttonVariant()}
          className={saveState === 'saved' ? 'soft-save-button' : ''}
          disabled={saveState === 'saving' || !isDirty}
          onClick={() => { void save(); }}
        >
          {buttonLabel()}
        </Button>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  );
}

function RunSheetEvidenceList({
  runId,
  runSheetId,
  evidences = [],
  readOnly,
  onChanged,
}: {
  runId: number;
  runSheetId: number;
  evidences?: Evidence[];
  readOnly: boolean;
  onChanged: () => Promise<void>;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | undefined>();
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState('');

  const upload = async () => {
    if (!file || readOnly) {
      return;
    }
    setIsUploading(true);
    setError('');
    try {
      await testSheetApi.uploadRunSheetEvidence(runId, runSheetId, file);
      setFile(undefined);
      if (inputRef.current) {
        inputRef.current.value = '';
      }
      await onChanged();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setIsUploading(false);
    }
  };

  const remove = async (evidence: Evidence) => {
    if (readOnly) {
      return;
    }
    setError('');
    try {
      await testSheetApi.deleteEvidence(evidence.id);
      await onChanged();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  return (
    <section className="run-read-details">
      <h4>Documents ajoutes</h4>
      {!readOnly && (
        <div className="document-upload-panel">
          <DocumentFilePicker
            id={`run-sheet-evidence-${runSheetId}`}
            file={file}
            inputRef={inputRef}
            onFileChange={setFile}
            label="Ajouter un document"
          />
          <Button type="button" disabled={!file || isUploading} onClick={() => { void upload(); }}>
            {isUploading ? 'Ajout...' : 'Ajouter'}
          </Button>
        </div>
      )}
      {error && <p className="error">{error}</p>}
      {evidences.length === 0 ? (
        <p className="muted">Aucun document ajoute</p>
      ) : (
        <div className="document-list">
          {evidences.map((evidence) => (
            <div className="document-list-item" key={evidence.id}>
              <div className="document-content">
                <div className="document-title-row">
                  <span className="document-name" title={evidence.name}>{evidence.name}</span>
                  <span className="document-size">{formatBytes(evidence.sizeBytes)}</span>
                </div>
              </div>
              <div className="button-row end">
                <a className="ui-button secondary sm" href={testSheetApi.evidenceDownloadUrl(evidence.id)}>Telecharger</a>
                {!readOnly && <Button type="button" size="sm" variant="danger" onClick={() => { void remove(evidence); }}>Supprimer</Button>}
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function RunResultReadDetails({ actualResult, comment }: { actualResult: string; comment: string }) {
  const hasActualResult = hasMarkdownContent(actualResult);
  const hasComment = hasMarkdownContent(comment);

  if (!hasActualResult && !hasComment) {
    return <p className="muted">Aucun resultat renseigne</p>;
  }

  return (
    <div className="run-read-details">
      {hasActualResult && (
        <section>
          <h4>Resultat obtenu</h4>
          <MarkdownPreview content={actualResult} />
        </section>
      )}
      {hasComment && (
        <section>
          <h4>Commentaire</h4>
          <MarkdownPreview content={comment} />
        </section>
      )}
    </div>
  );
}
