import React, { useEffect, useState } from 'react';
import { SheetInput, TestSheet } from '../../api/testSheet';
import { Button } from '../ui/Button';

type Props = {
  sheet?: TestSheet;
  nextOrder: number;
  onSubmit: (input: SheetInput) => Promise<void>;
  onCancel?: () => void;
  formId?: string;
  hideActions?: boolean;
};

export function TestSheetForm({ sheet, nextOrder, onSubmit, onCancel, formId, hideActions = false }: Props) {
  const [value, setValue] = useState<SheetInput>(newSheet(nextOrder));
  const [saving, setSaving] = useState(false);
  const isEditing = Boolean(sheet?.id);

  useEffect(() => {
    setValue(sheet ? {
      name: sheet.name,
      description: sheet.description,
      prerequisites: sheet.prerequisites,
      config: sheet.config,
      command: sheet.command,
      notes: sheet.notes,
      action: sheet.action,
      expectedResult: sheet.expectedResult,
      executionOrder: sheet.executionOrder,
      mockupSettings: sheet.mockupSettings,
    } : newSheet(nextOrder));
  }, [sheet, nextOrder]);

  return (
    <form
      id={formId}
      className="form-grid sheet-form"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        await onSubmit(value);
        setSaving(false);
        if (!isEditing) {
          setValue(newSheet(nextOrder + 1));
        }
      }}
    >
      <label>
        Nom
        <input value={value.name} onChange={(event) => setValue({ ...value, name: event.target.value })} required />
      </label>
      <label>
        Description
        <textarea value={value.description} onChange={(event) => setValue({ ...value, description: event.target.value })} />
      </label>
      <label>
        Prerequis
        <textarea
          value={value.prerequisites}
          onChange={(event) => setValue({ ...value, prerequisites: event.target.value })}
          placeholder={'Markdown accepte.\n\nFichier a charger :\n\n```txt\nTEST\n```\n\nConfigurer RealTerm sur le Digi en .51.'}
        />
      </label>
      <label>
        Configuration
        <textarea value={value.config} onChange={(event) => setValue({ ...value, config: event.target.value })} placeholder="Markdown accepte" />
      </label>
      <label>
        Commande
        <textarea value={value.command} onChange={(event) => setValue({ ...value, command: event.target.value })} placeholder="Markdown accepte" />
      </label>
      <label>
        Notes
        <textarea value={value.notes} onChange={(event) => setValue({ ...value, notes: event.target.value })} placeholder="Markdown accepte" />
      </label>
      {!hideActions && (
        <div className="button-row">
          <Button type="submit" disabled={saving}>{saving ? 'Enregistrement...' : isEditing ? 'Sauvegarder' : 'Ajouter'}</Button>
          {onCancel && <Button variant="secondary" type="button" onClick={onCancel}>Annuler</Button>}
        </div>
      )}
    </form>
  );
}

function newSheet(order: number): SheetInput {
  return {
    name: '',
    description: '',
    prerequisites: '',
    config: '',
    command: '',
    notes: '',
    action: '',
    expectedResult: '',
    executionOrder: order,
    mockupSettings: '',
  };
}
