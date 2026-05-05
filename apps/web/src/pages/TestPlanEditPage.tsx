import React, { useEffect, useMemo, useState } from 'react';
import { testSheetApi, TestPlan, TestSheet } from '../api/testSheet';
import { TestPlanForm } from '../components/test-sheet/TestPlanForm';
import { TestSheetEditor } from '../components/test-sheet/TestSheetEditor';
import { TestSheetList } from '../components/test-sheet/TestSheetList';
import { Button } from '../components/ui/Button';
import { Card, CardHeader } from '../components/ui/Card';
import { EmptyState } from '../components/ui/EmptyState';
import { PageHeader } from '../components/ui/PageHeader';

type Props = {
  planId: number;
  onBack: () => void;
  onRun: (runId: number) => void;
};

type SheetEditorMode = 'closed' | 'create' | 'edit';

export function TestPlanEditPage({ planId, onBack, onRun }: Props) {
  const [plan, setPlan] = useState<TestPlan | undefined>();
  const [sheets, setSheets] = useState<TestSheet[]>([]);
  const [sheetEditorMode, setSheetEditorMode] = useState<SheetEditorMode>('closed');
  const [editingSheet, setEditingSheet] = useState<TestSheet | undefined>();
  const [error, setError] = useState('');

  const isNew = planId === 0 && !plan;
  const effectivePlanId = plan?.id ?? planId;
  const nextOrder = useMemo(() => Math.max(0, ...sheets.map((sheet) => sheet.executionOrder)) + 1, [sheets]);

  const load = async () => {
    if (isNew) {
      return;
    }
    const [loadedPlan, loadedSheets] = await Promise.all([testSheetApi.getPlan(planId), testSheetApi.listSheets(planId)]);
    setPlan(loadedPlan);
    setSheets(loadedSheets);
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

  const closeEditor = () => {
    setSheetEditorMode('closed');
    setEditingSheet(undefined);
  };

  const savePlan = async (input: { name: string; description: string; mockupSettings: string }) => {
    const saved = isNew ? await testSheetApi.createPlan(input) : await testSheetApi.updatePlan(effectivePlanId, input);
    setPlan(saved);
  };

  const afterSheetSaved = async () => {
    await refreshSheets();
    closeEditor();
  };

  const openCreateSheet = () => {
    setEditingSheet(undefined);
    setSheetEditorMode('create');
  };

  const openEditSheet = (sheet: TestSheet) => {
    setEditingSheet(sheet);
    setSheetEditorMode('edit');
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Edition"
        title={isNew ? 'Nouveau plan' : plan?.name ?? 'Plan de test'}
        description={isNew ? 'Enregistrez le plan avant d ajouter les fiches.' : `${sheets.length} fiche${sheets.length > 1 ? 's' : ''} dans ce plan.`}
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
            Lancer une execution
          </Button>
        )}
      />

      {error && <p className="error">{error}</p>}

      <Card>
        <CardHeader>
          <div>
            <span className="section-kicker">Informations generales</span>
            <h3>Plan</h3>
          </div>
        </CardHeader>
        <TestPlanForm plan={plan} onSubmit={savePlan} />
      </Card>

      <section className="sheet-list-section">
        <div className="section-header">
          <div>
            <span className="section-kicker">Ordre d execution</span>
            <h3>Fiches de test</h3>
          </div>
        </div>

        {!plan ? (
          <EmptyState title="Plan non enregistre" description="Enregistrez le plan avant d ajouter des fiches." />
        ) : (
          <>
            <TestSheetList
              sheets={sheets}
              onEdit={openEditSheet}
              onDelete={async (sheet) => {
                await testSheetApi.deleteSheet(sheet.id);
                await refreshSheets();
                if (editingSheet?.id === sheet.id) {
                  closeEditor();
                }
              }}
              onDuplicate={async (sheet) => {
                await testSheetApi.duplicateSheet(sheet.id);
                await refreshSheets();
              }}
              onMove={async (sheet, direction) => {
                const currentIndex = sheets.findIndex((item) => item.id === sheet.id);
                const next = [...sheets];
                const targetIndex = currentIndex + direction;
                [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];
                await testSheetApi.reorderSheets(effectivePlanId, next.map((item) => item.id));
                await refreshSheets();
              }}
            />

            {sheetEditorMode === 'closed' && (
              <div className="add-sheet-row">
                <Button type="button" onClick={openCreateSheet}>+ Ajouter une fiche</Button>
              </div>
            )}

            {sheetEditorMode !== 'closed' && (
              <TestSheetEditor
                mode={sheetEditorMode === 'create' ? 'create' : 'edit'}
                planId={effectivePlanId}
                sheet={editingSheet}
                nextOrder={nextOrder}
                onCancel={closeEditor}
                onSaved={afterSheetSaved}
                onRefresh={refreshSheets}
              />
            )}
          </>
        )}
      </section>
    </section>
  );
}
