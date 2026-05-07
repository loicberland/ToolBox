import React, { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { SheetInput, StepInput, testSheetApi, TestDocument, TestSheet, TestSheetStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card, CardHeader } from '../ui/Card';
import { messages } from '../../i18n';
import { DocumentFilePicker, DocumentList } from './DocumentList';
import { TestSheetForm, TestSheetFormHandle } from './TestSheetForm';
import { TestStepForm, TestStepFormHandle } from './TestStepForm';
import { TestStepList } from './TestStepList';

type Mode = 'create' | 'edit';
type StepEditorMode = 'closed' | 'create' | 'edit';

const emptySheet: TestSheet = {
  id: 0,
  planId: 0,
  name: '',
  description: '',
  prerequisites: '',
  config: '',
  command: '',
  notes: '',
  action: '',
  expectedResult: '',
  executionOrder: 1,
  mockupSettings: '',
  steps: [],
};

type Props = {
  mode: Mode;
  planId: number;
  sheet?: TestSheet;
  nextOrder: number;
  onCancel: () => void;
  onSaved: () => Promise<void>;
  onCreated: (sheet: TestSheet) => void;
  onRefresh: () => Promise<TestSheet[]>;
  onModelMutation: <T>(mutation: () => Promise<T>) => Promise<T>;
  planDocuments: TestDocument[];
  onDocumentsChanged: () => Promise<void>;
};

export type TestSheetEditorHandle = {
  submit: () => Promise<void>;
};

export const TestSheetEditor = forwardRef<TestSheetEditorHandle, Props>(function TestSheetEditor({ mode, planId, sheet, nextOrder, onCancel, onSaved, onCreated, onRefresh, onModelMutation, planDocuments, onDocumentsChanged }, ref) {
  const sheetFormRef = useRef<TestSheetFormHandle>(null);
  const stepFormRef = useRef<TestStepFormHandle>(null);
  const [currentSheet, setCurrentSheet] = useState<TestSheet | undefined>(sheet);
  const [stepEditorMode, setStepEditorMode] = useState<StepEditorMode>('closed');
  const [editingStep, setEditingStep] = useState<TestSheetStep | undefined>();

  const isCreate = mode === 'create';
  const formId = `test-sheet-form-${sheet?.id ?? 'new'}`;
  const activeSheet = currentSheet ?? sheet;
  const canEditSteps = !isCreate && Boolean(activeSheet?.id);
  const steps = canEditSteps ? (activeSheet?.steps ?? []) : [];
  const nextStepOrder = useMemo(() => Math.max(0, ...steps.map((step) => step.executionOrder)) + 1, [steps]);

  useEffect(() => {
    setCurrentSheet(sheet);
    closeStepEditor();
  }, [mode, sheet?.id]);

  useEffect(() => {
    setCurrentSheet(sheet);
  }, [sheet]);

  useImperativeHandle(ref, () => ({
    submit: async () => {
      await sheetFormRef.current?.submit();
    },
  }));

  const openCreateStep = () => {
    setEditingStep(undefined);
    setStepEditorMode('create');
  };

  const openEditStep = async (step: TestSheetStep) => {
    if (stepEditorMode === 'edit' && editingStep?.id === step.id) {
      await stepFormRef.current?.submit();
      return;
    }
    if (stepEditorMode === 'edit') {
      await stepFormRef.current?.submit();
    }
    setEditingStep(step);
    setStepEditorMode('edit');
  };

  const closeStepEditor = () => {
    setEditingStep(undefined);
    setStepEditorMode('closed');
  };

  const refreshCurrentSheet = async () => {
    const loadedSheets = await onRefresh();
    if (activeSheet) {
      setCurrentSheet(loadedSheets.find((item) => item.id === activeSheet.id));
    }
  };

  const saveSheet = async (input: SheetInput) => {
    const normalizedInput = {
      ...input,
      mockupSettings: sheet?.mockupSettings ?? input.mockupSettings ?? '',
    };
    if (isCreate) {
      const created = await onModelMutation(() => testSheetApi.createSheet(planId, normalizedInput));
      const loadedSheets = await onRefresh();
      const createdSheet = loadedSheets.find((item) => item.id === created.id) ?? created;
      closeStepEditor();
      onCreated(createdSheet);
      return;
    } else if (sheet) {
      await onModelMutation(() => testSheetApi.updateSheet(sheet.id, normalizedInput));
    }
    await onSaved();
  };

  const saveStep = async (input: StepInput) => {
    if (!canEditSteps || !activeSheet) {
      return;
    }
    if (editingStep) {
      await onModelMutation(() => testSheetApi.updateStep(editingStep.id, input));
    } else {
      await onModelMutation(() => testSheetApi.createStep(activeSheet.id, input));
    }
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const deleteStep = async (step: TestSheetStep) => {
    if (!canEditSteps) {
      return;
    }
    await onModelMutation(() => testSheetApi.deleteStep(step.id));
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const duplicateStep = async (step: TestSheetStep) => {
    if (!canEditSteps) {
      return;
    }
    await onModelMutation(() => testSheetApi.duplicateStep(step.id));
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const moveStep = async (step: TestSheetStep, direction: -1 | 1) => {
    if (!canEditSteps || !activeSheet) {
      return;
    }
    const currentIndex = steps.findIndex((item) => item.id === step.id);
    const targetIndex = currentIndex + direction;
    if (currentIndex === -1 || targetIndex < 0 || targetIndex >= steps.length) {
      return;
    }
    const next = [...steps];
    [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];

    await onModelMutation(() => testSheetApi.reorderSteps(activeSheet.id, next.map((item) => item.id)));
    await refreshCurrentSheet();
    closeStepEditor();
  };

  return (
    <Card className="sheet-editor-card">
      <CardHeader>
        <div>
          <span className="section-kicker">{isCreate ? messages.testSheet.edit.newSheet : messages.testSheet.edit.editing}</span>
          <h3>{isCreate ? messages.testSheet.edit.addSheet : activeSheet?.name ?? messages.testSheet.edit.editSheet}</h3>
        </div>
      </CardHeader>
      <TestSheetForm
        ref={sheetFormRef}
        sheet={activeSheet ?? { ...emptySheet, executionOrder: nextOrder }}
        nextOrder={nextOrder}
        onSubmit={saveSheet}
        formId={formId}
        hideActions
      />
      {canEditSteps && activeSheet && (
        <DocumentAssociationPanel
          title={messages.testSheet.run.sheetDocuments}
          documents={activeSheet.documents ?? []}
          planDocuments={planDocuments}
          onLink={(documentId) => onModelMutation(() => testSheetApi.linkSheetDocument(activeSheet.id, documentId))}
          onUnlink={(documentId) => onModelMutation(() => testSheetApi.unlinkSheetDocument(activeSheet.id, documentId))}
          onUpload={async (file) => {
            const document = await onModelMutation(() => testSheetApi.uploadDocument(planId, file));
            await onModelMutation(() => testSheetApi.linkSheetDocument(activeSheet.id, document.id));
          }}
          onChanged={async () => {
            await onDocumentsChanged();
            await refreshCurrentSheet();
          }}
        />
      )}
      {canEditSteps && (
        <div className="sheet-steps-panel">
          <div className="section-header compact">
            <div>
              <span className="section-kicker">{messages.testSheet.edit.testSteps}</span>
              <h3>{steps.length} {messages.testSheet.edit.step}{steps.length > 1 ? 's' : ''}</h3>
            </div>
          </div>
          <TestStepList
            steps={steps}
            onEdit={openEditStep}
            onDelete={deleteStep}
            onDuplicate={duplicateStep}
            onMove={moveStep}
            editingStepId={stepEditorMode === 'edit' ? editingStep?.id : undefined}
            renderEditor={(step) => (
              <>
                <TestStepForm
                  ref={stepFormRef}
                  step={step}
                  nextOrder={nextStepOrder}
                  onSubmit={saveStep}
                  onCancel={closeStepEditor}
                />
                <DocumentAssociationPanel
                  title={messages.testSheet.run.actionDocuments}
                  documents={step.documents ?? []}
                  planDocuments={planDocuments}
                  onLink={(documentId) => onModelMutation(() => testSheetApi.linkStepDocument(step.id, documentId))}
                  onUnlink={(documentId) => onModelMutation(() => testSheetApi.unlinkStepDocument(step.id, documentId))}
                  onUpload={async (file) => {
                    const document = await onModelMutation(() => testSheetApi.uploadDocument(planId, file));
                    await onModelMutation(() => testSheetApi.linkStepDocument(step.id, document.id));
                  }}
                  onChanged={async () => {
                    await onDocumentsChanged();
                    await refreshCurrentSheet();
                  }}
                />
              </>
            )}
          />
          {stepEditorMode === 'closed' && (
            <div className="add-sheet-row">
              <Button type="button" onClick={openCreateStep}>+ {messages.testSheet.edit.addStep}</Button>
            </div>
          )}
          {stepEditorMode === 'create' && (
            <TestStepForm
              ref={stepFormRef}
              nextOrder={nextStepOrder}
              onSubmit={saveStep}
              onCancel={closeStepEditor}
            />
          )}
        </div>
      )}
      <div className="button-row end editor-footer">
        <Button type="submit" form={formId}>{isCreate ? messages.testSheet.edit.createSheet : messages.common.save}</Button>
        <Button type="button" variant="secondary" onClick={onCancel}>{messages.common.cancel}</Button>
      </div>
    </Card>
  );
});

function DocumentAssociationPanel({
  title,
  documents,
  planDocuments,
  onLink,
  onUnlink,
  onUpload,
  onChanged,
}: {
  title: string;
  documents: TestDocument[];
  planDocuments: TestDocument[];
  onLink: (documentId: number) => Promise<void>;
  onUnlink: (documentId: number) => Promise<void>;
  onUpload: (file: File) => Promise<void>;
  onChanged: () => Promise<void>;
}) {
  const [selectedDocumentId, setSelectedDocumentId] = useState('');
  const [file, setFile] = useState<File | undefined>();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const fileInputId = React.useId();
  const linkedIds = new Set(documents.map((document) => document.id));
  const availableDocuments = planDocuments.filter((document) => !linkedIds.has(document.id));

  const resetFile = () => {
    setFile(undefined);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <section className="document-association-panel">
      <div className="section-header compact">
        <div>
          <span className="section-kicker">{messages.testSheet.documents.documents}</span>
          <h3>{title}</h3>
        </div>
      </div>
      <DocumentList
        documents={documents}
        emptyText={messages.testSheet.documents.noAssociatedDocument}
        onRemove={async (document) => {
          await onUnlink(document.id);
          await onChanged();
        }}
      />
      <div className="document-upload-row">
        <select value={selectedDocumentId} onChange={(event) => setSelectedDocumentId(event.target.value)} disabled={availableDocuments.length === 0}>
          <option value="">{messages.testSheet.documents.associateExistingDocument}</option>
          {availableDocuments.map((document) => (
            <option key={document.id} value={document.id}>{document.originalName}</option>
          ))}
        </select>
        <Button
          type="button"
          variant="secondary"
          disabled={!selectedDocumentId}
          onClick={async () => {
            await onLink(Number(selectedDocumentId));
            setSelectedDocumentId('');
            await onChanged();
          }}
        >
          {messages.testSheet.documents.associate}
        </Button>
      </div>
      <div className="document-upload-row">
        <DocumentFilePicker
          id={fileInputId}
          file={file}
          inputRef={fileInputRef}
          onFileChange={setFile}
          label={`+ ${messages.testSheet.documents.chooseFile}`}
        />
        <Button
          type="button"
          disabled={!file}
          onClick={async () => {
            if (!file) {
              return;
            }
            await onUpload(file);
            resetFile();
            await onChanged();
          }}
        >
          {messages.testSheet.documents.importAndAssociate}
        </Button>
      </div>
    </section>
  );
}
