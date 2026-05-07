import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { messages } from '../../i18n';
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
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          pre({ children }) {
            return <CopyableCodeBlock>{children}</CopyableCodeBlock>;
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

export function hasMarkdownContent(content?: string) {
  return Boolean(content?.trim());
}

function CopyableCodeBlock({ children }: { children: React.ReactNode }) {
  const [copied, setCopied] = React.useState(false);
  const text = getNodeText(children);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch (error) {
      console.error('Unable to copy code block', error);
    }
  };

  return (
    <div className="copyable-code-block">
      <button
        aria-label={copied ? messages.testSheet.report.codeCopied : messages.testSheet.report.copyCode}
        className="copy-code-button"
        title={copied ? messages.testSheet.report.codeCopied : messages.testSheet.report.copyCode}
        type="button"
        onClick={copy}
      >
        {copied ? <CheckIcon /> : <CopyIcon />}
      </button>
      <pre>{children}</pre>
    </div>
  );
}

function CopyIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
      <rect x="8" y="8" width="10" height="12" rx="2" fill="none" stroke="currentColor" strokeWidth="2" />
      <path d="M6 16H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
      <path d="m5 12 5 5L20 7" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function getNodeText(node: React.ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node);
  }
  if (Array.isArray(node)) {
    return node.map(getNodeText).join('');
  }
  if (React.isValidElement<{ children?: React.ReactNode }>(node)) {
    return getNodeText(node.props.children);
  }
  return '';
}

