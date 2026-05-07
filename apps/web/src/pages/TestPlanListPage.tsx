import React, { useEffect, useMemo, useState } from 'react';
import { testSheetApi, TestPlanSummary, TestRunSummary, TestRunStatus } from '../api/testSheet';
import { PageHeader } from '../components/ui/PageHeader';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { MarkdownPreview, hasMarkdownContent } from '../components/ui/MarkdownPreview';
import { StatusBadge } from '../components/test-sheet/StatusBadge';
import { messages, statusLabel } from '../i18n';

type Props = {
  onEdit: (planId: number) => void;
  onRun: (runId: number) => void;
  onReport: (runId: number) => void;
};

type SortKey = 'updatedAt' | 'latestRun' | 'status' | 'name';
const statusFilterOptions: Array<{ status: TestRunStatus | 'pending'; label: string }> = [
  { status: 'pending', label: statusLabel('pending') },
  { status: 'running', label: statusLabel('running') },
  { status: 'completed', label: statusLabel('completed') },
  { status: 'canceled', label: statusLabel('canceled') },
];

export function TestPlanListPage({ onEdit, onRun, onReport }: Props) {
  const [plans, setPlans] = useState<TestPlanSummary[]>([]);
  const [runsByPlan, setRunsByPlan] = useState<Record<number, TestRunSummary[]>>({});
  const [historyPlanId, setHistoryPlanId] = useState<number | undefined>();
  const [planToDelete, setPlanToDelete] = useState<TestPlanSummary | undefined>();
  const [planToPermanentDelete, setPlanToPermanentDelete] = useState<TestPlanSummary | undefined>();
  const [permanentDeleteConfirmation, setPermanentDeleteConfirmation] = useState('');
  const [info, setInfo] = useState('');
  const [runConflict, setRunConflict] = useState<{ plan: TestPlanSummary; run: TestRunSummary } | undefined>();
  const [query, setQuery] = useState('');
  const [statusFilters, setStatusFilters] = useState<Array<TestRunStatus | 'pending'>>([]);
  const [showDeletedPlans, setShowDeletedPlans] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>('latestRun');
  const [error, setError] = useState('');

  const load = async () => {
    try {
      setPlans(await testSheetApi.listPlanSummaries(showDeletedPlans));
    } catch (err) {
      setError((err as Error).message);
    }
  };

  useEffect(() => {
    load();
  }, [showDeletedPlans]);

  const filteredPlans = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return [...plans]
      .filter((plan) => statusFilters.length === 0 || statusFilters.includes(plan.status as TestRunStatus | 'pending'))
      .filter((plan) => {
        if (!needle) {
          return true;
        }
        return [plan.name, plan.description, plan.latestRun?.planName ?? ''].some((value) => value.toLowerCase().includes(needle));
      })
      .sort((a, b) => comparePlans(a, b, sortKey));
  }, [plans, query, statusFilters, sortKey]);

  const toggleStatusFilter = (status: TestRunStatus | 'pending') => {
    setStatusFilters((current) => current.includes(status)
      ? current.filter((item) => item !== status)
      : [...current, status]);
  };

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

  const closePermanentDeleteDialog = () => {
    setPlanToPermanentDelete(undefined);
    setPermanentDeleteConfirmation('');
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Test Sheet"
        title={messages.testSheet.plans.title}
        description={messages.testSheet.plans.description}
        actions={<Button type="button" onClick={() => onEdit(0)}>{messages.testSheet.plans.newPlan}</Button>}
      />
      {error && <p className="error">{error}</p>}
      {info && <p className="info-message">{info}</p>}

      <div className="plan-toolbar">
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={messages.testSheet.plans.search} />
        <select value={sortKey} onChange={(event) => setSortKey(event.target.value as SortKey)}>
          <option value="latestRun">{messages.testSheet.plans.latestRun}</option>
          <option value="updatedAt">{messages.testSheet.plans.latestUpdate}</option>
          <option value="status">Statut</option>
          <option value="name">Nom</option>
        </select>
      </div>

      <div className="plan-filter-panel">
        <div className="filter-group" aria-label="Filtrer par statut">
          <span className="filter-label">{messages.testSheet.plans.statusFilters}</span>
          {statusFilterOptions.map((option) => (
            <label key={option.status} className="checkbox-filter">
              <input
                type="checkbox"
                checked={statusFilters.includes(option.status)}
                onChange={() => toggleStatusFilter(option.status)}
              />
              {option.label}
            </label>
          ))}
        </div>
        <label className="checkbox-filter">
          <input
            type="checkbox"
            checked={showDeletedPlans}
            onChange={(event) => setShowDeletedPlans(event.target.checked)}
          />
          {messages.testSheet.plans.showHiddenPlans}
        </label>
      </div>

      <div className="plan-summary-list">
        {filteredPlans.map((plan) => (
          <Card className="test-plan-summary-card" key={plan.id}>
            <div className="plan-summary-header">
              <div>
                <div className="card-topline">
                  <StatusBadge status={plan.status} />
                  {plan.deletedAt && <span className="soft-delete-badge">{messages.testSheet.plans.hidden}</span>}
                  <span className="muted">{plan.groupCount} {messages.testSheet.plans.groupSingular}{plan.groupCount > 1 ? 's' : ''}</span>
                  <span className="muted">{plan.sheetCount} {messages.testSheet.plans.sheetSingular}{plan.sheetCount > 1 ? 's' : ''}</span>
                  <span className="muted">{plan.runCount} {messages.testSheet.plans.runSingular}{plan.runCount > 1 ? 's' : ''}</span>
                </div>
                <h3>{plan.name}</h3>
              </div>
              <div className="plan-actions">
                {!plan.deletedAt && (
                  <>
                    {plan.latestRun?.status === 'running' && <Button type="button" size="sm" variant="primary" onClick={() => onRun(plan.latestRun!.id)}>{messages.testSheet.plans.continue}</Button>}
                    {plan.latestRun && plan.latestRun.status !== 'running' && <Button type="button" size="sm" variant="primary" onClick={() => onRun(plan.latestRun!.id)}>{messages.testSheet.plans.open}</Button>}
                    <Button type="button" size="sm" variant="primary" disabled={plan.sheetCount === 0} onClick={() => createRun(plan)}>{messages.testSheet.plans.newRunShort}</Button>
                  </>
                )}
                <Button type="button" size="sm" variant="secondary" onClick={() => openHistory(plan)}>{messages.testSheet.plans.history}</Button>
                {plan.deletedAt ? (
                  <>
                    <Button type="button" size="sm" variant="secondary" onClick={async () => { await testSheetApi.restorePlan(plan.id); await load(); }}>{messages.common.restore}</Button>
                    <Button type="button" size="sm" variant="danger" onClick={() => setPlanToPermanentDelete(plan)}>{messages.testSheet.plans.permanentDelete}</Button>
                  </>
                ) : (
                  <>
                    <Button type="button" size="sm" variant="secondary" onClick={() => onEdit(plan.id)}>{messages.common.edit}</Button>
                    <Button type="button" size="sm" variant="secondary" onClick={async () => { await testSheetApi.duplicatePlan(plan.id); await load(); }}>{messages.testSheet.plans.duplicate}</Button>
                    <Button type="button" size="sm" variant="warning" onClick={() => setPlanToDelete(plan)}>{messages.testSheet.plans.hide}</Button>
                  </>
                )}
              </div>
            </div>

            {hasMarkdownContent(plan.description) && <MarkdownPreview content={plan.description} compact />}
            <PlanRunProgress run={plan.latestRun} />
            <div className="card-meta">
              <span>{messages.testSheet.plans.updatedAt}</span>
              <strong>{formatDate(plan.updatedAt)}</strong>
              <span>{messages.testSheet.plans.latestRun}</span>
              <strong>{plan.latestRun ? formatDate(plan.latestRun.startedAt) : '-'}</strong>
            </div>

            {historyPlanId === plan.id && (
              <div className="run-history-list">
                {(runsByPlan[plan.id] ?? []).filter((run) => run.id !== plan.latestRun?.id).map((run) => (
                  <div className="run-history-item" key={run.id}>
                    <div>
                      <div className="card-topline">
                        <StatusBadge status={run.status} />
                        <strong>{messages.testSheet.plans.executionNumber}{run.runNumber}</strong>
                      </div>
                      <p className="muted">{messages.testSheet.plans.begin} : {formatDate(run.startedAt)}{run.finishedAt ? ` - ${messages.testSheet.plans.end} : ${formatDate(run.finishedAt)}` : ''}</p>
                      <PlanRunProgress run={run} compact />
                    </div>
                    <div className="button-row end">
                      <Button type="button" variant="secondary" onClick={() => onRun(run.id)}>{run.status === 'running' ? messages.testSheet.plans.continue : messages.testSheet.plans.consult}</Button>
                      <Button type="button" variant="secondary" onClick={() => onReport(run.id)}>{messages.testSheet.run.report}</Button>
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
        title={messages.testSheet.dialogs.hidePlanTitle}
        message={`"${planToDelete?.name ?? ''}" ${messages.testSheet.dialogs.hidePlanMessage}`}
        confirmLabel={messages.testSheet.dialogs.hidePlanConfirm}
        onCancel={() => setPlanToDelete(undefined)}
        onConfirm={async () => {
          if (planToDelete) {
            await testSheetApi.deletePlan(planToDelete.id);
            setPlanToDelete(undefined);
            await load();
          }
        }}
      />

      {planToPermanentDelete && (
        <div className="dialog-backdrop" role="presentation">
          <div className="confirm-dialog permanent-delete-dialog" role="dialog" aria-modal="true" aria-labelledby="permanent-delete-title">
            <h3 id="permanent-delete-title">{messages.testSheet.dialogs.deletePlanTitle}</h3>
            <p>{messages.testSheet.dialogs.deletePlanWarning}</p>
            <p>{messages.testSheet.dialogs.deletePlanNoUndo}</p>
            <label>
              {messages.testSheet.dialogs.deletePlanConfirmation} <strong>{planToPermanentDelete.name}</strong>
              <input
                value={permanentDeleteConfirmation}
                onChange={(event) => setPermanentDeleteConfirmation(event.target.value)}
                autoFocus
              />
            </label>
            <div className="button-row end">
              <Button type="button" variant="secondary" onClick={closePermanentDeleteDialog}>{messages.common.cancel}</Button>
              <Button
                type="button"
                variant="danger"
                disabled={permanentDeleteConfirmation !== planToPermanentDelete.name}
                onClick={async () => {
                  await testSheetApi.permanentDeletePlan(planToPermanentDelete.id);
                  closePermanentDeleteDialog();
                  setInfo(messages.testSheet.dialogs.deletePlanDone);
                  await load();
                }}
              >
                {messages.testSheet.plans.permanentDelete}
              </Button>
            </div>
          </div>
        </div>
      )}

      {runConflict && (
        <div className="dialog-backdrop" role="presentation">
          <div className="confirm-dialog" role="dialog" aria-modal="true">
            <h3>{messages.testSheet.dialogs.runConflictTitle}</h3>
            <p>{messages.testSheet.dialogs.runConflictMessage}</p>
            <div className="button-row end">
              <Button type="button" variant="secondary" onClick={() => setRunConflict(undefined)}>{messages.common.close}</Button>
              <Button type="button" variant="secondary" onClick={() => { onRun(runConflict.run.id); }}>{messages.testSheet.plans.continue}</Button>
              <Button type="button" onClick={async () => {
                await testSheetApi.cancelRun(runConflict.run.id);
                const run = await testSheetApi.createRun(runConflict.plan.id);
                setRunConflict(undefined);
                onRun(run.id);
              }}>{messages.testSheet.dialogs.cancelAndReplay}</Button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function PlanRunProgress({ run, compact = false }: { run?: TestRunSummary; compact?: boolean }) {
  if (!run) {
    return <p className="muted">{messages.testSheet.plans.noRun}</p>;
  }
  const totalGroups = run.totalGroups || (run.totalSheets > 0 ? 1 : 0);
  const pendingGroups = run.totalGroups ? run.pendingGroups : (run.pendingSteps > 0 ? 1 : 0);
  const passedGroups = run.totalGroups ? run.passedGroups : (run.pendingSteps === 0 && run.failedSteps === 0 && run.blockedSteps === 0 && run.skippedSteps !== run.totalSteps ? 1 : 0);
  const failedGroups = run.totalGroups ? run.failedGroups : (run.failedSteps > 0 ? 1 : 0);
  const blockedGroups = run.totalGroups ? run.blockedGroups : (run.failedSteps === 0 && run.blockedSteps > 0 ? 1 : 0);
  const skippedGroups = run.totalGroups ? run.skippedGroups : (run.totalSteps > 0 && run.skippedSteps === run.totalSteps ? 1 : 0);
  const done = totalGroups - pendingGroups;
  const percent = totalGroups === 0 ? 0 : Math.round((done / totalGroups) * 100);
  return (
    <div className={compact ? 'plan-run-progress compact' : 'plan-run-progress'}>
      {!compact && <strong>{messages.testSheet.plans.executionNumber}{run.runNumber}</strong>}
      <strong>{done} / {totalGroups} {messages.testSheet.run.subPlansProcessed}</strong>
      <div className="progress-track" aria-label={`Progression ${percent}%`}>
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
      <p className="muted">
        {passedGroups} {messages.testSheet.run.passedPlural} - {failedGroups} {messages.testSheet.run.failedPlural} - {blockedGroups} {messages.testSheet.run.blockedPlural} - {skippedGroups} {messages.testSheet.run.skippedPlural} - {pendingGroups} {statusLabel('pending').toLowerCase()}
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

