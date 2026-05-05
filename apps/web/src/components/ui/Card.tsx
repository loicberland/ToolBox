import React from 'react';

type Props = React.HTMLAttributes<HTMLElement> & {
  children: React.ReactNode;
};

export function Card({ className = '', children, ...props }: Props) {
  return (
    <section className={`ui-card ${className}`.trim()} {...props}>
      {children}
    </section>
  );
}

export function CardHeader({ className = '', children, ...props }: Props) {
  return (
    <header className={`ui-card-header ${className}`.trim()} {...props}>
      {children}
    </header>
  );
}
