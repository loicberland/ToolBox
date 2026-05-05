import React, { useEffect, useMemo, useState } from 'react';
import { testSheetApi, TestPlan, TestSheet } from '../api/testSheet';
import { TestPlanForm } from '../components/test-sheet/TestPlanForm';
import { TestSheetForm } from '../components/test-sheet/TestSheetForm';
import { TestSheetList } from '../components/test-sheet/TestSheetList';

type Props = {
  planId: number;
  onBack: () => void;
  onRun: (runId: number) => void;
};

export function TestPlanEditPage({ planId, onBack, onRun }: Props) {
  const [plan, setPlan] = useState<TestPlan | undefined>();
  const [sheets, setSheets] = useState<TestSheet[]>([]);
  const [editingSheet, setEditingSheet] = useState<TestSheet | undefined>();
  const [error, setError] = useState('');

  const isNew = planId === 0 && !plan;
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

  const effectivePlanId = plan?.id ?? planId;

  const savePlan = async (input: { name: string; description: string; mockupSettings: string }) => {
    const saved = isNew ? await testSheetApi.createPlan(input) : await testSheetApi.updatePlan(effectivePlanId, input);
    setPlan(saved);
  };

  return (
    <section className="workspace">
      <header className="page-header">
        <div>
          <button className="link-button" type="button" onClick={onBack}>Retour</button>
          <h2>{isNew ? 'Nouveau plan' : plan?.name ?? 'Plan de test'}</h2>
        </div>
        {!isNew && (
          <button
            type="button"
            disabled={sheets.length === 0}
            onClick={async () => {
              const run = await testSheetApi.createRun(planId);
              onRun(run.id);
            }}
          >
            Lancer une execution
          </button>
        )}
      </header>
      {error && <p className="error">{error}</p>}
      <div className="edit-layout">
        <section>
          <h3>Plan</h3>
          <TestPlanForm plan={plan} onSubmit={savePlan} />
        </section>
        <section>
          <h3>Fiches</h3>
          {!plan && <p className="muted">Enregistrez le plan avant d'ajouter des fiches.</p>}
          {plan && (
            <>
              <TestSheetForm
                sheet={editingSheet}
                nextOrder={nextOrder}
                onCancel={() => setEditingSheet(undefined)}
                onSubmit={async (input) => {
                  if (editingSheet) {
                    await testSheetApi.updateSheet(editingSheet.id, input);
                  } else {
                    await testSheetApi.createSheet(effectivePlanId, input);
                  }
                  setEditingSheet(undefined);
                  setSheets(await testSheetApi.listSheets(effectivePlanId));
                }}
              />
              <TestSheetList
                sheets={sheets}
                onEdit={setEditingSheet}
                onDelete={async (sheet) => {
                  await testSheetApi.deleteSheet(sheet.id);
                  setSheets(await testSheetApi.listSheets(effectivePlanId));
                }}
                onDuplicate={async (sheet) => {
                  await testSheetApi.duplicateSheet(sheet.id);
                  setSheets(await testSheetApi.listSheets(effectivePlanId));
                }}
                onMove={async (sheet, direction) => {
                  const currentIndex = sheets.findIndex((item) => item.id === sheet.id);
                  const next = [...sheets];
                  const targetIndex = currentIndex + direction;
                  [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];
                  await testSheetApi.reorderSheets(effectivePlanId, next.map((item) => item.id));
                  setSheets(await testSheetApi.listSheets(effectivePlanId));
                }}
              />
            </>
          )}
        </section>
      </div>
    </section>
  );
}
