import React from 'react';
import { TestPlan } from '../../api/testSheet';
import { Button } from '../../../../shared/components/ui/Button';
import { Card } from '../../../../shared/components/ui/Card';
import { Badge } from '../../../../shared/components/ui/Badge';
import { hasMarkdownContent, MarkdownPreview } from '../../../../shared/components/ui/MarkdownPreview';
import { StatusBadge } from '../execution/StatusBadge';
import { messages } from '../../../../i18n';

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
        <Badge tone="neutral">{sheetCount} {messages.testSheet.plans.sheetSingular}{sheetCount > 1 ? 's' : ''}</Badge>
      </div>
      <div className="card-main">
        <h3>{plan.name}</h3>
        {hasMarkdownContent(plan.description) ? <MarkdownPreview content={plan.description} compact /> : <p>{messages.testSheet.plans.withoutDescription}</p>}
      </div>
      <div className="card-meta">
        <span>{messages.testSheet.plans.updatedAt}</span>
        <strong>{formatDate(plan.updatedAt)}</strong>
      </div>
      <div className="button-row">
        <Button type="button" onClick={onEdit}>{messages.common.edit}</Button>
        <Button type="button" variant="secondary" disabled={sheetCount === 0} onClick={onRun}>{messages.common.execute}</Button>
        <Button type="button" variant="secondary" onClick={onDuplicate}>{messages.testSheet.plans.duplicate}</Button>
        <Button type="button" variant="danger" onClick={onDelete}>{messages.common.delete}</Button>
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
