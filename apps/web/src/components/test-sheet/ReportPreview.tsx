import React from 'react';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';

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
          <strong>Rapport copiable</strong>
        </div>
        <Button type="button" variant="secondary" onClick={copy} disabled={!markdown}>Copier</Button>
      </div>
      <div className="report-preview">
        {hasMarkdownContent(markdown) ? <MarkdownPreview content={markdown} /> : <p>Chargement du rapport...</p>}
      </div>
    </Card>
  );
}
