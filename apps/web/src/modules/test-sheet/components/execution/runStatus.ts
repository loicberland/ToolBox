import { TestRunSheet, TestRunStatus, TestRunStep } from '../../api/testSheet';
import { messages } from '../../../../i18n';

export type RunItemStatus = TestRunSheet['status'];

export function computeSheetStatusFromSteps(steps: TestRunStep[], fallback: RunItemStatus = 'pending'): RunItemStatus {
  if (steps.length === 0) {
    return fallback;
  }

  const nonSkippedSteps = steps.filter((step) => step.status !== 'skipped');
  if (nonSkippedSteps.length === 0) {
    return 'skipped';
  }
  if (nonSkippedSteps.some((step) => step.status === 'failed')) {
    return 'failed';
  }
  if (nonSkippedSteps.some((step) => step.status === 'blocked')) {
    return 'blocked';
  }
  if (nonSkippedSteps.some((step) => step.status === 'pending')) {
    return 'pending';
  }
  if (nonSkippedSteps.every((step) => step.status === 'passed')) {
    return 'passed';
  }
  return fallback;
}

export function getRunStepProgress(steps: TestRunStep[] = []) {
  return {
    total: steps.length,
    pending: steps.filter((step) => step.status === 'pending').length,
    passed: steps.filter((step) => step.status === 'passed').length,
    failed: steps.filter((step) => step.status === 'failed').length,
    blocked: steps.filter((step) => step.status === 'blocked').length,
    skipped: steps.filter((step) => step.status === 'skipped').length,
    done: steps.filter((step) => step.status !== 'pending').length,
  };
}

export function getRunSheetProgress(sheet: TestRunSheet) {
  const steps = sheet.steps ?? [];
  return {
    ...getRunStepProgress(steps),
    status: computeSheetStatusFromSteps(steps, sheet.status),
  };
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

export function getRunSheetProgressSummary(sheet: TestRunSheet) {
  const progress = getRunSheetProgress(sheet);
  const parts = [`${progress.total} action${progress.total > 1 ? 's' : ''}`];

  if (progress.passed > 0) {
    parts.push(`${progress.passed} ${messages.testSheet.run.passedPlural}`);
  }
  if (progress.failed > 0) {
    parts.push(`${progress.failed} ${messages.testSheet.run.failedPlural}`);
  }
  if (progress.blocked > 0) {
    parts.push(`${progress.blocked} ${messages.testSheet.run.blockedPlural}`);
  }
  if (progress.skipped > 0) {
    parts.push(`${progress.skipped} ${messages.testSheet.run.skippedPlural}`);
  }
  if (progress.pending > 0 || parts.length === 1) {
    parts.push(`${progress.pending} ${messages.status.pending.toLowerCase()}`);
  }

  return parts.join(' - ');
}

export function isRunEditable(status?: TestRunStatus | string) {
  return status === 'running';
}

export function isRunReadOnly(status?: TestRunStatus | string) {
  return Boolean(status) && !isRunEditable(status);
}
