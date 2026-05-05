import React from 'react';
import { TestRunSheet } from '../../api/testSheet';
import { Card } from '../ui/Card';
import { StatusBadge } from './StatusBadge';

type Props = {
  status: string;
  sheets: TestRunSheet[];
};

export function TestRunProgress({ status, sheets }: Props) {
  const steps = sheets.flatMap((sheet) => sheet.steps ?? []);
  const done = steps.filter((step) => step.status !== 'pending').length;
  const total = steps.length;
  const percent = total === 0 ? 0 : Math.round((done / total) * 100);

  return (
    <Card className="run-progress-card">
      <div className="run-progress-header">
        <div>
          <span className="section-kicker">Progression</span>
          <strong>{done} / {total} etapes traitees</strong>
        </div>
        <StatusBadge status={status} />
      </div>
      <div className="progress-track" aria-label={`Progression ${percent}%`}>
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
      <div className="status-summary">
        {(['pending', 'passed', 'failed', 'blocked', 'skipped'] as const).map((item) => (
          <div key={item}>
            <StatusBadge status={item} />
            <strong>{steps.filter((step) => step.status === item).length}</strong>
          </div>
        ))}
      </div>
    </Card>
  );
}
