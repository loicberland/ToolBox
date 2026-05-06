import React from 'react';
import { TestSheet } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { Badge } from '../ui/Badge';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';

type Props = {
  sheet: TestSheet;
  index: number;
  total: number;
  onEdit: () => void | Promise<void>;
  onDelete: () => void | Promise<void>;
  onDuplicate: () => void | Promise<void>;
  onMove: (direction: -1 | 1) => void | Promise<void>;
};

export function TestSheetCard({ sheet, index, total, onEdit, onDelete, onDuplicate, onMove }: Props) {
  const stepCount = sheet.steps?.length ?? 0;
  const stopAndRun = (event: React.MouseEvent, action: () => void | Promise<void>) => {
    event.stopPropagation();
    void action();
  };

  return (
    <Card className="test-sheet-card" role="button" tabIndex={0} onClick={() => { void onEdit(); }} onKeyDown={(event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        void onEdit();
      }
    }}>
      <div className="sheet-card-order">{sheet.executionOrder}</div>
      <div className="sheet-card-content">
        <div className="card-topline">
          <Badge tone="blue">Fiche</Badge>
          <Badge tone="neutral">{stepCount} etape{stepCount > 1 ? 's' : ''}</Badge>
        </div>
        <h3>{sheet.name}</h3>
        {hasMarkdownContent(sheet.description) ? <MarkdownPreview content={sheet.description} compact /> : <p>Aucune description</p>}
        <div className="button-row">
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, () => onMove(-1))} disabled={index === 0}>Monter</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, () => onMove(1))} disabled={index === total - 1}>Descendre</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, onEdit)}>Modifier</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, onDuplicate)}>Dupliquer</Button>
          <Button type="button" variant="danger" size="sm" onClick={(event) => stopAndRun(event, onDelete)}>Supprimer</Button>
        </div>
      </div>
    </Card>
  );
}
