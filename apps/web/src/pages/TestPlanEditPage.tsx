import React, { useEffect, useMemo, useRef, useState } from 'react';
import { testSheetApi, TestDocument, TestPlan, TestSheet } from '../api/testSheet';
import { DocumentFilePicker, DocumentList } from '../components/test-sheet/DocumentList';
import { TestPlanForm } from '../components/test-sheet/TestPlanForm';
import { TestSheetEditor, TestSheetEditorHandle } from '../components/test-sheet/TestSheetEditor';
import { TestSheetList } from '../components/test-sheet/TestSheetList';
import { Button } from '../components/ui/Button';
import { Card, CardHeader } from '../components/ui/Card';
import { PageHeader } from '../components/ui/PageHeader';
import { messages } from '../i18n';

type Props = {
  planId: number;
  onBack: () => void;
  onRun: (runId: number) => void;
};

type SheetEditorMode = 'closed' | 'create' | 'edit';

const modelChangedRunCanceledMessage = messages.testSheet.dialogs.modelChangedRunCanceled;

export function TestPlanEditPage({ planId, onBack, onRun }: Props) {
  const [plan, setPlan] = useState<TestPlan | undefined>();
  const [sheets, setSheets] = useState<TestSheet[]>([]);
  const [documents, setDocuments] = useState<TestDocument[]>([]);
  const [sheetEditorMode, setSheetEditorMode] = useState<SheetEditorMode>('closed');
  const [editingSheet, setEditingSheet] = useState<TestSheet | undefined>();
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const sheetEditorRef = useRef<TestSheetEditorHandle>(null);
  const sheetEditorContainerRef = useRef<HTMLDivElement | null>(null);

  const isNew = planId === 0 && !plan;
  const effectivePlanId = plan?.id ?? planId;
  const nextOrder = useMemo(() => Math.max(0, ...sheets.map((sheet) => sheet.executionOrder)) + 1, [sheets]);

  const load = async () => {
    if (isNew) {
      return;
    }
    const [loadedPlan, loadedSheets, loadedDocuments] = await Promise.all([
      testSheetApi.getPlan(planId),
      testSheetApi.listSheets(planId),
      testSheetApi.listDocuments(planId),
    ]);
    setPlan(loadedPlan);
    setSheets(loadedSheets);
    setDocuments(loadedDocuments);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, [planId]);

  const refreshSheets = async () => {
    const loadedSheets = await testSheetApi.listSheets(effectivePlanId);
    setSheets(loadedSheets);
    if (editingSheet) {
      setEditingSheet(loadedSheets.find((item) => item.id === editingSheet.id));
    }
    return loadedSheets;
  };

  const refreshDocuments = async () => {
    if (!effectivePlanId) {
      return [];
    }
    const loadedDocuments = await testSheetApi.listDocuments(effectivePlanId);
    setDocuments(loadedDocuments);
    return loadedDocuments;
  };

  const closeEditor = () => {
    setSheetEditorMode('closed');
    setEditingSheet(undefined);
  };

  const savePlan = async (input: { name: string; description: string; mockupSettings: string }) => {
    const saved = isNew
      ? await testSheetApi.createPlan(input)
      : await runModelMutation(() => testSheetApi.updatePlan(effectivePlanId, input));
    setPlan(saved);
  };

  const runModelMutation = async <T,>(mutation: () => Promise<T>): Promise<T> => {
    setInfo('');
    const hadRunningRun = await hasRunningRun(effectivePlanId);
    const result = await mutation();
    if (hadRunningRun) {
      setInfo(modelChangedRunCanceledMessage);
    }
    return result;
  };

  const afterSheetSaved = async () => {
    await refreshSheets();
    closeEditor();
  };

  const afterSheetCreated = (sheet: TestSheet) => {
    setEditingSheet(sheet);
    setSheetEditorMode('edit');
    scrollToSheetEditor();
  };

  const openCreateSheet = () => {
    setEditingSheet(undefined);
    setSheetEditorMode('create');
    scrollToSheetEditor();
  };

  const openEditSheet = (sheet: TestSheet) => {
    setEditingSheet(sheet);
    setSheetEditorMode('edit');
    scrollToSheetEditor();
  };

  const scrollToSheetEditor = () => {
    requestAnimationFrame(() => {
      sheetEditorContainerRef.current?.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
      });
    });
  };

  const toggleEditSheet = async (sheet: TestSheet) => {
    if (sheetEditorMode === 'edit' && editingSheet?.id === sheet.id) {
      await sheetEditorRef.current?.submit();
      return;
    }
    if (sheetEditorMode === 'edit') {
      await sheetEditorRef.current?.submit();
    }
    openEditSheet(sheet);
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow={messages.testSheet.plans.editEyebrow}
        title={isNew ? messages.testSheet.plans.newPlan : plan?.name ?? messages.testSheet.plans.testPlan}
        description={isNew ? messages.testSheet.plans.savePlanBeforeSheets : `${sheets.length} ${messages.testSheet.plans.sheetSingular}${sheets.length > 1 ? 's' : ''} ${messages.testSheet.plans.sheetsInPlan}`}
        onBack={onBack}
        actions={!isNew && (
          <Button
            type="button"
            disabled={sheets.length === 0}
            onClick={async () => {
              const run = await testSheetApi.createRun(effectivePlanId);
              onRun(run.id);
            }}
          >
            {messages.testSheet.plans.startRun}
          </Button>
        )}
      />

      {error && <p className="error">{error}</p>}
      {info && <p className="info-message">{info}</p>}

      <Card>
        <CardHeader>
          <div>
            <span className="section-kicker">{messages.testSheet.plans.generalInfo}</span>
            <h3>{messages.testSheet.plans.plan}</h3>
          </div>
        </CardHeader>
        <TestPlanForm plan={plan} onSubmit={savePlan} />
      </Card>

      {plan && (
        <Card>
          <CardHeader>
            <div>
              <span className="section-kicker">{messages.testSheet.plans.library}</span>
              <h3>{messages.testSheet.plans.planDocuments}</h3>
            </div>
          </CardHeader>
          <PlanDocumentsPanel
            planId={effectivePlanId}
            documents={documents}
            onChanged={async () => {
              await refreshDocuments();
              await refreshSheets();
            }}
          />
        </Card>
      )}

      <section className="sheet-list-section">
        <div className="section-header">
          <div>
            <span className="section-kicker">{messages.testSheet.plans.executionOrder}</span>
            <h3>{messages.testSheet.edit.sheets}</h3>
          </div>
        </div>

        {plan && (
          <>
            <TestSheetList
              sheets={sheets}
              onEdit={toggleEditSheet}
              onDelete={async (sheet) => {
                await runModelMutation(() => testSheetApi.deleteSheet(sheet.id));
                await refreshSheets();
                if (editingSheet?.id === sheet.id) {
                  closeEditor();
                }
              }}
              onDuplicate={async (sheet) => {
                await runModelMutation(() => testSheetApi.duplicateSheet(sheet.id));
                await refreshSheets();
              }}
              onMove={async (sheet, direction) => {
                const currentIndex = sheets.findIndex((item) => item.id === sheet.id);
                const next = [...sheets];
                const targetIndex = currentIndex + direction;
                [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];
                await runModelMutation(() => testSheetApi.reorderSheets(effectivePlanId, next.map((item) => item.id)));
                await refreshSheets();
              }}
              editingSheetId={sheetEditorMode === 'edit' ? editingSheet?.id : undefined}
              renderEditor={(sheet) => (
                <div ref={sheetEditorContainerRef}>
                  <TestSheetEditor
                    ref={sheetEditorRef}
                    mode="edit"
                    planId={effectivePlanId}
                    sheet={sheet}
                    nextOrder={nextOrder}
                    onCancel={closeEditor}
                    onSaved={afterSheetSaved}
                    onCreated={afterSheetCreated}
                    onRefresh={refreshSheets}
                    onModelMutation={runModelMutation}
                    planDocuments={documents}
                    onDocumentsChanged={async () => {
                      await refreshDocuments();
                      await refreshSheets();
                    }}
                  />
                </div>
              )}
            />

            {sheetEditorMode === 'closed' && (
              <div className="add-sheet-row">
                <Button type="button" onClick={openCreateSheet}>+ {messages.testSheet.edit.addSheet}</Button>
              </div>
            )}

            {sheetEditorMode === 'create' && (
              <div ref={sheetEditorContainerRef}>
                <TestSheetEditor
                  ref={sheetEditorRef}
                  mode="create"
                  planId={effectivePlanId}
                  sheet={editingSheet}
                  nextOrder={nextOrder}
                  onCancel={closeEditor}
                  onSaved={afterSheetSaved}
                  onCreated={afterSheetCreated}
                  onRefresh={refreshSheets}
                  onModelMutation={runModelMutation}
                  planDocuments={documents}
                  onDocumentsChanged={async () => {
                    await refreshDocuments();
                    await refreshSheets();
                  }}
                />
              </div>
            )}
          </>
        )}
      </section>
    </section>
  );
}

