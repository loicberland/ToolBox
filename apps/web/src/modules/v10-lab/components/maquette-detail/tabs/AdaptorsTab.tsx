import React from 'react';
import { UnitKind, V10Config, V10Product } from '../../../api/v10Lab';
import { UnitsForm } from '../../units/UnitsTab';

export function AdaptorsTab({ config, product, onChange, onScanCfg }: { config: V10Config; product: V10Product; onChange: (config: V10Config) => void; onScanCfg: (unitKind: UnitKind, file: File, importExistingKeys: boolean, replaceExistingUnits: boolean) => void }) {
  return <UnitsForm config={config} product={product} unitKind="adaptor" onChange={onChange} onScanCfg={onScanCfg} />;
}
