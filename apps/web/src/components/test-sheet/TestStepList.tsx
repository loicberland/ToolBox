import React from 'react';
import { TestSheetStep } from '../../api/testSheet';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { hasMarkdownContent, MarkdownPreview } from '../ui/MarkdownPreview';

type Props = {
  steps: TestSheetStep[];
  onEdit: (step: TestSheetStep) => void;
  onDelete: (step: TestSheetStep) => void;
  onDuplicate: (step: TestSheetStep) => void;
  onMove: (step: TestSheetStep, direction: -1 | 1) => void;
  editingStepId?: number;
  renderEditor?: (step: TestSheetStep) => React.ReactNode;
};

export function TestStepList({ steps, onEdit, onDelete, onDuplicate, onMove, editingStepId, renderEditor }: Props) {
  if (steps.length === 0) {
    return null;
  }

  return (
    <div className="step-list">
      {steps.map((step, index) => (
        <React.Fragment key={step.id}>
          <Card className="step-card">
            <div className="sheet-card-order">{step.executionOrder}</div>
            <div className="step-card-content">
              {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} compact /> : <p className="muted">Etape sans action</p>}
              <div className="button-row">
                <Button type="button" variant="secondary" size="sm" onClick={() => onMove(step, -1)} disabled={index === 0}>Monter</Button>
                <Button type="button" variant="secondary" size="sm" onClick={() => onMove(step, 1)} disabled={index === steps.length - 1}>Descendre</Button>
                <Button type="button" variant="secondary" size="sm" onClick={() => onEdit(step)}>Modifier</Button>
                <Button type="button" variant="secondary" size="sm" onClick={() => onDuplicate(step)}>Dupliquer</Button>
                <Button type="button" variant="danger" size="sm" onClick={() => onDelete(step)}>Supprimer</Button>
              </div>
            </div>
          </Card>
          {step.id === editingStepId && renderEditor?.(step)}
        </React.Fragment>
      ))}
    </div>
  );
}
