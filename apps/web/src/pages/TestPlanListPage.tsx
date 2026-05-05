import React, { useEffect, useState } from 'react';
import { testSheetApi, TestPlan } from '../api/testSheet';
import { EmptyState } from '../components/ui/EmptyState';
import { PageHeader } from '../components/ui/PageHeader';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { Button } from '../components/ui/Button';
import { TestPlanCard } from '../components/test-sheet/TestPlanCard';

type Props = {
  onEdit: (planId: number) => void;
  onRun: (runId: number) => void;
};

export function TestPlanListPage({ onEdit, onRun }: Props) {
  const [plans, setPlans] = useState<TestPlan[]>([]);
  const [sheetCounts, setSheetCounts] = useState<Record<number, number>>({});
  const [planToDelete, setPlanToDelete] = useState<TestPlan | undefined>();
  const [error, setError] = useState('');

  const load = async () => {
    try {
      const loadedPlans = await testSheetApi.listPlans();
      setPlans(loadedPlans);
      const counts = await Promise.all(
        loadedPlans.map(async (plan) => [plan.id, (await testSheetApi.listSheets(plan.id)).length] as const),
      );
      setSheetCounts(counts.reduce<Record<number, number>>((acc, [planId, count]) => {
        acc[planId] = count;
        return acc;
      }, {}));
    } catch (err) {
      setError((err as Error).message);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Test Sheet"
        title="Plans de test"
        description="Preparation et execution des Product Reviews."
        actions={<Button type="button" onClick={() => onEdit(0)}>Nouveau plan</Button>}
      />
      {error && <p className="error">{error}</p>}
      {plans.length === 0 ? (
        <EmptyState title="Aucun plan" description="Creez un plan pour preparer une Product Review." actionLabel="Nouveau plan" onAction={() => onEdit(0)} />
      ) : (
        <div className="plan-grid">
          {plans.map((plan) => (
            <TestPlanCard
              key={plan.id}
              plan={plan}
              sheetCount={sheetCounts[plan.id] ?? 0}
              onEdit={() => onEdit(plan.id)}
              onRun={async () => {
                const run = await testSheetApi.createRun(plan.id);
                onRun(run.id);
              }}
              onDuplicate={async () => {
                await testSheetApi.duplicatePlan(plan.id);
                load();
              }}
              onDelete={() => setPlanToDelete(plan)}
            />
          ))}
        </div>
      )}
      <ConfirmDialog
        open={Boolean(planToDelete)}
        title="Supprimer le plan"
        message={`Supprimer "${planToDelete?.name ?? ''}" ?`}
        confirmLabel="Supprimer"
        onCancel={() => setPlanToDelete(undefined)}
        onConfirm={async () => {
          if (planToDelete) {
            await testSheetApi.deletePlan(planToDelete.id);
            setPlanToDelete(undefined);
            load();
          }
        }}
      />
    </section>
  );
}
