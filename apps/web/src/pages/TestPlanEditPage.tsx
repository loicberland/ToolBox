import React, { useEffect, useMemo, useState } from 'react';
import { testSheetApi, TestPlan, TestSheet, TestSheetStep } from '../api/testSheet';
import { TestPlanForm } from '../components/test-sheet/TestPlanForm';
import { TestSheetForm } from '../components/test-sheet/TestSheetForm';
import { TestSheetList } from '../components/test-sheet/TestSheetList';
import { TestStepForm } from '../components/test-sheet/TestStepForm';
import { TestStepList } from '../components/test-sheet/TestStepList';
import { Button } from '../components/ui/Button';
import { Card, CardHeader } from '../components/ui/Card';
import { EmptyState } from '../components/ui/EmptyState';
import { PageHeader } from '../components/ui/PageHeader';

type Props = {
  planId: number;
  onBack: () => void;
  onRun: (runId: number) => void;
};

export function TestPlanEditPage({ planId, onBack, onRun }: Props) {
  const [plan, setPlan] = useState<TestPlan | undefined>();
  const [sheets, setSheets] = useState<TestSheet[]>([]);
  const [editingSheet, setEditingSheet] = useState<TestSheet | undefined>();
  const [editingStep, setEditingStep] = useState<TestSheetStep | undefined>();
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

  const refreshSheets = async () => {
    const loadedSheets = await testSheetApi.listSheets(effectivePlanId);
    setSheets(loadedSheets);
    if (editingSheet) {
      setEditingSheet(loadedSheets.find((sheet) => sheet.id === editingSheet.id));
    }
    return loadedSheets;
  };

  const selectedSteps = editingSheet?.steps ?? [];
  const nextStepOrder = Math.max(0, ...selectedSteps.map((step) => step.executionOrder)) + 1;

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Edition"
        title={isNew ? 'Nouveau plan' : plan?.name ?? 'Plan de test'}
        description={isNew ? 'Structurez le plan avant de creer les fiches.' : `${sheets.length} fiche${sheets.length > 1 ? 's' : ''} dans ce plan.`}
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
      <div className="edit-layout">
        <div className="edit-main-column">
          <Card>
            <CardHeader>
              <div>
                <span className="section-kicker">Informations generales</span>
                <h3>Plan</h3>
              </div>
            </CardHeader>
          <TestPlanForm plan={plan} onSubmit={savePlan} />
          </Card>

          <Card>
            <CardHeader>
              <div>
                <span className="section-kicker">Fiches de test</span>
                <h3>{editingSheet ? 'Modifier une fiche' : 'Ajouter une fiche'}</h3>
              </div>
            </CardHeader>
            {!plan && <EmptyState title="Plan non enregistre" description="Enregistrez le plan avant d'ajouter des fiches." />}
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
                    const created = await testSheetApi.createSheet(effectivePlanId, input);
                    setEditingSheet(created);
                  }
                  await refreshSheets();
                }}
              />
              {editingSheet && (
                <div className="sheet-steps-panel">
                  <div className="section-header compact">
                    <div>
                      <span className="section-kicker">Etapes</span>
                      <h3>{editingSheet.name}</h3>
                    </div>
                  </div>
                  <TestStepForm
                    step={editingStep}
                    nextOrder={nextStepOrder}
                    onCancel={() => setEditingStep(undefined)}
                    onSubmit={async (input) => {
                      if (editingStep) {
                        await testSheetApi.updateStep(editingStep.id, input);
                      } else {
                        await testSheetApi.createStep(editingSheet.id, input);
                      }
                      setEditingStep(undefined);
                      await refreshSheets();
                    }}
                  />
                  <TestStepList
                    steps={selectedSteps}
                    onEdit={setEditingStep}
                    onDelete={async (step) => {
                      await testSheetApi.deleteStep(step.id);
                      await refreshSheets();
                    }}
                    onDuplicate={async (step) => {
                      await testSheetApi.duplicateStep(step.id);
                      await refreshSheets();
                    }}
                    onMove={async (step, direction) => {
                      const currentIndex = selectedSteps.findIndex((item) => item.id === step.id);
                      const next = [...selectedSteps];
                      const targetIndex = currentIndex + direction;
                      [next[currentIndex], next[targetIndex]] = [next[targetIndex], next[currentIndex]];
                      await testSheetApi.reorderSteps(editingSheet.id, next.map((item) => item.id));
                      await refreshSheets();
                    }}
                  />
                </div>
              )}
            </>
          )}
          </Card>
        </div>

        <aside className="edit-side-column">
          <Card>
            <CardHeader>
              <div>
                <span className="section-kicker">Prerequis</span>
                <h3>Couverture</h3>
              </div>
            </CardHeader>
            <div className="metric-list">
              <div><span>Fiches avec prerequis</span><strong>{sheets.filter((sheet) => sheet.prerequisites).length}</strong></div>
              <div><span>Etapes de test</span><strong>{sheets.reduce((total, sheet) => total + (sheet.steps?.length ?? 0), 0)}</strong></div>
              <div><span>Total fiches</span><strong>{sheets.length}</strong></div>
            </div>
          </Card>

          <Card>
            <CardHeader>
              <div>
                <span className="section-kicker">Documents</span>
                <h3>Pieces jointes</h3>
              </div>
            </CardHeader>
            <div className="document-dropzone">
              <strong>Upload a venir</strong>
              <span>Stockage prepare cote base</span>
            </div>
          </Card>
        </aside>
      </div>

      <section className="sheet-list-section">
        <div className="section-header">
          <div>
            <span className="section-kicker">Ordre d'execution</span>
            <h3>Fiches de test</h3>
          </div>
        </div>
        <TestSheetList
          sheets={sheets}
          onEdit={(sheet) => {
            setEditingSheet(sheet);
            setEditingStep(undefined);
          }}
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
      </section>
    </section>
  );
}
