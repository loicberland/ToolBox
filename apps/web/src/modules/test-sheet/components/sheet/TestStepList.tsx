import React from 'react';
import { TestSheetStep } from '../../api/testSheet';
import { Button } from '../../../../shared/components/ui/Button';
import { Card } from '../../../../shared/components/ui/Card';
import { hasMarkdownContent, MarkdownPreview } from '../../../../shared/components/ui/MarkdownPreview';
import { messages } from '../../../../i18n';

type Props = {
  steps: TestSheetStep[];
  onEdit: (step: TestSheetStep) => void | Promise<void>;
  onDelete: (step: TestSheetStep) => void | Promise<void>;
  onDuplicate: (step: TestSheetStep) => void | Promise<void>;
  onMove: (step: TestSheetStep, direction: -1 | 1) => void | Promise<void>;
  registerItem: (stepId: number) => React.RefCallback<HTMLElement>;
  editingStepId?: number;
  renderEditor?: (step: TestSheetStep) => React.ReactNode;
};

export function TestStepList({ steps, onEdit, onDelete, onDuplicate, onMove, registerItem, editingStepId, renderEditor }: Props) {
  if (steps.length === 0) {
    return null;
  }

  return (
    <div className="step-list">
      {steps.map((step, index) => (
        <React.Fragment key={step.id}>
          <Card ref={registerItem(step.id)} className="step-card flip-reorder-item" role="button" tabIndex={0} onClick={() => { void onEdit(step); }} onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              void onEdit(step);
            }
          }}>
            <div className="sheet-card-order">{step.executionOrder}</div>
            <div className="step-card-content">
              {hasMarkdownContent(step.action) ? <MarkdownPreview content={step.action} compact /> : <p className="muted">{messages.testSheet.run.noAction}</p>}
              <div className="button-row">
                <Button type="button" variant="secondary" size="sm" onClick={(event) => { event.stopPropagation(); void onMove(step, -1); }} disabled={index === 0}>{messages.common.moveUp}</Button>
                <Button type="button" variant="secondary" size="sm" onClick={(event) => { event.stopPropagation(); void onMove(step, 1); }} disabled={index === steps.length - 1}>{messages.common.moveDown}</Button>
                <Button type="button" variant="secondary" size="sm" onClick={(event) => { event.stopPropagation(); void onEdit(step); }}>{messages.common.edit}</Button>
                <Button type="button" variant="secondary" size="sm" onClick={(event) => { event.stopPropagation(); void onDuplicate(step); }}>{messages.testSheet.plans.duplicate}</Button>
                <Button type="button" variant="danger" size="sm" onClick={(event) => { event.stopPropagation(); void onDelete(step); }}>{messages.common.delete}</Button>
              </div>
            </div>
          </Card>
          {step.id === editingStepId && renderEditor?.(step)}
        </React.Fragment>
      ))}
    </div>
  );
}
