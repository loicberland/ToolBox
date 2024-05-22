import React from 'react';
import { statusLabel } from '../../i18n';
import { Badge } from '../ui/Badge';

type Status =
  | 'pending'
  | 'passed'
  | 'failed'
  | 'blocked'
  | 'skipped'
  | 'running'
  | 'completed'
  | 'canceled'
  | 'archived'
  | 'finished'
  | 'aborted'
  | 'draft'
  | 'ready';

const tones: Record<Status, 'neutral' | 'blue' | 'green' | 'red' | 'orange' | 'gray'> = {
  pending: 'gray',
  passed: 'green',
  failed: 'red',
  blocked: 'orange',
  skipped: 'neutral',
  running: 'blue',
  completed: 'green',
  canceled: 'red',
  archived: 'neutral',
  finished: 'green',
  aborted: 'red',
  draft: 'gray',
  ready: 'blue',
};

export function StatusBadge({ status }: { status: Status | string }) {
  const knownStatus = status as Status;
  return <Badge tone={tones[knownStatus] ?? 'neutral'}>{statusLabel(status)}</Badge>;
}
