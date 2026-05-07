import React, { useEffect, useState } from 'react';
import { testSheetApi, TestRun } from '../api/testSheet';
import { TestRunProgress } from '../components/test-sheet/TestRunProgress';
import { TestRunSheetDetail } from '../components/test-sheet/TestRunSheetDetail';
import { TestRunSheetList } from '../components/test-sheet/TestRunSheetList';
import { Button } from '../components/ui/Button';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { PageHeader } from '../components/ui/PageHeader';
import { StatusBadge } from '../components/test-sheet/StatusBadge';
import { getRunSheetProgress, isRunEditable, isRunReadOnly } from '../components/test-sheet/runStatus';
import { messages } from '../i18n';

type Props = {
  runId: number;
  onBack: () => void;
  onReport: (runId: number) => void;
};

export function TestRunPage({ runId, onBack, onReport }: Props) {
  const [run, setRun] = useState<TestRun | undefined>();
  const [error, setError] = useState('');
  const [selectedSheetId, setSelectedSheetId] = useState<number | undefined>();
  const [confirmFinish, setConfirmFinish] = useState(false);

  const load = () => testSheetApi.getRun(runId).then(setRun).catch((err: Error) => setError(err.message));

  useEffect(() => {
    load();
  }, [runId]);

  useEffect(() => {
    if (!run?.sheets.length) {
      setSelectedSheetId(undefined);
      return;
    }

    const selectedSheetStillExists = run.sheets.some((sheet) => sheet.id === selectedSheetId);
    if (selectedSheetStillExists) {
      return;
    }

    const firstPending = run.sheets.find((sheet) => getRunSheetProgress(sheet).status === 'pending');
    setSelectedSheetId((firstPending ?? run.sheets[0]).id);
  }, [run, selectedSheetId]);

  const selectedSheet = run?.sheets.find((sheet) => sheet.id === selectedSheetId);
  const readOnly = run ? isRunReadOnly(run.status) : false;
  const runEditable = run ? isRunEditable(run.status) : false;
  const finish = async () => {
    if (!run || !runEditable) {
      return;
    }
    if (hasPendingWork(run)) {
      setConfirmFinish(true);
      return;
    }
    await testSheetApi.finishRun(runId);
    await load();
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow={messages.testSheet.run.eyebrow}
        title={run ? `${messages.testSheet.plans.executionNumber}${run.runNumber} - ${run.planName}` : messages.testSheet.run.eyebrow}
        description={selectedSheet ? `${messages.testSheet.run.selectedTest} : ${selectedSheet.name}` : undefined}
        onBack={onBack}
        actions={(
          <div className="button-row">
            {run && <StatusBadge status={run.status} />}
            <Button variant="secondary" type="button" onClick={() => onReport(runId)}>{messages.testSheet.run.report}</Button>
            {runEditable && <Button type="button" onClick={finish}>{messages.testSheet.run.finish}</Button>}
          </div>
        )}
      />
      {error && <p className="error">{error}</p>}
      {run && (
        <div className="test-run-execution">
          <div className="test-run-progress">
            <TestRunProgress status={run.status} sheets={run.sheets} />
          </div>
          <div className="test-run-layout">
            <div className="test-run-sidebar">
              <TestRunSheetList sheets={run.sheets} selectedSheetId={selectedSheetId} onSelect={setSelectedSheetId} />
            </div>
            <aside className="test-run-detail">
              {selectedSheet && (
                <TestRunSheetDetail
                  sheet={selectedSheet}
                  readOnly={readOnly}
                  onSaveSheet={async (sheetId, input) => {
                    if (readOnly) {
                      return;
                    }
                    await testSheetApi.updateRunSheet(runId, sheetId, input);
                    await load();
                  }}
                  onSaveStep={async (stepId, input) => {
                    if (readOnly) {
                      return;
                    }
                    await testSheetApi.updateRunStep(runId, stepId, input);
                    await load();
                  }}
                  onEvidenceChanged={load}
                />
              )}
            </aside>
          </div>
        </div>
      )}
      <ConfirmDialog
        open={confirmFinish}
        title={messages.testSheet.run.finishTitle}
        message={messages.testSheet.run.finishMessage}
        confirmLabel={messages.testSheet.run.finishAnyway}
        onCancel={() => setConfirmFinish(false)}
        onConfirm={async () => {
          if (!runEditable) {
            setConfirmFinish(false);
            return;
          }
          await testSheetApi.finishRun(runId);
          setConfirmFinish(false);
          await load();
        }}
      />
    </section>
  );
}

function hasPendingWork(run: TestRun) {
  return run.sheets.some((sheet) => {
    const steps = sheet.steps ?? [];
    if (steps.length === 0) {
      return sheet.status === 'pending';
    }
    return steps.some((step) => step.status === 'pending');
  });
}
