import React from 'react';

type ToastType = 'error' | 'info';

type Props = {
  message: string;
  type: ToastType;
  onClose: () => void;
};

export function Toast({ message, type, onClose }: Props) {
  if (!message) {
    return null;
  }
  return (
    <div className={`toast ${type}`} role="alert" aria-live="assertive">
      <div className="toast-content">{message}</div>
      <button type="button" className="toast-close" aria-label="Fermer la notification" onClick={onClose}>×</button>
    </div>
  );
}
