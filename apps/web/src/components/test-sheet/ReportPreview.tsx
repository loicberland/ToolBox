import React from 'react';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';

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
      <pre className="report-preview">{markdown || 'Chargement du rapport...'}</pre>
    </Card>
  );
}
