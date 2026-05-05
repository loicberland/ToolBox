import React, { useEffect, useState } from 'react';
import { testSheetApi, TestPlan } from '../api/testSheet';

type Props = {
  onEdit: (planId: number) => void;
};

export function TestPlanListPage({ onEdit }: Props) {
  const [plans, setPlans] = useState<TestPlan[]>([]);
  const [error, setError] = useState('');

  const load = () => testSheetApi.listPlans().then(setPlans).catch((err: Error) => setError(err.message));

  useEffect(() => {
    load();
  }, []);

  return (
    <section className="workspace">
      <header className="page-header">
        <div>
          <h2>Plans de test</h2>
          <p>Preparation et execution des Product Reviews.</p>
        </div>
        <button type="button" onClick={() => onEdit(0)}>Nouveau plan</button>
      </header>
      {error && <p className="error">{error}</p>}
      <div className="plan-grid">
        {plans.map((plan) => (
          <article className="plan-card" key={plan.id}>
            <h3>{plan.name}</h3>
            <p>{plan.description || 'Sans description'}</p>
            <div className="button-row">
              <button type="button" onClick={() => onEdit(plan.id)}>Ouvrir</button>
              <button className="secondary" type="button" onClick={async () => { await testSheetApi.duplicatePlan(plan.id); load(); }}>Dupliquer</button>
              <button className="danger" type="button" onClick={async () => { await testSheetApi.deletePlan(plan.id); load(); }}>Supprimer</button>
            </div>
          </article>
        ))}
      </div>
      {plans.length === 0 && <p className="muted">Aucun plan pour le moment.</p>}
    </section>
  );
}
