import React, { useEffect, useRef } from 'react';
import { Button } from '../../../../shared/components/ui/Button';
import { ConfirmDialog } from '../../../../shared/components/ui/ConfirmDialog';
import { messages } from '../../../../i18n';
import { RequiredDot } from '../form/RequiredDot';

const m = messages.v10Lab;
export function DuplicateMaquetteDialog({ open, name, parentPath, copyData, busy, error, onNameChange, onParentPathChange, onCopyDataChange, onCancel, onConfirm }: { open: boolean; name: string; parentPath: string; copyData: boolean; busy: boolean; error: string; onNameChange: (value: string) => void; onParentPathChange: (value: string) => void; onCopyDataChange: (value: boolean) => void; onCancel: () => void; onConfirm: () => void }) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  useEffect(() => { if (open) window.requestAnimationFrame(() => inputRef.current?.focus()); }, [open]);
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => { if (event.key === 'Escape' && !busy) onCancel(); };
    window.addEventListener('keydown', onKeyDown); return () => window.removeEventListener('keydown', onKeyDown);
  }, [busy, onCancel]);
  if (!open) return null;
  return <div className="dialog-backdrop"><div className="confirm-dialog" role="dialog" aria-modal="true" aria-label={m.duplicateTitle}>
    <h3>{m.duplicateTitle}</h3>
    <label>{m.duplicateName}<input ref={inputRef} value={name} disabled={busy} onChange={(event) => onNameChange(event.currentTarget.value)} /></label>
    <label>{m.parentPath}<input value={parentPath} disabled={busy} onChange={(event) => onParentPathChange(event.currentTarget.value)} /></label>
    <label className="duplicate-copy-data"><input type="checkbox" checked={copyData} disabled={busy} onChange={(event) => onCopyDataChange(event.currentTarget.checked)} /><span>{m.copyData}</span></label>
    {error && <p className="error">{error}</p>}
    <div className="button-row end"><Button type="button" variant="secondary" disabled={busy} onClick={onCancel}>{messages.common.cancel}</Button><Button type="button" disabled={busy || !name.trim() || !parentPath.trim()} onClick={onConfirm}>{busy ? m.duplicating : m.duplicate}</Button></div>
  </div></div>;
}


