import React from 'react';
import { Button } from './Button';

type Props = {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ open, title, message, confirmLabel = 'Confirmer', onConfirm, onCancel }: Props) {
  if (!open) {
    return null;
  }
  return (
    <div className="dialog-backdrop" role="presentation">
      <div className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title">
        <h3 id="confirm-dialog-title">{title}</h3>
        <p>{message}</p>
        <div className="button-row end">
          <Button type="button" variant="secondary" onClick={onCancel}>Annuler</Button>
          <Button type="button" variant="danger" onClick={onConfirm}>{confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}
