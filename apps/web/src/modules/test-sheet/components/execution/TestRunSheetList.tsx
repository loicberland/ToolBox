import React from 'react';
import { TestRunSheet } from '../../api/testSheet';
import { Card, CardHeader } from '../../../../shared/components/ui/Card';
import { hasMarkdownContent } from '../../../../shared/components/ui/MarkdownPreview';
import { SmartEllipsisText } from '../../../../shared/components/ui/SmartEllipsisText';
import { StatusBadge } from '../execution/StatusBadge';
import { getRunSheetProgress, getRunSheetProgressSummary } from '../execution/runStatus';

type Props = {
  sheets: TestRunSheet[];
  selectedSheetId?: number;
  onSelect: (sheetId: number) => void;
};

export function TestRunSheetList({ sheets, selectedSheetId, onSelect }: Props) {
  const selectWithKeyboard = (event: React.KeyboardEvent<HTMLDivElement>, sheetId: number) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onSelect(sheetId);
    }
  };

  return (
    <Card className="run-sheet-list-card">
      <CardHeader>
        <div>
          <span className="section-kicker">Tests</span>
          <h3>{sheets.length} fiche{sheets.length > 1 ? 's' : ''}</h3>
        </div>
      </CardHeader>
      <div className="run-sheet-list">
        {sheets.map((sheet) => {
          const progress = getRunSheetProgress(sheet);

          return (
            <div
              className={`run-sheet-list-item ${sheet.id === selectedSheetId ? 'active' : ''}`}
              key={sheet.id}
              role="button"
              tabIndex={0}
              onClick={() => onSelect(sheet.id)}
              onKeyDown={(event) => selectWithKeyboard(event, sheet.id)}
            >
              <span className="run-list-order">{sheet.executionOrder}</span>
              <div className="run-sheet-list-main">
                <div className="run-sheet-list-title">
                  <SmartEllipsisText text={sheet.name} className="run-sheet-name" />
                  <StatusBadge status={progress.status} />
                </div>
                {hasMarkdownContent(sheet.description) && <SmartEllipsisText text={sheet.description} className="run-sheet-description" />}
                <div className="run-sheet-progress-summary">{getRunSheetProgressSummary(sheet)}</div>
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
}
