import React, { useEffect, useRef, useState } from 'react';
import { Evidence, RunSheetInput, RunStepInput, TestRunSheet, TestRunStep, testSheetApi } from '../../api/testSheet';
import { messages } from '../../../../i18n';
import { Button } from '../../../../shared/components/ui/Button';
import { Card } from '../../../../shared/components/ui/Card';
import { MarkdownCollapsibleSection } from '../../../../shared/components/ui/MarkdownCollapsibleSection';
import { hasMarkdownContent, MarkdownPreview } from '../../../../shared/components/ui/MarkdownPreview';
import { MarkdownTextarea } from '../../../../shared/components/ui/MarkdownTextarea';
import { SmartEllipsisText } from '../../../../shared/components/ui/SmartEllipsisText';
import { DocumentFilePicker, DocumentList, formatBytes } from '../sheet/DocumentList';
import { StatusBadge } from '../execution/StatusBadge';
import { TestRunStepProgress } from './TestRunStepProgress';
import { getRunSheetProgress } from '../execution/runStatus';

type Props = {
  sheet: TestRunSheet;
  readOnly?: boolean;
  onSaveSheet: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onSaveStep: (stepId: number, input: RunStepInput) => Promise<void>;
  onEvidenceChanged: () => Promise<void>;
};

export function TestRunSheetDetail({ sheet, readOnly = false, onSaveSheet, onSaveStep, onEvidenceChanged }: Props) {
  const [openedStepId, setOpenedStepId] = useState<number | undefined>();
  const [stepDrafts, setStepDrafts] = useState<Record<number, RunStepInput>>({});
  const [showSheetEvidenceSection, setShowSheetEvidenceSection] = useState((sheet.evidences ?? []).length > 0);
  const [showSheetCommentSection, setShowSheetCommentSection] = useState(hasMarkdownContent(sheet.comment));
  const progress = getRunSheetProgress(sheet);
  const hasSheetEvidences = (sheet.evidences ?? []).length > 0;
  const hasSheetComment = hasMarkdownContent(sheet.comment);

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
    setShowSheetEvidenceSection((sheet.evidences ?? []).length > 0);
    setShowSheetCommentSection(hasMarkdownContent(sheet.comment));
  }, [sheet.id, sheet.evidences, sheet.comment]);

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

      {readOnly && <p className="readonly-notice">{messages.testSheet.run.readOnly}</p>}

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
                  <SmartEllipsisText text={hasMarkdownContent(step.action) ? step.action : messages.testSheet.run.unnamedAction} />
                </div>
                <StatusBadge status={getStepDraft(step).status} />
              </div>
              {step.id === openedStepId && (
                <TestRunStepDetail
                  draft={getStepDraft(step)}
                  runId={sheet.runId}
                  step={step}
                  onDraftChange={(draft) => updateStepDraft(step.id, draft)}
                  onEvidenceChanged={onEvidenceChanged}
                  readOnly={readOnly}
                  onSave={async (input) => {
                    if (readOnly) {
                      return;
                    }
                    updateStepDraft(step.id, input);
                    await onSaveStep(step.id, input);
                    const steps = sheet.steps ?? [];
                    const currentIndex = steps.findIndex((item) => item.id === step.id);
                    const nextStep = currentIndex >= 0 ? steps[currentIndex + 1] : undefined;
                    const shouldOpenNext = input.status === 'passed' || input.status === 'skipped';
                    if (shouldOpenNext && nextStep) {
                      removeStepDraft(step.id);
                      updateStepDraft(nextStep.id, getStepDraft(nextStep));
                      setOpenedStepId(nextStep.id);
                      return;
                    }
                    if (!nextStep) {
                      removeStepDraft(step.id);
                      setOpenedStepId(undefined);
                    }
                  }}
                />
              )}
            </React.Fragment>
          ))}
        </div>
      ) : (
        <p className="muted">{messages.testSheet.run.noActions}</p>
      )}

      <h4>{messages.testSheet.run.sheetResult}</h4>
      {(!readOnly || hasSheetEvidences) && (
        <RunOptionalSection
          title={messages.testSheet.run.documentsAdded}
          open={showSheetEvidenceSection}
          hasContent={hasSheetEvidences}
          onOpen={() => setShowSheetEvidenceSection(true)}
        >
          <RunEvidenceList
            evidences={sheet.evidences}
            readOnly={readOnly}
            upload={(file) => testSheetApi.uploadRunSheetEvidence(sheet.runId, sheet.id, file)}
            remove={(evidence) => testSheetApi.deleteEvidence(evidence.id)}
            downloadUrl={(evidence) => testSheetApi.evidenceDownloadUrl(evidence.id)}
            onChanged={onEvidenceChanged}
            bare
          />
        </RunOptionalSection>
      )}
      {(!readOnly || hasSheetComment) && (
        <RunOptionalSection
          title={messages.testSheet.run.comment}
          open={showSheetCommentSection}
          hasContent={hasSheetComment}
          onOpen={() => setShowSheetCommentSection(true)}
        >
          <RunSheetCommentEditor
            sheet={sheet}
            readOnly={readOnly}
            onSave={onSaveSheet}
            onCloseIfEmpty={() => setShowSheetCommentSection(false)}
          />
        </RunOptionalSection>
      )}
      <div className="run-sheet-footer-actions">
        <Button
          type="button"
          variant="secondary"
          onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
        >
          ↑ {messages.testSheet.run.backToTop}
        </Button>
      </div>
    </Card>
  );
}

