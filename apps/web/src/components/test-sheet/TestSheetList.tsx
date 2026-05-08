import React from 'react';
import { TestSheet } from '../../api/testSheet';
import { TestSheetCard } from './TestSheetCard';

type Props = {
  sheets: TestSheet[];
  onEdit: (sheet: TestSheet) => void | Promise<void>;
  onDelete: (sheet: TestSheet) => void | Promise<void>;
  onDuplicate: (sheet: TestSheet) => void | Promise<void>;
  onMove: (sheet: TestSheet, direction: -1 | 1) => void | Promise<void>;
  editingSheetId?: number;
  recentlyMovedSheetId?: number;
  renderEditor?: (sheet: TestSheet) => React.ReactNode;
};

export function TestSheetList({ sheets, onEdit, onDelete, onDuplicate, onMove, editingSheetId, recentlyMovedSheetId, renderEditor }: Props) {
  if (sheets.length === 0) {
    return null;
  }
  return (
    <div className="sheet-list">
      {sheets.map((sheet, index) => (
        <React.Fragment key={sheet.id}>
          <TestSheetCard
            sheet={sheet}
            index={index}
            total={sheets.length}
            onEdit={() => onEdit(sheet)}
            onDelete={() => onDelete(sheet)}
            onDuplicate={() => onDuplicate(sheet)}
            onMove={(direction) => onMove(sheet, direction)}
            recentlyMoved={sheet.id === recentlyMovedSheetId}
          />
          {sheet.id === editingSheetId && renderEditor?.(sheet)}
        </React.Fragment>
      ))}
    </div>
  );
}
