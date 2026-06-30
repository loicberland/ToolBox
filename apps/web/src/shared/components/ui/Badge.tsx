import React from 'react';

type Props = {
  children: React.ReactNode;
  tone?: 'neutral' | 'blue' | 'green' | 'red' | 'orange' | 'gray';
  className?: string;
};

export function Badge({ children, tone = 'neutral', className = '' }: Props) {
  return <span className={`ui-badge ${tone} ${className}`.trim()}>{children}</span>;
}
