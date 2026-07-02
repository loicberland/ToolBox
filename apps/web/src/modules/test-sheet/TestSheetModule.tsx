import React, { useState } from 'react';
import { TestPlanEditPage } from './pages/TestPlanEditPage';
import { TestPlanListPage } from './pages/TestPlanListPage';
import { TestRunPage } from './pages/TestRunPage';
import { TestRunReportPage } from './pages/TestRunReportPage';

type TestSheetView =
  | { name: 'test-plans' }
  | { name: 'test-plan-edit'; planId: number }
  | { name: 'test-run'; runId: number }
  | { name: 'test-report'; runId: number; returnTo: 'plans' | 'run' };

export function TestSheetModule() {
  const [view, setView] = useState<TestSheetView>({ name: 'test-plans' });

  return (
    <>
      {view.name === 'test-plans' && (
        <TestPlanListPage
          onEdit={(planId) => setView({ name: 'test-plan-edit', planId })}
          onRun={(runId) => setView({ name: 'test-run', runId })}
          onReport={(runId) => setView({ name: 'test-report', runId, returnTo: 'plans' })}
        />
      )}
      {view.name === 'test-plan-edit' && (
        <TestPlanEditPage
          planId={view.planId}
          onBack={() => setView({ name: 'test-plans' })}
          onRun={(runId) => setView({ name: 'test-run', runId })}
        />
      )}
      {view.name === 'test-run' && (
        <TestRunPage
          runId={view.runId}
          onBack={() => setView({ name: 'test-plans' })}
          onReport={(runId) => setView({ name: 'test-report', runId, returnTo: 'run' })}
        />
      )}
      {view.name === 'test-report' && (
        <TestRunReportPage
          runId={view.runId}
          onBack={() => setView(view.returnTo === 'run' ? { name: 'test-run', runId: view.runId } : { name: 'test-plans' })}
        />
      )}
    </>
  );
}
