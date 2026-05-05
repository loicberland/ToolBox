import React from 'react';
import { TestSheet } from '../../api/testSheet';
import { EmptyState } from '../ui/EmptyState';
import { TestSheetCard } from './TestSheetCard';

type Props = {
  sheets: TestSheet[];
  onEdit: (sheet: TestSheet) => void;
  onDelete: (sheet: TestSheet) => void;
  onDuplicate: (sheet: TestSheet) => void;
  onMove: (sheet: TestSheet, direction: -1 | 1) => void;
};

export function TestSheetList({ sheets, onEdit, onDelete, onDuplicate, onMove }: Props) {
  if (sheets.length === 0) {
    return <EmptyState title="Aucune fiche" description="Ajoutez une premiere fiche pour pouvoir lancer une execution." />;
  }
  return (
    <div className="sheet-list">
      {sheets.map((sheet, index) => (
        <TestSheetCard
          key={sheet.id}
          sheet={sheet}
          index={index}
          total={sheets.length}
          onEdit={() => onEdit(sheet)}
          onDelete={() => onDelete(sheet)}
          onDuplicate={() => onDuplicate(sheet)}
          onMove={(direction) => onMove(sheet, direction)}
        />
      ))}
    </div>
  );
}
