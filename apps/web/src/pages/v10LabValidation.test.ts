import type { V10Config, V10Product } from '../api/v10Lab';
import { isServiceDsnRequired, validateServiceDsns } from './v10LabValidation';

describe('isServiceDsnRequired', () => {
  test.each([
    ['sqlite', false],
    [' SQLite ', false],
    ['', false],
    ['mysql', true],
    ['postgres', true],
    ['mssql', true],
    ['oracle', true],
  ])('%s -> %s', (dbType, expected) => {
    expect(isServiceDsnRequired(dbType)).toBe(expected);
  });
});

describe('validateServiceDsns', () => {
  const product: V10Product = {
    id: 'gedix-prod-v10',
    name: 'Gedix Prod V10',
    label: 'Gedix Prod V10',
    description: '',
    defaultAppName: 'prod',
    services: [
      { name: 'auth', label: 'auth', hasDatabase: true, supportsExtraKeys: true },
      { name: 'document', label: 'gx-document', hasDatabase: true, supportsExtraKeys: true },
      { name: 'filestore', label: 'filestore', hasDatabase: true, supportsExtraKeys: true },
    ],
    unitSingularLabel: 'connector',
    unitPluralLabel: 'connectors',
    unitCfgSectionName: 'connectors',
    unitFolderPrefix: 'connector-',
  };

  it('accepts SQLite without DSN', () => {
    expect(validateServiceDsns(configWithServices({
      auth: { dbType: 'sqlite', dbDsn: '', extraKeys: {} },
    }), product)).toBe('');
  });

  it('rejects PostgreSQL without DSN', () => {
    expect(validateServiceDsns(configWithServices({
      document: { dbType: 'postgres', dbDsn: '', extraKeys: {} },
    }), product)).toBe('Service "gx-document" : le champ DSN est obligatoire pour le type de base "postgres".');
  });

  it('returns the first invalid service among many services', () => {
    expect(validateServiceDsns(configWithServices({
      auth: { dbType: 'sqlite', dbDsn: '', extraKeys: {} },
      document: { dbType: 'postgres', dbDsn: '', extraKeys: {} },
      filestore: { dbType: 'mysql', dbDsn: 'server=localhost', extraKeys: {} },
    }), product)).toContain('gx-document');
  });
});

function configWithServices(services: V10Config['gedixConfig']['services']): V10Config {
  return {
    name: 'Demo',
    product: 'gedix-prod-v10',
    release: { zipPath: '', workDir: '', overwrite: false },
    maquette: { targetPath: '', envName: 'live', appName: 'prod' },
    gedixConfig: {
      fqdn: '',
      port: 80,
      services,
      connectors: {},
      agents: {},
      adaptors: {},
      units: {},
    },
    runtime: { debugTargets: [], debugTargetFlags: {}, openConsole: true },
    pipeline: [],
  };
}
