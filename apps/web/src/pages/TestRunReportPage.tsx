import React, { useEffect, useState } from 'react';
import { testSheetApi } from '../api/testSheet';
import { ReportPreview } from '../components/test-sheet/ReportPreview';
import { PageHeader } from '../components/ui/PageHeader';

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
      <PageHeader
        eyebrow="Rapport"
        title="Rapport Markdown"
        description="Version lisible et copiable de l'execution."
        onBack={onBack}
      />
      {error && <p className="error">{error}</p>}
      <ReportPreview markdown={markdown} />
    </section>
  );
}
