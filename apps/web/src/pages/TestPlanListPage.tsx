import React, { useEffect, useMemo, useState } from 'react';
import { testSheetApi, TestPlanSummary, TestRunSummary, TestRunStatus } from '../api/testSheet';
import { PageHeader } from '../components/ui/PageHeader';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { MarkdownPreview, hasMarkdownContent } from '../components/ui/MarkdownPreview';
import { StatusBadge } from '../components/test-sheet/StatusBadge';

type Props = {
  onEdit: (planId: number) => void;
  onRun: (runId: number) => void;
  onReport: (runId: number) => void;
};

type Filter = 'all' | TestRunStatus;
type SortKey = 'updatedAt' | 'latestRun' | 'status' | 'name';

export function TestPlanListPage({ onEdit, onRun, onReport }: Props) {
  const [plans, setPlans] = useState<TestPlanSummary[]>([]);
  const [runsByPlan, setRunsByPlan] = useState<Record<number, TestRunSummary[]>>({});
  const [historyPlanId, setHistoryPlanId] = useState<number | undefined>();
  const [planToDelete, setPlanToDelete] = useState<TestPlanSummary | undefined>();
  const [runToArchive, setRunToArchive] = useState<TestRunSummary | undefined>();
  const [runConflict, setRunConflict] = useState<{ plan: TestPlanSummary; run: TestRunSummary; replaySourceRunId?: number } | undefined>();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  const [sortKey, setSortKey] = useState<SortKey>('latestRun');
  const [error, setError] = useState('');

  const load = async () => {
    try {
      setPlans(await testSheetApi.listPlanSummaries());
    } catch (err) {
      setError((err as Error).message);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const filteredPlans = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return [...plans]
      .filter((plan) => filter === 'all' || plan.status === filter)
      .filter((plan) => {
        if (!needle) {
          return true;
        }
        return [plan.name, plan.description, plan.latestRun?.planName ?? ''].some((value) => value.toLowerCase().includes(needle));
      })
      .sort((a, b) => comparePlans(a, b, sortKey));
  }, [plans, query, filter, sortKey]);

  const openHistory = async (plan: TestPlanSummary) => {
    if (historyPlanId === plan.id) {
      setHistoryPlanId(undefined);
      return;
    }
    setHistoryPlanId(plan.id);
    if (!runsByPlan[plan.id]) {
      setRunsByPlan((current) => ({ ...current, [plan.id]: [] }));
      const runs = await testSheetApi.listPlanRuns(plan.id);
      setRunsByPlan((current) => ({ ...current, [plan.id]: runs }));
    }
  };

  const createRun = async (plan: TestPlanSummary) => {
    const existingRun = findRunningRun(plan, runsByPlan[plan.id]);
    if (existingRun) {
      setRunConflict({ plan, run: existingRun });
      return;
    }
    const run = await testSheetApi.createRun(plan.id);
    onRun(run.id);
  };

  const replayPlan = async (plan: TestPlanSummary) => {
    const existingRun = findRunningRun(plan, runsByPlan[plan.id]);
    if (existingRun) {
      setRunConflict({ plan, run: existingRun });
      return;
    }
    if (plan.latestRun) {
      const run = await testSheetApi.replayRun(plan.latestRun.id);
      onRun(run.id);
      return;
    }
    await createRun(plan);
  };

  const refreshPlanRuns = async (planId: number) => {
    const runs = await testSheetApi.listPlanRuns(planId);
    setRunsByPlan((current) => ({ ...current, [planId]: runs }));
    await load();
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Test Sheet"
        title="Plans de test"
        description="Suivi des modeles et historique des executions."
        actions={<Button type="button" onClick={() => onEdit(0)}>Nouveau plan</Button>}
      />
      {error && <p className="error">{error}</p>}

      <div className="plan-toolbar">
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Rechercher un plan" />
        <select value={filter} onChange={(event) => setFilter(event.target.value as Filter)}>
          <option value="all">Tous</option>
          <option value="pending">En attente</option>
          <option value="running">En cours</option>
          <option value="completed">Termines</option>
          <option value="canceled">Annules</option>
          <option value="archived">Archives</option>
        </select>
        <select value={sortKey} onChange={(event) => setSortKey(event.target.value as SortKey)}>
          <option value="latestRun">Derniere execution</option>
          <option value="updatedAt">Derniere modification</option>
          <option value="status">Statut</option>
          <option value="name">Nom</option>
        </select>
      </div>

      <div className="plan-summary-list">
        {filteredPlans.map((plan) => (
          <Card className="test-plan-summary-card" key={plan.id}>
            <div className="plan-summary-header">
              <div>
                <div className="card-topline">
                  <StatusBadge status={plan.status} />
                  <span className="muted">{plan.sheetCount} fiche{plan.sheetCount > 1 ? 's' : ''}</span>
                  <span className="muted">{plan.runCount} execution{plan.runCount > 1 ? 's' : ''}</span>
                </div>
                <h3>{plan.name}</h3>
              </div>
              <div className="button-row end">
                {plan.latestRun?.status === 'running' && <Button type="button" onClick={() => onRun(plan.latestRun!.id)}>Continuer</Button>}
                {plan.latestRun && plan.latestRun.status !== 'running' && <Button type="button" onClick={() => onRun(plan.latestRun!.id)}>Ouvrir</Button>}
                <Button type="button" variant="secondary" disabled={plan.sheetCount === 0 && !plan.latestRun} onClick={() => replayPlan(plan)}>
                  {plan.latestRun ? 'Rejouer' : 'Nouvelle execution'}
                </Button>
                <Button type="button" variant="secondary" onClick={() => openHistory(plan)}>Historique</Button>
                <Button type="button" variant="secondary" onClick={() => onEdit(plan.id)}>Modifier le modele</Button>
                <Button type="button" variant="secondary" onClick={async () => { await testSheetApi.duplicatePlan(plan.id); await load(); }}>Dupliquer</Button>
                <Button type="button" variant="danger" onClick={() => setPlanToDelete(plan)}>Supprimer</Button>
              </div>
            </div>

            {hasMarkdownContent(plan.description) && <MarkdownPreview content={plan.description} compact />}
            <PlanRunProgress run={plan.latestRun} />
            <div className="card-meta">
              <span>Mis a jour</span>
              <strong>{formatDate(plan.updatedAt)}</strong>
              <span>Derniere execution</span>
              <strong>{plan.latestRun ? formatDate(plan.latestRun.startedAt) : '-'}</strong>
            </div>

            {historyPlanId === plan.id && (
              <div className="run-history-list">
                {(runsByPlan[plan.id] ?? []).filter((run) => run.id !== plan.latestRun?.id).map((run) => (
                  <div className="run-history-item" key={run.id}>
                    <div>
                      <div className="card-topline">
                        <StatusBadge status={run.status} />
                        <strong>Execution #{run.id}</strong>
                      </div>
                      <p className="muted">Debut : {formatDate(run.startedAt)}{run.finishedAt ? ` - Fin : ${formatDate(run.finishedAt)}` : ''}</p>
                      <PlanRunProgress run={run} compact />
                    </div>
                    <div className="button-row end">
                      <Button type="button" variant="secondary" onClick={() => onRun(run.id)}>{run.status === 'running' ? 'Continuer' : 'Ouvrir'}</Button>
                      <Button type="button" variant="secondary" onClick={() => onReport(run.id)}>Rapport</Button>
                      <Button type="button" variant="secondary" onClick={async () => {
                        const runningRun = findRunningRun(plan, runsByPlan[plan.id]);
                        if (runningRun) {
                          setRunConflict({ plan, run: runningRun, replaySourceRunId: run.id });
                          return;
                        }
                        const replay = await testSheetApi.replayRun(run.id);
                        onRun(replay.id);
                      }}>Rejouer</Button>
                      {run.status !== 'archived' && <Button type="button" variant="secondary" onClick={() => setRunToArchive(run)}>Archiver</Button>}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        ))}
      </div>

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
            await load();
          }
        }}
      />

      <ConfirmDialog
        open={Boolean(runToArchive)}
        title="Archiver l execution"
        message={`Archiver l execution #${runToArchive?.id ?? ''} ? Elle restera consultable dans l historique.`}
        confirmLabel="Archiver"
        onCancel={() => setRunToArchive(undefined)}
        onConfirm={async () => {
          if (runToArchive) {
            await testSheetApi.archiveRun(runToArchive.id);
            await refreshPlanRuns(runToArchive.planId);
            setRunToArchive(undefined);
          }
        }}
      />

      {runConflict && (
        <div className="dialog-backdrop" role="presentation">
          <div className="confirm-dialog" role="dialog" aria-modal="true">
            <h3>Execution deja en cours</h3>
            <p>Une execution est deja en cours pour ce plan. Voulez-vous la continuer ou l annuler et recommencer ?</p>
            <div className="button-row end">
              <Button type="button" variant="secondary" onClick={() => setRunConflict(undefined)}>Fermer</Button>
              <Button type="button" variant="secondary" onClick={() => { onRun(runConflict.run.id); }}>Continuer</Button>
              <Button type="button" onClick={async () => {
                await testSheetApi.cancelRun(runConflict.run.id);
                const run = runConflict.replaySourceRunId
                  ? await testSheetApi.replayRun(runConflict.replaySourceRunId)
                  : await testSheetApi.createRun(runConflict.plan.id);
                setRunConflict(undefined);
                onRun(run.id);
              }}>Annuler et rejouer</Button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function PlanRunProgress({ run, compact = false }: { run?: TestRunSummary; compact?: boolean }) {
  if (!run) {
    return <p className="muted">Aucune execution</p>;
  }
  const done = run.totalSteps - run.pendingSteps;
  const percent = run.totalSteps === 0 ? 0 : Math.round((done / run.totalSteps) * 100);
  return (
    <div className={compact ? 'plan-run-progress compact' : 'plan-run-progress'}>
      <strong>{done} / {run.totalSteps} actions traitees</strong>
      <div className="progress-track" aria-label={`Progression ${percent}%`}>
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
      <p className="muted">
        {run.passedSteps} reussies - {run.failedSteps} echouees - {run.blockedSteps} bloquees - {run.skippedSteps} ignorees - {run.pendingSteps} en attente
      </p>
    </div>
  );
}

function comparePlans(a: TestPlanSummary, b: TestPlanSummary, sortKey: SortKey) {
  if (sortKey === 'name') {
    return a.name.localeCompare(b.name);
  }
  if (sortKey === 'status') {
    return a.status.localeCompare(b.status) || a.name.localeCompare(b.name);
  }
  if (sortKey === 'updatedAt') {
    return dateValue(b.updatedAt) - dateValue(a.updatedAt);
  }
  return dateValue(b.latestRun?.startedAt) - dateValue(a.latestRun?.startedAt);
}

function findRunningRun(plan: TestPlanSummary, runs?: TestRunSummary[]) {
  return (runs ?? (plan.latestRun ? [plan.latestRun] : [])).find((run) => run.status === 'running');
}

function dateValue(value?: string) {
  return value ? new Date(value).getTime() : 0;
}

function formatDate(value?: string) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  return `${date.toLocaleDateString('fr-FR')} ${date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}`;
}
