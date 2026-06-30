import type { V10Action, V10Config, V10Product } from './api/v10Lab';
import fr from '../../i18n/fr.json';
import { readFileSync } from 'fs';
import { join } from 'path';

Object.assign(globalThis, { window: { TOOLBOX: { services: { api: { url: '/api' } } } } });

const { executableCommandGroups, maquetteJSONFileName, normalizeActionParamsForSave, normalizePipelineStepsForActionDefinitions, prettyJSONForDownload } = require('./V10LabModule') as typeof import('./V10LabModule');

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

describe('DuplicateMaquetteDialog', () => {
  it('renders a checked copy-data checkbox before its text in the horizontal label', () => {
    const source = readFileSync(join(__dirname, 'components/dialogs/DuplicateMaquetteDialog.tsx'), 'utf8');
    expect(source).toMatch(/<label className="duplicate-copy-data"><input type="checkbox" checked=\{copyData\}/);
    expect(source).toMatch(/<input type="checkbox" checked=\{copyData\}[\s\S]*?<span>\{m\.copyData\}<\/span><\/label>/);
  });
});

describe('action conditional fields normalization', () => {
  const action: V10Action = {
    id: 'create-machine',
    label: 'Machine',
    description: '',
    kind: 'api',
    products: ['gedix-prod-v10'],
    fields: [
      { name: 'entity_name', label: 'Nom', type: 'string', required: true, default: 'Machine', description: '' },
      { name: 'has_command_program', label: 'Programme', type: 'bool', required: false, default: false, description: '' },
      { name: 'command_program_name', label: 'Nom programme', type: 'string', required: false, default: '', description: '', hiddenWhen: { has_command_program: false } },
      { name: 'command_program_regexp', label: 'Regex', type: 'string', required: false, default: '', description: '', hiddenWhen: { has_command_program: false } },
      { name: 'description', label: 'Description', type: 'string', required: false, default: '', description: '' },
      {
        name: 'rows',
        label: 'Rows',
        type: 'object[]',
        required: false,
        default: [],
        description: '',
        itemFields: [
          { name: 'enabled', label: 'Enabled', type: 'bool', required: false, default: false, description: '' },
          { name: 'detail', label: 'Detail', type: 'string', required: false, default: '', description: '', hiddenWhen: { enabled: false } },
        ],
      },
    ],
  };

  it('keeps visible conditional values and unrelated fields', () => {
    expect(normalizeActionParamsForSave(action, {
      entity_name: 'Machine',
      has_command_program: true,
      command_program_name: 'CMD',
      command_program_regexp: 'LOAD',
      description: 'kept',
    })).toEqual({
      entity_name: 'Machine',
      has_command_program: true,
      command_program_name: 'CMD',
      command_program_regexp: 'LOAD',
      description: 'kept',
    });
  });

  it('removes every hidden value without failing on missing hidden fields', () => {
    expect(normalizeActionParamsForSave(action, {
      entity_name: 'Machine',
      has_command_program: false,
      command_program_name: 'old',
      command_program_regexp: 'old-regexp',
      description: 'kept',
    })).toEqual({
      entity_name: 'Machine',
      has_command_program: false,
      description: 'kept',
    });
  });

  it('does not restore old hidden values when the checkbox is checked again later', () => {
    const unchecked = normalizeActionParamsForSave(action, {
      entity_name: 'Machine',
      has_command_program: false,
      command_program_name: 'old',
    });

    expect(normalizeActionParamsForSave(action, { ...unchecked, has_command_program: true })).toEqual({
      entity_name: 'Machine',
      has_command_program: true,
    });
  });

  it('normalizes nested conditional fields in object arrays', () => {
    expect(normalizeActionParamsForSave(action, {
      entity_name: 'Machine',
      rows: [
        { enabled: true, detail: 'visible' },
        { enabled: false, detail: 'hidden' },
      ],
    })).toEqual({
      entity_name: 'Machine',
      rows: [
        { enabled: true, detail: 'visible' },
        { enabled: false },
      ],
    });
  });

  it('normalizes imported or exported pipeline steps with matching action definitions', () => {
    expect(normalizePipelineStepsForActionDefinitions([{
      action: 'create-machine',
      label: 'Machine',
      params: { entity_name: 'Machine', has_command_program: false, command_program_name: 'old' },
    }], [action])[0].params).toEqual({ entity_name: 'Machine', has_command_program: false });
  });
});

describe('JSON maquette import UI', () => {
  it('uses the translated success message after an import', () => {
    const source = readFileSync(join(__dirname, 'V10LabModule.tsx'), 'utf8');
    expect(fr.v10Lab.importJSON.imported).toBe('Maquette importée.');
    expect(source).toContain('setMessage(m.importJSON.imported)');
  });
});

describe('JSON maquette download', () => {
  it('uses the revised JSON action labels in the expected order', () => {
    const source = readFileSync(join(__dirname, 'components/maquette-detail/tabs/JsonTab.tsx'), 'utf8');
    expect(fr.v10Lab).not.toHaveProperty('saveJson');
    expect(fr.v10Lab.applyJsonChanges).toBe('Appliquer les modifications');
    expect(fr.v10Lab.downloadJson).toBe('Enregistrer le JSON');
    expect(source).toMatch(/m\.applyJsonChanges[\s\S]*?m\.json\.validateConfig[\s\S]*?m\.downloadJson/);
  });

  it('formats the current editor JSON with two-space indentation', () => {
    expect(prettyJSONForDownload('{"name":"Demo","release":{"zipPath":""}}')).toBe('{\n  "name": "Demo",\n  "release": {\n    "zipPath": ""\n  }\n}');
    expect(() => prettyJSONForDownload('{')).toThrow();
  });

  it('uses the current maquette name as a Windows-safe JSON filename', () => {
    expect(maquetteJSONFileName('Demo Prod')).toBe('Demo Prod.json');
    expect(maquetteJSONFileName('Samson_copie.json')).toBe('Samson_copie.json');
    expect(maquetteJSONFileName('Demo: test?.json')).toBe('Demo- test-.json');
  });

  it('downloads through a JSON Blob without updating the maquette', () => {
    const source = readFileSync(join(__dirname, 'V10LabModule.tsx'), 'utf8');
    const download = source.match(/function downloadJSON\(\)[\s\S]*?async function selectReleaseZip/)?.[0] ?? '';
    expect(download).toContain("type: 'application/json;charset=utf-8'");
    expect(download).toContain('window.URL.createObjectURL(blob)');
    expect(download).toContain('window.URL.revokeObjectURL(url)');
    expect(download).toContain('m.jsonDownloaded');
    expect(download).not.toContain('updateMaquette');
  });
});
