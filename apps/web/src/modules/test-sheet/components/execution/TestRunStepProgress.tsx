import React from 'react';
import { TestRunStep } from '../../api/testSheet';
import { messages } from '../../../../i18n';
import { StatusBadge } from '../execution/StatusBadge';
import { getRunStepProgress } from '../execution/runStatus';

type Props = {
  steps: TestRunStep[];
  title?: string;
};

export function TestRunStepProgress({ steps, title = messages.testSheet.run.progress }: Props) {
  const progress = getRunStepProgress(steps);
  const percent = progress.total === 0 ? 0 : Math.round((progress.done / progress.total) * 100);

  return (
    <div className="run-step-progress">
      <div className="run-step-progress-header">
        <div>
          <span className="section-kicker">{title}</span>
          <strong>{progress.done} / {progress.total} {messages.testSheet.run.actionsProcessed}</strong>
        </div>
      </div>
      <div className="progress-track" aria-label={`Progression ${percent}%`}>
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
      <div className="run-step-progress-summary">
        <div><StatusBadge status="pending" /><strong>{progress.pending}</strong></div>
        <div><StatusBadge status="passed" /><strong>{progress.passed}</strong></div>
        <div><StatusBadge status="failed" /><strong>{progress.failed}</strong></div>
        <div><StatusBadge status="blocked" /><strong>{progress.blocked}</strong></div>
        <div><StatusBadge status="skipped" /><strong>{progress.skipped}</strong></div>
      </div>
    </div>
  );
}
