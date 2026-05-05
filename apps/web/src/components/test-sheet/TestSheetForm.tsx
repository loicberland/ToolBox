import React, { useEffect, useState } from 'react';
import { SheetInput, TestSheet } from '../../api/testSheet';
import { Button } from '../ui/Button';

type Props = {
  sheet?: TestSheet;
  nextOrder: number;
  onSubmit: (input: SheetInput) => Promise<void>;
  onCancel?: () => void;
};

export function TestSheetForm({ sheet, nextOrder, onSubmit, onCancel }: Props) {
  const [value, setValue] = useState<SheetInput>(newSheet(nextOrder));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue(sheet ? {
      name: sheet.name,
      description: sheet.description,
      prerequisites: sheet.prerequisites,
      action: sheet.action,
      expectedResult: sheet.expectedResult,
      executionOrder: sheet.executionOrder,
      mockupSettings: sheet.mockupSettings,
    } : newSheet(nextOrder));
  }, [sheet, nextOrder]);

  return (
    <form
      className="form-grid sheet-form"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        await onSubmit(value);
        setSaving(false);
        if (!sheet) {
          setValue(newSheet(nextOrder + 1));
        }
      }}
    >
      <label>
        Nom
        <input value={value.name} onChange={(event) => setValue({ ...value, name: event.target.value })} required />
      </label>
      <label>
        Ordre
        <input type="number" min="1" value={value.executionOrder} onChange={(event) => setValue({ ...value, executionOrder: Number(event.target.value) })} />
      </label>
      <label>
        Description
        <textarea value={value.description} onChange={(event) => setValue({ ...value, description: event.target.value })} />
      </label>
      <label>
        Prerequis
        <textarea value={value.prerequisites} onChange={(event) => setValue({ ...value, prerequisites: event.target.value })} />
      </label>
      <label>
        Action a effectuer
        <textarea value={value.action} onChange={(event) => setValue({ ...value, action: event.target.value })} />
      </label>
      <label>
        Resultat attendu
        <textarea value={value.expectedResult} onChange={(event) => setValue({ ...value, expectedResult: event.target.value })} />
      </label>
      <label>
        Parametres de maquette
        <textarea value={value.mockupSettings} onChange={(event) => setValue({ ...value, mockupSettings: event.target.value })} />
      </label>
      <div className="button-row">
        <Button type="submit" disabled={saving}>{saving ? 'Enregistrement...' : sheet ? 'Modifier la fiche' : 'Ajouter la fiche'}</Button>
        {onCancel && <Button variant="secondary" type="button" onClick={onCancel}>Annuler</Button>}
      </div>
    </form>
  );
}

function newSheet(order: number): SheetInput {
  return {
    name: '',
    description: '',
    prerequisites: '',
    action: '',
    expectedResult: '',
    executionOrder: order,
    mockupSettings: '',
  };
}