async function hasRunningRun(planId: number) {
  if (!planId) {
    return false;
  }
  const runs = await testSheetApi.listPlanRuns(planId);
  return runs.some((run) => run.status === 'running');
}

function PlanDocumentsPanel({
  planId,
  documents,
  onChanged,
}: {
  planId: number;
  documents: TestDocument[];
  onChanged: () => Promise<void>;
}) {
  const [file, setFile] = useState<File | undefined>();
  const [description, setDescription] = useState('');
  const [uploading, setUploading] = useState(false);
  const [deletingDocumentId, setDeletingDocumentId] = useState<number | undefined>();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const fileInputId = React.useId();

  const resetUploadForm = () => {
    setFile(undefined);
    setDescription('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const upload = async () => {
    if (!file) {
      return;
    }
    setUploading(true);
    try {
      await testSheetApi.uploadDocument(planId, file, description);
      resetUploadForm();
      await onChanged();
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="document-panel">
      <DocumentList
        documents={documents}
        onDelete={async (document) => {
          if (!window.confirm(messages.testSheet.dialogs.deletePlanDocumentConfirm)) {
            return;
          }
          setDeletingDocumentId(document.id);
          try {
            await testSheetApi.deleteDocument(document.id);
            await onChanged();
          } finally {
            setDeletingDocumentId(undefined);
            resetUploadForm();
          }
        }}
      />
      <div className="document-upload-row">
        <DocumentFilePicker
          id={fileInputId}
          file={file}
          inputRef={fileInputRef}
          onFileChange={setFile}
          label={`+ ${messages.testSheet.documents.chooseFile}`}
        />
        <input value={description} onChange={(event) => setDescription(event.target.value)} />
        <Button type="button" disabled={!file || uploading || deletingDocumentId !== undefined} onClick={upload}>{uploading ? messages.testSheet.documents.importing : `+ ${messages.testSheet.documents.addDocument}`}</Button>
      </div>
    </div>
  );
}