function RunSheetReadDetails({ sheet }: { sheet: TestRunSheet }) {
  const details = [
    [messages.testSheet.edit.prerequisites, sheet.prerequisites],
    [messages.testSheet.edit.configuration, sheet.config],
    [messages.testSheet.edit.command, sheet.command],
    [messages.testSheet.edit.notes, sheet.notes],
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
  runId,
  step,
  draft,
  onDraftChange,
  onEvidenceChanged,
  onSave,
  readOnly,
}: {
  runId: number;
  step: TestRunStep;
  draft: RunStepInput;
  onDraftChange: (draft: RunStepInput) => void;
  onEvidenceChanged: () => Promise<void>;
  onSave: (input: RunStepInput) => Promise<void>;
  readOnly: boolean;
}) {
  const [isSaving, setIsSaving] = useState(false);
  const [showEvidenceSection, setShowEvidenceSection] = useState((step.evidences ?? []).length > 0);
  const [showCommentSection, setShowCommentSection] = useState(hasMarkdownContent(draft.comment || step.comment));
  const mountedRef = useRef(true);

  useEffect(() => {
    setShowEvidenceSection((step.evidences ?? []).length > 0);
    setShowCommentSection(hasMarkdownContent(draft.comment || step.comment));
  }, [step.id, step.evidences, step.comment]);

  useEffect(() => () => {
    mountedRef.current = false;
  }, []);

  const setStatusAndSave = async (status: TestRunStep['status']) => {
    if (readOnly) {
      return;
    }
    const nextDraft = { ...draft, status };
    onDraftChange(nextDraft);
    setIsSaving(true);
    try {
      await onSave(nextDraft);
      if (mountedRef.current) {
        setShowEvidenceSection((step.evidences ?? []).length > 0);
        setShowCommentSection(hasMarkdownContent(nextDraft.comment));
      }
    } finally {
      if (mountedRef.current) {
        setIsSaving(false);
      }
    }
  };
  const hasField = hasMarkdownContent(step.field);
  const hasEvidences = (step.evidences ?? []).length > 0;
  const hasComment = hasMarkdownContent(draft.comment || step.comment);

  return (
    <div className="run-step-detail">
      <div className="run-step-header">
        <div>
          <span className="section-kicker">{messages.testSheet.run.action} {step.executionOrder}</span>
          {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} /> : <h4>{messages.testSheet.run.noAction}</h4>}
        </div>
        <StatusBadge status={draft.status} />
      </div>
      {hasField && (
        <dl className="compact-definition-list">
          <dt>{messages.testSheet.edit.specificField}</dt>
          <dd><MarkdownPreview content={step.field} compact /></dd>
        </dl>
      )}
      {hasMarkdownContent(step.expectedResult) && (
        <section className="expected-result-block">
          <h4 className="expected-result-title">{messages.testSheet.run.expectedResult}</h4>
          <div className="expected-result-content">
            <MarkdownPreview content={step.expectedResult} compact />
          </div>
        </section>
      )}
      {step.documents && step.documents.length > 0 && (
        <section className="run-read-details">
          <h4>{messages.testSheet.run.actionDocuments}</h4>
          <DocumentList documents={step.documents} />
        </section>
      )}
      {(!readOnly || hasEvidences) && (
        <RunOptionalSection
          title={messages.testSheet.run.documentsAdded}
          open={showEvidenceSection}
          hasContent={hasEvidences}
          onOpen={() => setShowEvidenceSection(true)}
        >
          <RunEvidenceList
            evidences={step.evidences}
            readOnly={readOnly}
            upload={(file) => testSheetApi.uploadRunStepEvidence(runId, step.id, file)}
            remove={(evidence) => testSheetApi.deleteRunStepEvidence(evidence.id)}
            downloadUrl={(evidence) => testSheetApi.runStepEvidenceDownloadUrl(evidence.id)}
            onChanged={onEvidenceChanged}
            bare
          />
        </RunOptionalSection>
      )}
      {(!readOnly || hasComment) && (
        <RunOptionalSection
          title={messages.testSheet.run.comment}
          open={showCommentSection}
          hasContent={hasComment}
          onOpen={() => setShowCommentSection(true)}
        >
          {readOnly ? (
            <MarkdownPreview content={step.comment} />
          ) : (
            <MarkdownTextarea
              value={draft.comment}
              onChange={(comment) => onDraftChange({ ...draft, comment })}
            />
          )}
        </RunOptionalSection>
      )}
      {!readOnly && (
        <div className="status-action-grid" aria-label="Changer le statut de l action">
          <Button type="button" variant="success" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('passed'); }}>{messages.status.passed}</Button>
          <Button type="button" variant="danger" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('failed'); }}>{messages.status.failed}</Button>
          <Button type="button" variant="warning" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('blocked'); }}>{messages.status.blocked}</Button>
          <Button type="button" variant="secondary" size="sm" disabled={isSaving} onClick={(event) => { event.stopPropagation(); void setStatusAndSave('skipped'); }}>{messages.status.skipped}</Button>
        </div>
      )}
    </div>
  );
}

