import React from 'react';
import { TestSheet } from '../../api/testSheet';

type Props = {
  sheets: TestSheet[];
  onEdit: (sheet: TestSheet) => void;
  onDelete: (sheet: TestSheet) => void;
  onDuplicate: (sheet: TestSheet) => void;
  onMove: (sheet: TestSheet, direction: -1 | 1) => void;
};

export function TestSheetList({ sheets, onEdit, onDelete, onDuplicate, onMove }: Props) {
  if (sheets.length === 0) {
    return <p className="muted">Aucune fiche pour ce plan.</p>;
  }
  return (
    <div className="sheet-list">
      {sheets.map((sheet, index) => (
        <article className="sheet-card" key={sheet.id}>
          <div>
            <span className="order-badge">{sheet.executionOrder}</span>
            <h3>{sheet.name}</h3>
            <p>{sheet.description || 'Sans description'}</p>
          </div>
          <div className="button-row">
            <button className="secondary" type="button" onClick={() => onMove(sheet, -1)} disabled={index === 0}>Monter</button>
            <button className="secondary" type="button" onClick={() => onMove(sheet, 1)} disabled={index === sheets.length - 1}>Descendre</button>
            <button className="secondary" type="button" onClick={() => onEdit(sheet)}>Modifier</button>
            <button className="secondary" type="button" onClick={() => onDuplicate(sheet)}>Dupliquer</button>
            <button className="danger" type="button" onClick={() => onDelete(sheet)}>Supprimer</button>
          </div>
        </article>
      ))}
    </div>
  );
}
