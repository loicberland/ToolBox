import React from 'react';
import { Badge } from '../ui/Badge';

type Status =
  | 'pending'
  | 'passed'
  | 'failed'
  | 'blocked'
  | 'skipped'
  | 'running'
  | 'completed'
  | 'archived'
  | 'finished'
  | 'aborted'
  | 'draft'
  | 'ready';

const labels: Record<Status, string> = {
  pending: 'En attente',
  passed: 'Reussi',
  failed: 'Echoue',
  blocked: 'Bloque',
  skipped: 'Ignore',
  running: 'En cours',
  completed: 'Termine',
  archived: 'Archive',
  finished: 'Termine',
  aborted: 'Abandonne',
  draft: 'Brouillon',
  ready: 'Pret',
};

const tones: Record<Status, 'neutral' | 'blue' | 'green' | 'red' | 'orange' | 'gray'> = {
  pending: 'gray',
  passed: 'green',
  failed: 'red',
  blocked: 'orange',
  skipped: 'neutral',
  running: 'blue',
  completed: 'green',
  archived: 'neutral',
  finished: 'green',
  aborted: 'red',
  draft: 'gray',
  ready: 'blue',
};

export function StatusBadge({ status }: { status: Status | string }) {
  const knownStatus = status as Status;
  return <Badge tone={tones[knownStatus] ?? 'neutral'}>{labels[knownStatus] ?? status}</Badge>;
}
