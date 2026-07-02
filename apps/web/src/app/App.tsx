import React, { useCallback, useEffect, useRef, useState } from 'react';
import { fetchModules, ModuleInfo } from '../api/modules';
import { ModuleMenu } from './components/ModuleMenu';
import { TestSheetModule } from '../modules/test-sheet/TestSheetModule';
import { V10LabPage } from '../pages/V10LabPage';

export type BeforeLeaveHandler = () => Promise<boolean>;

const App = () => {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selectedModuleId, setSelectedModuleId] = useState<string>('test-sheet');
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
  };

  return (
    <main className="app-shell">
      <ModuleMenu modules={modules} selectedModuleId={selectedModuleId} onSelect={(moduleId) => { void switchModule(moduleId); }} />

      <section className="module-view">
        {error && <p className="error">{error}</p>}
        <div style={{ display: selectedModuleId === 'test-sheet' ? undefined : 'none' }}>
          <TestSheetModule />
        </div>
        <div style={{ display: selectedModuleId === 'v10-lab' ? undefined : 'none' }}>
          <V10LabPage onBeforeLeaveChange={(handler) => registerBeforeLeave('v10-lab', handler)} />
        </div>
        {selectedModuleId !== 'test-sheet' && selectedModuleId !== 'v10-lab' && (
          selectedModule && (
            <>
              <header>
                <h2>
                  {selectedModule.name}
                  {selectedModule.version && <span className="module-title-version">v{selectedModule.version}</span>}
                </h2>
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
