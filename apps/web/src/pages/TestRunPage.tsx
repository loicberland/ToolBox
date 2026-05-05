import React, { useEffect, useState } from 'react';
import { testSheetApi, TestRun } from '../api/testSheet';
import { TestRunSheetCard } from '../components/test-sheet/TestRunSheetCard';
import { TestRunProgress } from '../components/test-sheet/TestRunProgress';
import { Button } from '../components/ui/Button';
import { Card, CardHeader } from '../components/ui/Card';
import { PageHeader } from '../components/ui/PageHeader';
import { StatusBadge } from '../components/test-sheet/StatusBadge';

type Props = {
  runId: number;
  onBack: () => void;
  onReport: (runId: number) => void;
};

export function TestRunPage({ runId, onBack, onReport }: Props) {
  const [run, setRun] = useState<TestRun | undefined>();
  const [error, setError] = useState('');

  const load = () => testSheetApi.getRun(runId).then(setRun).catch((err: Error) => setError(err.message));

  useEffect(() => {
    load();
  }, [runId]);

  const currentSheet = run?.sheets.find((sheet) => sheet.status === 'pending') ?? run?.sheets[0];

  return (
    <section className="workspace">
      <PageHeader
        eyebrow="Execution"
        title={run ? `#${run.id} - ${run.planName}` : 'Execution'}
        description={currentSheet ? `Test courant : ${currentSheet.name}` : undefined}
        onBack={onBack}
        actions={(
          <div className="button-row">
            <Button variant="secondary" type="button" onClick={() => onReport(runId)}>Rapport</Button>
            <Button type="button" onClick={async () => { await testSheetApi.finishRun(runId); load(); }}>Terminer</Button>
          </div>
        )}
      />
      {error && <p className="error">{error}</p>}
      {run && (
        <div className="run-layout">
          <div className="run-main">
            <TestRunProgress status={run.status} sheets={run.sheets} />
            {currentSheet && (
              <TestRunSheetCard
                sheet={currentSheet}
                current
                onSave={async (sheetId, input) => {
                  await testSheetApi.updateRunSheet(runId, sheetId, input);
                  load();
                }}
              />
            )}
          </div>
          <aside className="run-side">
            <Card>
              <CardHeader>
                <div>
                  <span className="section-kicker">Liste des tests</span>
                  <h3>{run.sheets.length} fiches</h3>
                </div>
              </CardHeader>
              <div className="run-sheet-nav">
                {run.sheets.map((sheet) => (
                  <div className={`run-sheet-nav-item ${sheet.id === currentSheet?.id ? 'active' : ''}`} key={sheet.id}>
                    <span>{sheet.executionOrder}. {sheet.name}</span>
                    <StatusBadge status={sheet.status} />
                  </div>
                ))}
              </div>
            </Card>
          </aside>
          <div className="run-all-tests">
            <h3>Toutes les fiches executees</h3>
            <div className="run-list">
              {run.sheets.map((sheet) => (
                <TestRunSheetCard
                  key={sheet.id}
                  sheet={sheet}
                  current={sheet.id === currentSheet?.id}
                  onSave={async (sheetId, input) => {
                    await testSheetApi.updateRunSheet(runId, sheetId, input);
                    load();
                  }}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
