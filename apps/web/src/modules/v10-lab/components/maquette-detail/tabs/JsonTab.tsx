import React from 'react';
import { ExecutionResponse } from '../../../api/v10Lab';
import { messages } from '../../../../../i18n';
import { Button } from '../../../../../shared/components/ui/Button';

const m = messages.v10Lab;

export function JsonTab({ jsonText, execution, busy, onJsonTextChange, onCopy, onApply, onValidate, onDownload }: {
  jsonText: string;
  execution: ExecutionResponse | null;
  busy: boolean;
  onJsonTextChange: (value: string) => void;
  onCopy: () => void;
  onApply: () => void;
  onValidate: () => void;
  onDownload: () => void;
}) {
  return (
    <div className="v10-json-panel">
      <textarea value={jsonText} onChange={(event) => onJsonTextChange(event.currentTarget.value)} spellCheck={false} />
      {execution?.errors?.length ? <p className="error whitespace">{execution.errors.join('\n')}</p> : null}
      {execution?.status === 'valid' && <p className="info-message">{execution.output || m.validationOk}</p>}
      <div className="button-row end">
        <Button type="button" variant="secondary" onClick={onCopy}>{m.copy}</Button>
        <Button type="button" variant="secondary" onClick={onApply}>{m.applyJsonChanges}</Button>
        <Button type="button" variant="primary" onClick={onValidate} disabled={busy}>{m.json.validateConfig}</Button>
        <Button type="button" variant="success" onClick={onDownload}>{m.downloadJson}</Button>
      </div>
    </div>
  );
}
