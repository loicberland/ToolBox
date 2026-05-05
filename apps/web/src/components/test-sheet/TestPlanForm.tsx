import React, { useEffect, useState } from 'react';
import { PlanInput, TestPlan } from '../../api/testSheet';

type Props = {
  plan?: TestPlan;
  onSubmit: (input: PlanInput) => Promise<void>;
};

const emptyPlan: PlanInput = { name: '', description: '', mockupSettings: '' };

export function TestPlanForm({ plan, onSubmit }: Props) {
  const [value, setValue] = useState<PlanInput>(emptyPlan);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue(plan ? { name: plan.name, description: plan.description, mockupSettings: plan.mockupSettings } : emptyPlan);
  }, [plan]);

  return (
    <form
      className="form-grid"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        await onSubmit(value);
        setSaving(false);
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
        Parametres de maquette
        <textarea value={value.mockupSettings} onChange={(event) => setValue({ ...value, mockupSettings: event.target.value })} placeholder='{"environment":"demo"}' />
      </label>
      <button type="submit" disabled={saving}>{saving ? 'Enregistrement...' : 'Enregistrer'}</button>
    </form>
  );
}
