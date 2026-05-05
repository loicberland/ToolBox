import React, { useEffect, useState } from 'react';
import { fetchModules, ModuleInfo } from './api/modules';

const App = () => {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selectedModuleId, setSelectedModuleId] = useState<string>('');
  const [error, setError] = useState<string>('');

  useEffect(() => {
    fetchModules()
      .then((items) => {
        setModules(items);
        setSelectedModuleId(items[0]?.id ?? '');
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
              onClick={() => setSelectedModuleId(module.id)}
            >
              {module.name}
            </button>
          ))}
        </nav>
      </aside>

      <section className="module-view">
        {error && <p className="error">{error}</p>}
        {selectedModule && (
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
        )}
      </section>
    </main>
  );
};

export default App;
