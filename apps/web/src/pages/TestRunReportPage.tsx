import React, { useEffect, useState } from 'react';
import { testSheetApi } from '../api/testSheet';
import { ReportPreview } from '../components/test-sheet/ReportPreview';

type Props = {
  runId: number;
  onBack: () => void;
};

export function TestRunReportPage({ runId, onBack }: Props) {
  const [markdown, setMarkdown] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    testSheetApi.getReport(runId).then(setMarkdown).catch((err: Error) => setError(err.message));
  }, [runId]);

  return (
    <section className="workspace">
      <header className="page-header">
        <div>
          <button className="link-button" type="button" onClick={onBack}>Retour</button>
          <h2>Rapport Markdown</h2>
        </div>
      </header>
      {error && <p className="error">{error}</p>}
      <ReportPreview markdown={markdown} />
    </section>
  );
}
