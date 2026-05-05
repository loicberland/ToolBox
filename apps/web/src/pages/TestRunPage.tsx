import React, { useEffect, useState } from 'react';
import { testSheetApi, TestRun } from '../api/testSheet';
import { TestRunSheetCard } from '../components/test-sheet/TestRunSheetCard';

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

  return (
    <section className="workspace">
      <header className="page-header">
        <div>
          <button className="link-button" type="button" onClick={onBack}>Retour</button>
          <h2>{run ? `Execution #${run.id} - ${run.planName}` : 'Execution'}</h2>
        </div>
        <div className="button-row">
          <button className="secondary" type="button" onClick={() => onReport(runId)}>Rapport</button>
          <button type="button" onClick={async () => { await testSheetApi.finishRun(runId); load(); }}>Terminer</button>
        </div>
      </header>
      {error && <p className="error">{error}</p>}
      <div className="run-list">
        {run?.sheets.map((sheet) => (
          <TestRunSheetCard
            key={sheet.id}
            sheet={sheet}
            onSave={async (sheetId, input) => {
              await testSheetApi.updateRunSheet(runId, sheetId, input);
              load();
            }}
          />
        ))}
      </div>
    </section>
  );
}
