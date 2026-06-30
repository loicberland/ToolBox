import React from 'react';
import { TestSheet } from '../../api/testSheet';
import { Button } from '../../../../shared/components/ui/Button';
import { Card } from '../../../../shared/components/ui/Card';
import { Badge } from '../../../../shared/components/ui/Badge';
import { hasMarkdownContent, MarkdownPreview } from '../../../../shared/components/ui/MarkdownPreview';
import { messages } from '../../../../i18n';

type Props = {
  sheet: TestSheet;
  index: number;
  total: number;
  onEdit: () => void | Promise<void>;
  onDelete: () => void | Promise<void>;
  onDuplicate: () => void | Promise<void>;
  onMove: (direction: -1 | 1) => void | Promise<void>;
};

export const TestSheetCard = React.forwardRef<HTMLElement, Props>(function TestSheetCard({ sheet, index, total, onEdit, onDelete, onDuplicate, onMove }, ref) {
  const stepCount = sheet.steps?.length ?? 0;
  const stopAndRun = (event: React.MouseEvent, action: () => void | Promise<void>) => {
    event.stopPropagation();
    void action();
  };

  return (
    <Card ref={ref} className="test-sheet-card flip-reorder-item" role="button" tabIndex={0} onClick={() => { void onEdit(); }} onKeyDown={(event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        void onEdit();
      }
    }}>
      <div className="sheet-card-order">{sheet.executionOrder}</div>
      <div className="sheet-card-content">
        <div className="card-topline">
          <Badge tone="blue">{messages.testSheet.edit.sheet}</Badge>
          <Badge tone="neutral">{stepCount} {messages.testSheet.edit.step}{stepCount > 1 ? 's' : ''}</Badge>
        </div>
        <h3>{sheet.name}</h3>
        {hasMarkdownContent(sheet.description) ? <MarkdownPreview content={sheet.description} compact /> : <p>{messages.testSheet.plans.noDescription}</p>}
        <div className="button-row">
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, () => onMove(-1))} disabled={index === 0}>{messages.common.moveUp}</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, () => onMove(1))} disabled={index === total - 1}>{messages.common.moveDown}</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, onEdit)}>{messages.common.edit}</Button>
          <Button type="button" variant="secondary" size="sm" onClick={(event) => stopAndRun(event, onDuplicate)}>{messages.testSheet.plans.duplicate}</Button>
          <Button type="button" variant="danger" size="sm" onClick={(event) => stopAndRun(event, onDelete)}>{messages.common.delete}</Button>
        </div>
      </div>
    </Card>
  );
});
