import React from 'react';

type Props = {
  eyebrow?: string;
  title: string;
  description?: string;
  backLabel?: string;
  onBack?: () => void;
  actions?: React.ReactNode;
};

export function PageHeader({ eyebrow, title, description, backLabel = 'Retour', onBack, actions }: Props) {
  return (
    <header className="page-header">
      <div className="page-title-group">
        {onBack && (
          <button className="back-button" type="button" onClick={onBack}>
            <span aria-hidden="true">&lt;</span>
            {backLabel}
          </button>
        )}
        {eyebrow && <p className="page-eyebrow">{eyebrow}</p>}
        <h2>{title}</h2>
        {description && <p>{description}</p>}
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </header>
  );
}
