import React, { useEffect, useState } from 'react';
import { PlanInput, TestPlan } from '../../api/testSheet';
import { messages } from '../../i18n';
import { Button } from '../ui/Button';

type Props = {
  plan?: TestPlan;
  onSubmit: (input: PlanInput) => Promise<void>;
};

const emptyPlan: PlanInput = { name: '', description: '', mockupSettings: '' };

export function TestPlanForm({ plan, onSubmit }: Props) {
  const [value, setValue] = useState<PlanInput>(emptyPlan);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setValue(plan ? { name: plan.name, description: plan.description, mockupSettings: plan.mockupSettings } : emptyPlan);
  }, [plan]);

  return (
    <form
      className="form-grid"
      onSubmit={async (event) => {
        event.preventDefault();
        setError('');
        setSaving(true);
        try {
          await onSubmit(value);
        } catch (err) {
          setError((err as Error).message);
        } finally {
          setSaving(false);
        }
      }}
    >
      <label>
        {messages.testSheet.edit.name}
        <input value={value.name} onChange={(event) => setValue({ ...value, name: event.target.value })} required />
      </label>
      <label>
        {messages.testSheet.edit.description}
        <textarea value={value.description} onChange={(event) => setValue({ ...value, description: event.target.value })} />
      </label>
      {error && <p className="error">{error}</p>}
      <div className="form-actions">
        <Button type="submit" disabled={saving}>{saving ? messages.common.saving : messages.common.save}</Button>
      </div>
    </form>
  );
}
