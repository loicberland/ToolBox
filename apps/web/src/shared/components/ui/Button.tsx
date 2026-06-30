import React from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'warning';

type Props = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: 'sm' | 'md';
};

export function Button({ className = '', variant = 'primary', size = 'md', ...props }: Props) {
  return <button className={`ui-button ${variant} ${size} ${className}`.trim()} {...props} />;
}
