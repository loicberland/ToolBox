import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import './MarkdownPreview.css';

type Props = {
  content?: string;
  compact?: boolean;
};

export function MarkdownPreview({ content, compact = false }: Props) {
  if (!hasMarkdownContent(content)) {
    return null;
  }

  return (
    <div className={compact ? 'markdown-preview markdown-preview--compact' : 'markdown-preview'}>
      <ReactMarkdown remarkPlugins={[remarkGfm]}>
        {content}
      </ReactMarkdown>
    </div>
  );
}

export function hasMarkdownContent(content?: string) {
  return Boolean(content?.trim());
}
