import React, { useEffect, useMemo, useState } from 'react';
import { SheetInput, StepInput, testSheetApi, TestSheet, TestSheetStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card, CardHeader } from '../ui/Card';
import { TestSheetForm } from './TestSheetForm';
import { TestStepForm } from './TestStepForm';
import { TestStepList } from './TestStepList';

type Mode = 'create' | 'edit';
type StepEditorMode = 'closed' | 'create' | 'edit';

type DraftStep = TestSheetStep & {
  draftId?: number;
};

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
  onRefresh: () => Promise<TestSheet[]>;
};

export function TestSheetEditor({ mode, planId, sheet, nextOrder, onCancel, onSaved, onRefresh }: Props) {
  const [draftSteps, setDraftSteps] = useState<DraftStep[]>([]);
  const [currentSheet, setCurrentSheet] = useState<TestSheet | undefined>(sheet);
  const [stepEditorMode, setStepEditorMode] = useState<StepEditorMode>('closed');
  const [editingStep, setEditingStep] = useState<DraftStep | undefined>();
  const [draftID, setDraftID] = useState(1);

  const isCreate = mode === 'create';
  const steps = isCreate ? draftSteps : (currentSheet?.steps ?? []);
  const nextStepOrder = useMemo(() => Math.max(0, ...steps.map((step) => step.executionOrder)) + 1, [steps]);

  useEffect(() => {
    setDraftSteps([]);
    setCurrentSheet(sheet);
    closeStepEditor();
    setDraftID(1);
  }, [mode, sheet?.id]);

  useEffect(() => {
    setCurrentSheet(sheet);
  }, [sheet]);

  const openCreateStep = () => {
    setEditingStep(undefined);
    setStepEditorMode('create');
  };

  const openEditStep = (step: DraftStep) => {
    setEditingStep(step);
    setStepEditorMode('edit');
  };

  const closeStepEditor = () => {
    setEditingStep(undefined);
    setStepEditorMode('closed');
  };

  const refreshCurrentSheet = async () => {
    const loadedSheets = await onRefresh();
    if (currentSheet) {
      setCurrentSheet(loadedSheets.find((item) => item.id === currentSheet.id));
    }
  };

  useEffect(() => {
    if (isCreate) {
      return;
    }
    setDraftID(1);
  }, [isCreate]);

  const saveSheet = async (input: SheetInput) => {
    const normalizedInput = {
      ...input,
      mockupSettings: sheet?.mockupSettings ?? input.mockupSettings ?? '',
    };
    if (isCreate) {
      const created = await testSheetApi.createSheet(planId, normalizedInput);
      for (const step of draftSteps) {
        await testSheetApi.createStep(created.id, {
          action: step.action,
          field: step.field,
          expectedResult: step.expectedResult,
          executionOrder: step.executionOrder,
        });
      }
    } else if (sheet) {
      await testSheetApi.updateSheet(sheet.id, normalizedInput);
    }
    await onSaved();
  };

  const saveStep = async (input: StepInput) => {
    if (isCreate) {
      if (editingStep) {
        setDraftSteps((items) => items.map((item) => item.id === editingStep.id ? { ...item, ...input } : item));
      } else {
        const id = -draftID;
        setDraftID((value) => value + 1);
        setDraftSteps((items) => [...items, {
          id,
          draftId: id,
          sheetId: 0,
          action: input.action,
          field: input.field,
          expectedResult: input.expectedResult,
          executionOrder: input.executionOrder,
        }]);
      }
      closeStepEditor();
      return;
    }
    if (!currentSheet) {
      return;
    }
    if (editingStep) {
      await testSheetApi.updateStep(editingStep.id, input);
    } else {
      await testSheetApi.createStep(currentSheet.id, input);
    }
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const deleteStep = async (step: DraftStep) => {
    if (isCreate) {
      setDraftSteps((items) => items.filter((item) => item.id !== step.id));
      return;
    }
    await testSheetApi.deleteStep(step.id);
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const duplicateStep = async (step: DraftStep) => {
    if (isCreate) {
      const id = -draftID;
      setDraftID((value) => value + 1);
      setDraftSteps((items) => [...items, { ...step, id, draftId: id, executionOrder: nextStepOrder }]);
      return;
    }
    await testSheetApi.duplicateStep(step.id);
    await refreshCurrentSheet();
    closeStepEditor();
  };

  const moveStep = async (step: DraftStep, direction: -1 | 1) => {
    const currentIndex = steps.findIndex((item) => item.id === step.id);
    const targetIndex = currentIndex + direction;
    const next = [...steps];
    [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];

    if (isCreate) {
      setDraftSteps(next.map((item, index) => ({ ...item, executionOrder: index + 1 })));
      return;
    }
    if (currentSheet) {
      await testSheetApi.reorderSteps(currentSheet.id, next.map((item) => item.id));
      await refreshCurrentSheet();
      closeStepEditor();
    }
  };

  return (
    <Card className="sheet-editor-card">
      <CardHeader>
        <div>
          <span className="section-kicker">{isCreate ? 'Nouvelle fiche' : 'Modification'}</span>
          <h3>{isCreate ? 'Ajouter une fiche' : currentSheet?.name ?? 'Modifier la fiche'}</h3>
        </div>
      </CardHeader>
      <TestSheetForm sheet={currentSheet ?? { ...emptySheet, executionOrder: nextOrder }} nextOrder={nextOrder} onSubmit={saveSheet} onCancel={onCancel} />
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
        />
        {stepEditorMode === 'closed' && (
          <div className="add-sheet-row">
            <Button type="button" onClick={openCreateStep}>+ Ajouter une etape</Button>
          </div>
        )}
        {stepEditorMode !== 'closed' && (
          <TestStepForm
            step={stepEditorMode === 'edit' ? editingStep : undefined}
            nextOrder={nextStepOrder}
            onSubmit={saveStep}
            onCancel={closeStepEditor}
          />
        )}
      </div>
    </Card>
  );
}
