import React, { useEffect, useState } from 'react';
import { testSheetApi, TestRun } from '../api/testSheet';
import { TestRunProgress } from '../components/test-sheet/TestRunProgress';
import { TestRunSheetDetail } from '../components/test-sheet/TestRunSheetDetail';
import { TestRunSheetList } from '../components/test-sheet/TestRunSheetList';
import { Button } from '../components/ui/Button';
import { PageHeader } from '../components/ui/PageHeader';
import { StatusBadge } from '../components/test-sheet/StatusBadge';
import { getRunSheetProgress } from '../components/test-sheet/runStatus';

type Props = {
  runId: number;
  onBack: () => void;
  onReport: (runId: number) => void;
};

export function TestRunPage({ runId, onBack, onReport }: Props) {
  const [run, setRun] = useState<TestRun | undefined>();
  const [error, setError] = useState('');
  const [selectedSheetId, setSelectedSheetId] = useState<number | undefined>();

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
  const runFinished = run ? ['completed', 'finished'].includes(run.status) : false;

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Execution"
        title={run ? `#${run.id} - ${run.planName}` : 'Execution'}
        description={selectedSheet ? `Test selectionne : ${selectedSheet.name}` : undefined}
        onBack={onBack}
        actions={(
          <div className="button-row">
            {run && <StatusBadge status={run.status} />}
            <Button variant="secondary" type="button" onClick={() => onReport(runId)}>Rapport</Button>
            {run && !runFinished && <Button type="button" onClick={async () => { await testSheetApi.finishRun(runId); await load(); }}>Terminer</Button>}
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
                  onSaveSheet={async (sheetId, input) => {
                    await testSheetApi.updateRunSheet(runId, sheetId, input);
                    await load();
                  }}
                  onSaveStep={async (stepId, input) => {
                    await testSheetApi.updateRunStep(runId, stepId, input);
                    await load();
                  }}
                />
              )}
            </aside>
          </div>
        </div>
      )}
    </section>
  );
}
