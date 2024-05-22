import React from 'react';
import './SmartEllipsisText.css';

type Props = {
  text: string;
  className?: string;
};

export function SmartEllipsisText({ text, className = '' }: Props) {
  return (
    <span className={`smart-ellipsis ${className}`.trim()} title={text}>
      {text}
    </span>
  );
}
