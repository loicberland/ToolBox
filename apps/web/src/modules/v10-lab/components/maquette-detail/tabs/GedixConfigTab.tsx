import React from 'react';
import { V10Config } from '../../../api/v10Lab';
import { messages } from '../../../../../i18n';

const m = messages.v10Lab;
export function GedixForm({ config, onChange, compact = false }: { config: V10Config; onChange: (config: V10Config) => void; compact?: boolean }) {
  const content = (
    <>
      <label>FQDN
        <input value={config.gedixConfig.fqdn} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, fqdn: event.currentTarget.value } })} />
      </label>
      <label>Port
        <input type="number" min={0} max={65535} value={config.gedixConfig.port} onChange={(event) => onChange({ ...config, gedixConfig: { ...config.gedixConfig, port: Number(event.currentTarget.value) } })} />
      </label>
    </>
  );
  return compact ? content : <div className="form-grid v10-form-grid">{content}</div>;
}

export const GedixConfigTab = GedixForm;

