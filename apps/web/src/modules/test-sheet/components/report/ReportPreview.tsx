import React from 'react';
import { Button } from '../../../../shared/components/ui/Button';
import { Card } from '../../../../shared/components/ui/Card';
import { hasMarkdownContent, MarkdownPreview } from '../../../../shared/components/ui/MarkdownPreview';
import { messages } from '../../../../i18n';

type Props = {
  markdown: string;
};

export function ReportPreview({ markdown }: Props) {
  const copy = async () => {
    await navigator.clipboard.writeText(markdown);
  };

  return (
    <Card className="report-card">
      <div className="report-toolbar">
        <div>
          <span className="section-kicker">Markdown</span>
          <strong>{messages.testSheet.report.copyable}</strong>
        </div>
        <Button type="button" variant="secondary" onClick={copy} disabled={!markdown}>{messages.testSheet.report.copy}</Button>
      </div>
      <div className="report-preview">
        {hasMarkdownContent(markdown) ? <MarkdownPreview content={markdown} /> : <p>{messages.testSheet.report.loading}</p>}
      </div>
    </Card>
  );
}
