import React, { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { SheetInput, StepInput, testSheetApi, TestSheet, TestSheetStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card, CardHeader } from '../ui/Card';
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
};

export type TestSheetEditorHandle = {
  submit: () => Promise<void>;
};

export const TestSheetEditor = forwardRef<TestSheetEditorHandle, Props>(function TestSheetEditor({ mode, planId, sheet, nextOrder, onCancel, onSaved, onCreated, onRefresh, onModelMutation }, ref) {
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
          <span className="section-kicker">{isCreate ? 'Nouvelle fiche' : 'Modification'}</span>
          <h3>{isCreate ? 'Ajouter une fiche' : activeSheet?.name ?? 'Modifier la fiche'}</h3>
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
      {canEditSteps && (
        <div className="sheet-steps-panel">
          <div className="section-header compact">
            <div>
              <span className="section-kicker">Etapes de test</span>
              <h3>{steps.length} etape{steps.length > 1 ? 's' : ''}</h3>
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
              <TestStepForm
                ref={stepFormRef}
                step={step}
                nextOrder={nextStepOrder}
                onSubmit={saveStep}
                onCancel={closeStepEditor}
              />
            )}
          />
          {stepEditorMode === 'closed' && (
            <div className="add-sheet-row">
              <Button type="button" onClick={openCreateStep}>+ Ajouter une etape</Button>
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
        <Button type="submit" form={formId}>{isCreate ? 'Creer la fiche' : 'Enregistrer'}</Button>
        <Button type="button" variant="secondary" onClick={onCancel}>Annuler</Button>
      </div>
    </Card>
  );
});
