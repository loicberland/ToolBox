import { TestRunSheet, TestRunStep } from '../../api/testSheet';

export type RunItemStatus = TestRunSheet['status'];

export function computeSheetStatusFromSteps(steps: TestRunStep[], fallback: RunItemStatus = 'pending'): RunItemStatus {
  if (steps.length === 0) {
    return fallback;
  }
  if (steps.some((step) => step.status === 'failed')) {
    return 'failed';
  }
  if (steps.some((step) => step.status === 'blocked')) {
    return 'blocked';
  }
  if (steps.every((step) => step.status === 'passed')) {
    return 'passed';
  }
  if (steps.every((step) => step.status === 'skipped')) {
    return 'skipped';
  }
  return 'pending';
}

export function getRunSheetProgress(sheet: TestRunSheet) {
  const steps = sheet.steps ?? [];

  return {
    total: steps.length,
    pending: steps.filter((step) => step.status === 'pending').length,
    passed: steps.filter((step) => step.status === 'passed').length,
    failed: steps.filter((step) => step.status === 'failed').length,
    blocked: steps.filter((step) => step.status === 'blocked').length,
    skipped: steps.filter((step) => step.status === 'skipped').length,
    status: computeSheetStatusFromSteps(steps, sheet.status),
  };
}

export function getRunSheetProgressSummary(sheet: TestRunSheet) {
  const progress = getRunSheetProgress(sheet);
  const parts = [`${progress.total} action${progress.total > 1 ? 's' : ''}`];

  if (progress.passed > 0) {
    parts.push(`${progress.passed} reussie${progress.passed > 1 ? 's' : ''}`);
  }
  if (progress.failed > 0) {
    parts.push(`${progress.failed} echouee${progress.failed > 1 ? 's' : ''}`);
  }
  if (progress.blocked > 0) {
    parts.push(`${progress.blocked} bloquee${progress.blocked > 1 ? 's' : ''}`);
  }
  if (progress.skipped > 0) {
    parts.push(`${progress.skipped} ignoree${progress.skipped > 1 ? 's' : ''}`);
  }
  if (progress.pending > 0 || parts.length === 1) {
    parts.push(`${progress.pending} en attente`);
  }

  return parts.join(' - ');
}
