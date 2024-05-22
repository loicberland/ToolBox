import React from 'react';
import { hasMarkdownContent, MarkdownPreview } from './MarkdownPreview';

type Props = {
  title: string;
  content?: string;
  defaultOpen?: boolean;
  compact?: boolean;
};

export function MarkdownCollapsibleSection({ title, content, defaultOpen = true, compact = false }: Props) {
  if (!hasMarkdownContent(content)) {
    return null;
  }

  return (
    <details className="markdown-collapsible-section" open={defaultOpen}>
      <summary>{title}</summary>
      <MarkdownPreview content={content} compact={compact} />
    </details>
  );
}
