import React from 'react';
import { ModuleInfo } from '../../api/modules';

type Props = {
  modules: ModuleInfo[];
  selectedModuleId: string;
  onSelect: (moduleId: string) => void;
};

export function ModuleMenu({ modules, selectedModuleId, onSelect }: Props) {
  return (
    <aside className="module-menu">
      <h1>ToolBox</h1>
      <nav>
        {modules.map((module) => (
          <button
            key={module.id}
            className={module.id === selectedModuleId ? 'active' : ''}
            type="button"
            onClick={() => onSelect(module.id)}
          >
            <span className="module-menu-name">{module.name}</span>
            {module.version && <span className="module-menu-version">v{module.version}</span>}
          </button>
        ))}
      </nav>
    </aside>
  );
}
