import React from 'react';
import { RunGroup, TestRunSheet } from '../../api/testSheet';
import { Card } from '../ui/Card';
import { StatusBadge } from './StatusBadge';
import { getRunSheetProgress } from './runStatus';

type Props = {
  status: string;
  sheets: TestRunSheet[];
  groups?: RunGroup[];
};

export function TestRunProgress({ status, sheets, groups }: Props) {
  const groupProgress = groups && groups.length > 0
    ? groups.map((group) => getGroupStatus(group.sheets ?? []))
    : sheets.map(getRunSheetProgress).map((item) => item.status);
  const done = groupProgress.filter((item) => item !== 'pending').length;
  const total = groupProgress.length;
  const percent = total === 0 ? 0 : Math.round((done / total) * 100);

  return (
    <Card className="run-progress-card">
      <div className="run-progress-header">
        <div>
          <span className="section-kicker">Progression</span>
          <strong>{done} / {total} sous-plans traités</strong>
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
            <strong>{groupProgress.filter((progress) => progress === item).length}</strong>
          </div>
        ))}
      </div>
    </Card>
  );
}

export function getGroupStatus(sheets: TestRunSheet[] = []) {
  if (sheets.length === 0) {
    return 'pending' as const;
  }
  const statuses = sheets.map((sheet) => getRunSheetProgress(sheet).status);
  if (statuses.includes('failed')) {
    return 'failed' as const;
  }
  if (statuses.includes('blocked')) {
    return 'blocked' as const;
  }
  if (statuses.includes('pending')) {
    return 'pending' as const;
  }
  if (statuses.every((status) => status === 'skipped')) {
    return 'skipped' as const;
  }
  return 'passed' as const;
}
