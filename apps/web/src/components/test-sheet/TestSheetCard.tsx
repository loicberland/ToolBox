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
  onEdit: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
  onMove: (direction: -1 | 1) => void;
};

export function TestSheetCard({ sheet, index, total, onEdit, onDelete, onDuplicate, onMove }: Props) {
  const stepCount = sheet.steps?.length ?? 0;

  return (
    <Card className="test-sheet-card">
      <div className="sheet-card-order">{sheet.executionOrder}</div>
      <div className="sheet-card-content">
        <div className="card-topline">
          <Badge tone="blue">Fiche</Badge>
          <Badge tone="neutral">{stepCount} etape{stepCount > 1 ? 's' : ''}</Badge>
        </div>
        <h3>{sheet.name}</h3>
        {hasMarkdownContent(sheet.description) ? <MarkdownPreview content={sheet.description} compact /> : <p>Aucune description</p>}
        <div className="button-row">
          <Button type="button" variant="secondary" size="sm" onClick={() => onMove(-1)} disabled={index === 0}>Monter</Button>
          <Button type="button" variant="secondary" size="sm" onClick={() => onMove(1)} disabled={index === total - 1}>Descendre</Button>
          <Button type="button" variant="secondary" size="sm" onClick={onEdit}>Modifier</Button>
          <Button type="button" variant="secondary" size="sm" onClick={onDuplicate}>Dupliquer</Button>
          <Button type="button" variant="danger" size="sm" onClick={onDelete}>Supprimer</Button>
        </div>
      </div>
    </Card>
  );
}
