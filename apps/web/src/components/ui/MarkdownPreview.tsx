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
        aria-label={copied ? messages.testSheet.report.copied : messages.testSheet.report.copy}
        className="copy-code-button"
        title={copied ? messages.testSheet.report.copied : messages.testSheet.report.copy}
        type="button"
        onClick={copy}
      >
        <span aria-hidden="true">{copied ? 'OK' : 'Copy'}</span>
      </button>
      <pre>{children}</pre>
    </div>
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

