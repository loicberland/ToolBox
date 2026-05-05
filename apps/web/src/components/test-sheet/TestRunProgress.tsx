import React from 'react';
import { TestRunSheet } from '../../api/testSheet';
import { Card } from '../ui/Card';
import { StatusBadge } from './StatusBadge';

type Props = {
  status: string;
  sheets: TestRunSheet[];
};

export function TestRunProgress({ status, sheets }: Props) {
  const done = sheets.filter((sheet) => sheet.status !== 'pending').length;
  const total = sheets.length;
  const percent = total === 0 ? 0 : Math.round((done / total) * 100);

  return (
    <Card className="run-progress-card">
      <div className="run-progress-header">
        <div>
          <span className="section-kicker">Progression</span>
          <strong>{done} / {total} tests traites</strong>
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
            <strong>{sheets.filter((sheet) => sheet.status === item).length}</strong>
          </div>
        ))}
      </div>
    </Card>
  );
}
