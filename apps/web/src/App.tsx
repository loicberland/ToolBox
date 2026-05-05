import React, { useEffect, useState } from 'react';
import { fetchModules, ModuleInfo } from './api/modules';
import { TestPlanEditPage } from './pages/TestPlanEditPage';
import { TestPlanListPage } from './pages/TestPlanListPage';
import { TestRunPage } from './pages/TestRunPage';
import { TestRunReportPage } from './pages/TestRunReportPage';

type View =
  | { name: 'module' }
  | { name: 'test-plans' }
  | { name: 'test-plan-edit'; planId: number }
  | { name: 'test-run'; runId: number }
  | { name: 'test-report'; runId: number };

const App = () => {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selectedModuleId, setSelectedModuleId] = useState<string>('test-sheet');
  const [view, setView] = useState<View>({ name: 'test-plans' });
  const [error, setError] = useState<string>('');

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
                setSelectedModuleId(module.id);
                setView(module.id === 'test-sheet' ? { name: 'test-plans' } : { name: 'module' });
              }}
            >
              {module.name}
            </button>
          ))}
        </nav>
      </aside>

      <section className="module-view">
        {error && <p className="error">{error}</p>}
        {selectedModuleId === 'test-sheet' ? (
          <>
            {view.name === 'test-plans' && (
              <TestPlanListPage
                onEdit={(planId) => setView({ name: 'test-plan-edit', planId })}
                onRun={(runId) => setView({ name: 'test-run', runId })}
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
                onReport={(runId) => setView({ name: 'test-report', runId })}
              />
            )}
            {view.name === 'test-report' && <TestRunReportPage runId={view.runId} onBack={() => setView({ name: 'test-run', runId: view.runId })} />}
          </>
        ) : (
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
