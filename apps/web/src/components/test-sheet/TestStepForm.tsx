import React, { useEffect, useState } from 'react';
import { StepInput, TestSheetStep } from '../../api/testSheet';
import { Button } from '../ui/Button';

type Props = {
  step?: TestSheetStep;
  nextOrder: number;
  onSubmit: (input: StepInput) => Promise<void>;
  onCancel?: () => void;
};

export function TestStepForm({ step, nextOrder, onSubmit, onCancel }: Props) {
  const [value, setValue] = useState<StepInput>(newStep(nextOrder));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue(step ? {
      action: step.action,
      field: step.field,
      expectedResult: step.expectedResult,
      executionOrder: step.executionOrder,
    } : newStep(nextOrder));
  }, [step, nextOrder]);

  return (
    <form
      className="form-grid step-form"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        await onSubmit(value);
        setSaving(false);
        if (!step) {
          setValue(newStep(nextOrder + 1));
        }
      }}
    >
      <label>
        Champ specifique
        <input value={value.field} onChange={(event) => setValue({ ...value, field: event.target.value })} placeholder="Ex: bouton Valider, champ email..." />
      </label>
      <label>
        Action
        <textarea value={value.action} onChange={(event) => setValue({ ...value, action: event.target.value })} required />
      </label>
      <label>
        Resultat attendu
        <textarea value={value.expectedResult} onChange={(event) => setValue({ ...value, expectedResult: event.target.value })} />
      </label>
      <div className="button-row">
        <Button type="submit" disabled={saving}>{saving ? 'Enregistrement...' : step ? 'Sauvegarder' : 'Ajouter l etape'}</Button>
        {onCancel && <Button variant="secondary" type="button" onClick={onCancel}>Annuler</Button>}
      </div>
    </form>
  );
}

function newStep(order: number): StepInput {
  return {
    action: '',
    field: '',
    expectedResult: '',
    executionOrder: order,
  };
}
