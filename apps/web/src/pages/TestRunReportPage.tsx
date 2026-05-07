import React, { useEffect, useState } from 'react';
import { testSheetApi } from '../api/testSheet';
import { ReportPreview } from '../components/test-sheet/ReportPreview';
import { PageHeader } from '../components/ui/PageHeader';
import { messages } from '../i18n';

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
        eyebrow={messages.testSheet.report.eyebrow}
        title={messages.testSheet.report.title}
        description={messages.testSheet.report.description}
        onBack={onBack}
      />
      {error && <p className="error">{error}</p>}
      <ReportPreview markdown={markdown} />
    </section>
  );
}
