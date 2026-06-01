import React, { useCallback, useEffect, useRef, useState } from 'react';
import { fetchModules, ModuleInfo } from './api/modules';
import { TestPlanEditPage } from './pages/TestPlanEditPage';
import { TestPlanListPage } from './pages/TestPlanListPage';
import { TestRunPage } from './pages/TestRunPage';
import { TestRunReportPage } from './pages/TestRunReportPage';
import { V10LabPage } from './pages/V10LabPage';

type View =
  | { name: 'module' }
  | { name: 'test-plans' }
  | { name: 'test-plan-edit'; planId: number }
  | { name: 'test-run'; runId: number }
  | { name: 'test-report'; runId: number; returnTo: 'plans' | 'run' };

export type BeforeLeaveHandler = () => Promise<boolean>;

const App = () => {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selectedModuleId, setSelectedModuleId] = useState<string>('test-sheet');
  const [view, setView] = useState<View>({ name: 'test-plans' });
  const [error, setError] = useState<string>('');
  const beforeLeaveHandlers = useRef<Record<string, BeforeLeaveHandler | undefined>>({});

  useEffect(() => {
    fetchModules()
      .then((items) => {
        setModules(items);
        if (!items.some((item) => item.id === selectedModuleId)) {
          setSelectedModuleId(items[0]?.id ?? '');
        }
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  const selectedModule = modules.find((item) => item.id === selectedModuleId);
  const registerBeforeLeave = useCallback((moduleId: string, handler: BeforeLeaveHandler | null) => {
    beforeLeaveHandlers.current[moduleId] = handler ?? undefined;
  }, []);

  const switchModule = async (nextModuleId: string) => {
    if (nextModuleId === selectedModuleId) {
      return;
    }
    setError('');
    const handler = beforeLeaveHandlers.current[selectedModuleId];
    if (handler) {
      const canLeave = await handler();
      if (!canLeave) {
        return;
      }
    }
    setSelectedModuleId(nextModuleId);
    if (nextModuleId === 'test-sheet' && view.name === 'module') {
      setView({ name: 'test-plans' });
    }
  };

  return (
    <main className="app-shell">
      <aside className="module-menu">
        <h1>ToolBox</h1>
        <nav>
          {modules.map((module) => (
            <button
              key={module.id}
              className={module.id === selectedModuleId ? 'active' : ''}
              type="button"
              onClick={() => {
                void switchModule(module.id);
              }}
            >
              {module.name}
            </button>
          ))}
        </nav>
      </aside>

      <section className="module-view">
        {error && <p className="error">{error}</p>}
        <div style={{ display: selectedModuleId === 'test-sheet' ? undefined : 'none' }}>
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
        </div>
        <div style={{ display: selectedModuleId === 'v10-lab' ? undefined : 'none' }}>
          <V10LabPage onBeforeLeaveChange={(handler) => registerBeforeLeave('v10-lab', handler)} />
        </div>
        {selectedModuleId !== 'test-sheet' && selectedModuleId !== 'v10-lab' && (
          selectedModule && (
            <>
              <header>
                <h2>{selectedModule.name}</h2>
                <p>{selectedModule.description}</p>
              </header>
              <div className="actions">
                {selectedModule.actions.map((action) => (
                  <article key={action.id}>
                    <h3>{action.name}</h3>
                    <p>{action.description}</p>
                    <button type="button">Lancer</button>
                  </article>
                ))}
              </div>
            </>
          )
        )}
      </section>
    </main>
  );
};

export default App;
