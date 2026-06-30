import React from 'react';
import { messages } from '../../../i18n';
import { Button } from './Button';

type Props = {
  open: boolean;
  title: string;
  message: string;
  children?: React.ReactNode;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ open, title, message, children, confirmLabel = messages.common.confirm, onConfirm, onCancel }: Props) {
  if (!open) {
    return null;
  }
  return (
    <div className="dialog-backdrop" role="presentation">
      <div className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title">
        <h3 id="confirm-dialog-title">{title}</h3>
        <p>{message}</p>
        {children}
        <div className="button-row end">
          <Button type="button" variant="secondary" onClick={onCancel}>{messages.common.cancel}</Button>
          <Button type="button" variant="danger" onClick={onConfirm}>{confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}
