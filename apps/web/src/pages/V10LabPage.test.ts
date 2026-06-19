import type { V10Config, V10Product } from '../api/v10Lab';

Object.assign(globalThis, { window: { TOOLBOX: { services: { api: { url: '/api' } } } } });

const { executableCommandGroups } = require('./V10LabPage') as typeof import('./V10LabPage');

describe('executableCommandGroups', () => {
  const product: V10Product = {
    id: 'test',
    name: 'Test',
    label: 'Test',
    description: '',
    defaultAppName: 'prod',
    services: [{ name: 'auth', label: 'auth', hasDatabase: true, supportsExtraKeys: true }],
    unitSingularLabel: 'connector',
    unitPluralLabel: 'connectors',
    unitCfgSectionName: 'connectors',
    unitFolderPrefix: 'connector-',
    unitDefinitions: [
      { kind: 'connector', singularLabel: 'connector', pluralLabel: 'connectors', cfgSectionName: 'connectors', folderPrefix: 'connector-' },
      { kind: 'agent', singularLabel: 'agent', pluralLabel: 'agents', cfgSectionName: 'agents', folderPrefix: 'agent-' },
      { kind: 'adaptor', singularLabel: 'adaptor', pluralLabel: 'adaptors', cfgSectionName: 'adaptors', folderPrefix: 'adaptor-' },
    ],
  };
  const config: V10Config = {
    name: 'Demo',
    product: 'test',
    release: { zipPath: '', workDir: '', overwrite: false },
    maquette: { targetPath: '', envName: 'live', appName: 'prod' },
    gedixConfig: {
      fqdn: '', port: 80,
      services: { auth: { dbType: 'sqlite', dbDsn: '', extraKeys: {} } },
      connectors: { 'connector-filesystem-01': { rawConfig: '' } },
      agents: { 'agent-watch-01': { rawConfig: '' } },
      adaptors: { 'adaptor-digi-01': { rawConfig: '' } },
      units: {},
    },
    runtime: { debugTargets: [], debugTargetFlags: {}, openConsole: true },
    pipeline: [],
  };

  it('groups startup targets without root executables', () => {
    const groups = executableCommandGroups(config, product, false);

    expect(groups.map((group) => group.label)).toEqual(['Services', 'Connecteurs', 'Agents', 'Adaptors']);
    expect(groups.map((group) => group.options.map((option) => option.name))).toEqual([
      ['auth'], ['connector-filesystem-01'], ['agent-watch-01'], ['adaptor-digi-01'],
    ]);
  });

  it('does not render empty categories', () => {
    const groups = executableCommandGroups({ ...config, gedixConfig: { ...config.gedixConfig, agents: {}, adaptors: {} } }, product, false);

    expect(groups.map((group) => group.label)).toEqual(['Services', 'Connecteurs']);
  });
});
