import React, { useRef } from 'react';
import { messages } from '../../i18n';

type MarkdownFormat = 'bold' | 'italic' | 'underline' | 'strike' | 'link' | 'inlineCode' | 'codeblock' | 'quote';

type MarkdownTextareaProps = {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  rows?: number;
  className?: string;
  disabled?: boolean;
  readOnly?: boolean;
  required?: boolean;
};

const toolbarItems: Array<{ format: MarkdownFormat; label: string; title: string }> = [
  { format: 'bold', label: 'B', title: messages.markdownToolbar.bold },
  { format: 'italic', label: 'I', title: messages.markdownToolbar.italic },
  { format: 'underline', label: 'U', title: messages.markdownToolbar.underline },
  { format: 'strike', label: 'S', title: messages.markdownToolbar.strike },
  { format: 'link', label: 'Lien', title: messages.markdownToolbar.link },
  { format: 'inlineCode', label: '`', title: messages.markdownToolbar.inlineCode },
  { format: 'codeblock', label: '</>', title: messages.markdownToolbar.codeblock },
  { format: 'quote', label: '"', title: messages.markdownToolbar.quote },
];

export function MarkdownTextarea({
  label,
  value,
  onChange,
  rows,
  className = '',
  disabled = false,
  readOnly = false,
  required = false,
}: MarkdownTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const textareaId = React.useId();
  const isEditable = !disabled && !readOnly;

  const applyFormat = (format: MarkdownFormat) => {
    const textarea = textareaRef.current;
    if (!textarea || !isEditable) {
      return;
    }
    applyMarkdownFormat(textarea, value, onChange, format);
  };

  return (
    <div className={`markdown-field ${className}`.trim()}>
      {label && <label className="markdown-field-label" htmlFor={textareaId}>{label}</label>}
      <div className="markdown-textarea">
        {isEditable && (
          <div className="markdown-toolbar" aria-label="Formatage Markdown">
            {toolbarItems.map((item) => (
              <button
                key={item.format}
                type="button"
                className={`markdown-toolbar-button markdown-toolbar-button--${item.format}`}
                title={item.title}
                aria-label={item.title}
                onMouseDown={(event) => {
                  event.preventDefault();
                }}
                onClick={(event) => {
                  event.preventDefault();
                  applyFormat(item.format);
                }}
              >
                {item.label}
              </button>
            ))}
          </div>
        )}
        <textarea
          id={textareaId}
          ref={textareaRef}
          value={value}
          rows={rows}
          disabled={disabled}
          readOnly={readOnly}
          required={required}
          onChange={(event) => onChange(event.target.value)}
        />
      </div>
    </div>
  );
}

function applyMarkdownFormat(
  textarea: HTMLTextAreaElement,
  value: string,
  onChange: (next: string) => void,
  format: MarkdownFormat,
) {
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const selected = value.slice(start, end);
  const before = value.slice(0, start);
  const after = value.slice(end);
  const replacement = markdownReplacement(format, selected);
  const next = `${before}${replacement.text}${after}`;

  onChange(next);

  requestAnimationFrame(() => {
    textarea.focus();
    const selectionStart = start + replacement.selectionStart;
    const selectionEnd = start + replacement.selectionEnd;
    textarea.setSelectionRange(selectionStart, selectionEnd);
  });
}

function markdownReplacement(format: MarkdownFormat, selected: string) {
  const hasSelection = selected.length > 0;

  switch (format) {
    case 'bold':
      return wrapSelection(selected, '**', '**');
    case 'italic':
      return wrapSelection(selected, '*', '*');
    case 'underline':
      return wrapSelection(selected, '<u>', '</u>');
    case 'strike':
      return wrapSelection(selected, '~~', '~~');
    case 'link':
      if (hasSelection) {
        if (/^https?:\/\//i.test(selected)) {
          const text = `[](${selected})`;
          return { text, selectionStart: 1, selectionEnd: 1 };
        }
        const text = `[${selected}](url)`;
        return { text, selectionStart: text.length - 4, selectionEnd: text.length - 1 };
      }
      return { text: '[](url)', selectionStart: 1, selectionEnd: 1 };
    case 'inlineCode':
      return wrapSelection(selected, '`', '`');
    case 'codeblock': {
      const content = hasSelection ? selected : '';
      const text = `\`\`\`\n${content}\n\`\`\``;
      const cursor = '```\n'.length;
      return {
        text,
        selectionStart: hasSelection ? cursor : cursor,
        selectionEnd: hasSelection ? cursor + selected.length : cursor,
      };
    }
    case 'quote':
      if (hasSelection) {
        const text = selected.split('\n').map((line) => `> ${line}`).join('\n');
        return { text, selectionStart: 0, selectionEnd: text.length };
      }
      return { text: '> ', selectionStart: 2, selectionEnd: 2 };
    default:
      return { text: selected, selectionStart: 0, selectionEnd: selected.length };
  }
}

function wrapSelection(selected: string, prefix: string, suffix: string) {
  const text = `${prefix}${selected}${suffix}`;
  const selectionStart = prefix.length;
  const selectionEnd = prefix.length + selected.length;
  return { text, selectionStart, selectionEnd };
}
