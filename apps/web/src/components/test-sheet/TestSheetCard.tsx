import React from 'react';
import { TestSheet } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { Badge } from '../ui/Badge';

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
  return (
    <Card className="test-sheet-card">
      <div className="sheet-card-order">{sheet.executionOrder}</div>
      <div className="sheet-card-content">
        <div className="card-topline">
          <Badge tone="blue">Fiche</Badge>
          {sheet.prerequisites && <Badge tone="neutral">Prerequis</Badge>}
        </div>
        <h3>{sheet.name}</h3>
        <p>{sheet.description || 'Sans description'}</p>
        <dl className="compact-definition-list">
          <dt>Action</dt>
          <dd>{sheet.action || '-'}</dd>
          <dt>Attendu</dt>
          <dd>{sheet.expectedResult || '-'}</dd>
        </dl>
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
