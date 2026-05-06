import React from 'react';
import { TestPlan } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { Badge } from '../ui/Badge';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';
import { StatusBadge } from './StatusBadge';

type Props = {
  plan: TestPlan;
  sheetCount: number;
  onEdit: () => void;
  onRun: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
};

export function TestPlanCard({ plan, sheetCount, onEdit, onRun, onDuplicate, onDelete }: Props) {
  const status = sheetCount > 0 ? 'ready' : 'draft';

  return (
    <Card className="test-plan-card">
      <div className="card-topline">
        <StatusBadge status={status} />
        <Badge tone="neutral">{sheetCount} fiche{sheetCount > 1 ? 's' : ''}</Badge>
      </div>
      <div className="card-main">
        <h3>{plan.name}</h3>
        {hasMarkdownContent(plan.description) ? <MarkdownPreview content={plan.description} compact /> : <p>Sans description</p>}
      </div>
      <div className="card-meta">
        <span>Mis a jour</span>
        <strong>{formatDate(plan.updatedAt)}</strong>
      </div>
      <div className="button-row">
        <Button type="button" onClick={onEdit}>Modifier</Button>
        <Button type="button" variant="secondary" disabled={sheetCount === 0} onClick={onRun}>Executer</Button>
        <Button type="button" variant="secondary" onClick={onDuplicate}>Dupliquer</Button>
        <Button type="button" variant="danger" onClick={onDelete}>Supprimer</Button>
      </div>
    </Card>
  );
}

function formatDate(value: string) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  return `${date.toLocaleDateString('fr-FR')} ${date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}`;
}