function RunOptionalSection({
  title,
  open,
  hasContent,
  onOpen,
  children,
}: {
  title: string;
  open: boolean;
  hasContent: boolean;
  onOpen: () => void;
  children: React.ReactNode;
}) {
  if (!open && !hasContent) {
    return (
      <div className="run-optional-section collapsed">
        <span className="run-optional-section-title">{title}</span>
        <Button type="button" variant="secondary" size="sm" onClick={onOpen}>
          +
        </Button>
      </div>
    );
  }

  return (
    <section className="run-optional-section">
      <div className="run-optional-section-header">
        <h4>{title}</h4>
      </div>
      <div className="run-optional-section-body">
        {children}
      </div>
    </section>
  );
}

function RunSheetCommentEditor({
  sheet,
  readOnly,
  onSave,
  onCloseIfEmpty,
}: {
  sheet: TestRunSheet;
  readOnly: boolean;
  onSave: (sheetId: number, input: RunSheetInput) => Promise<void>;
  onCloseIfEmpty: () => void;
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
      if (!hasMarkdownContent(commentDraft)) {
        onCloseIfEmpty();
      }
    } catch (err) {
      setError((err as Error).message);
      setSaveState('error');
    }
  };

  const buttonLabel = () => {
    if (saveState === 'saving') {
      return messages.testSheet.run.savingComment;
    }
    if (saveState === 'error') {
      return messages.testSheet.run.saveError;
    }
    if (saveState === 'dirty') {
      return messages.testSheet.run.saveComment;
    }
    return messages.testSheet.run.savedComment;
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
    <>
      {readOnly ? (
        <MarkdownPreview content={sheet.comment} />
      ) : (
        <MarkdownTextarea
          value={commentDraft}
          onChange={updateComment}
        />
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
    </>
  );
}

function RunEvidenceList({
  evidences = [],
  readOnly,
  upload: uploadEvidence,
  remove: removeEvidence,
  downloadUrl,
  onChanged,
  bare = false,
}: {
  evidences?: Evidence[];
  readOnly: boolean;
  upload: (file: File) => Promise<Evidence>;
  remove: (evidence: Evidence) => Promise<void>;
  downloadUrl: (evidence: Evidence) => string;
  onChanged: () => Promise<void>;
  bare?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const inputId = React.useId();
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
      await uploadEvidence(file);
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
      await removeEvidence(evidence);
      await onChanged();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const content = (
    <>
      {!readOnly && (
        <div className="document-upload-panel">
          <DocumentFilePicker
            id={inputId}
            file={file}
            inputRef={inputRef}
            onFileChange={setFile}
            label={messages.testSheet.documents.chooseFile}
          />
          <Button type="button" disabled={!file || isUploading} onClick={() => { void upload(); }}>
            {isUploading ? messages.common.saving : messages.testSheet.documents.addDocument}
          </Button>
        </div>
      )}
      {error && <p className="error">{error}</p>}
      {evidences.length === 0 ? (
        <p className="muted">{messages.testSheet.run.noDocumentsAdded}</p>
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
                <a className="ui-button secondary sm" href={downloadUrl(evidence)}>{messages.common.download}</a>
                {!readOnly && <Button type="button" size="sm" variant="danger" onClick={() => { void remove(evidence); }}>{messages.common.delete}</Button>}
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );

  if (bare) {
    return content;
  }

  return (
    <section className="run-read-details">
      <h4>{messages.testSheet.run.documentsAdded}</h4>
      {content}
    </section>
  );
}
