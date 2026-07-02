import React from 'react';
import { testSheetApi, TestDocument } from '../../api/testSheet';
import { messages } from '../../../../i18n';
import { Button } from '../../../../shared/components/ui/Button';

type Props = {
  documents?: TestDocument[];
  emptyText?: string;
  compact?: boolean;
  onRemove?: (document: TestDocument) => void;
  onDelete?: (document: TestDocument) => void;
};

export function DocumentList({ documents = [], emptyText = messages.testSheet.documents.noDocument, compact = false, onRemove, onDelete }: Props) {
  if (documents.length === 0) {
    return <p className="muted">{emptyText}</p>;
  }

  return (
    <div className={`document-list ${compact ? 'compact' : ''}`.trim()}>
      {documents.map((document) => (
        <div className="document-list-item" key={document.id}>
          <div className="document-content">
            <div className="document-title-row">
              <span className="document-name" title={document.originalName}>{document.originalName}</span>
              <span className="document-size">{formatBytes(document.sizeBytes)}</span>
            </div>
            {document.description.trim() && (
              <div className="document-description">{document.description}</div>
            )}
          </div>
          <div className="button-row end">
            <a className="ui-button secondary sm" href={testSheetApi.documentDownloadUrl(document.id)}>{messages.common.download}</a>
            {onRemove && <Button type="button" size="sm" variant="secondary" onClick={() => onRemove(document)}>{messages.common.remove}</Button>}
            {onDelete && <Button type="button" size="sm" variant="danger" onClick={() => onDelete(document)}>{messages.common.delete}</Button>}
          </div>
        </div>
      ))}
    </div>
  );
}

type DocumentFilePickerProps = {
  id: string;
  file?: File;
  inputRef: React.RefObject<HTMLInputElement>;
  onFileChange: (file?: File) => void;
  label?: string;
  accept?: string;
};

export function DocumentFilePicker({ id, file, inputRef, onFileChange, label = messages.testSheet.documents.chooseFile, accept }: DocumentFilePickerProps) {
  return (
    <div className="document-file-picker">
      <input
        ref={inputRef}
        id={id}
        className="document-file-input"
        type="file"
        accept={accept}
        onChange={(event) => onFileChange(event.currentTarget.files?.[0])}
      />
      <label className="ui-button secondary document-file-button" htmlFor={id}>
        {label}
      </label>
      <span className={file ? 'document-file-name' : 'document-file-name muted'}>
        {file?.name ?? messages.testSheet.documents.noFileSelected}
      </span>
    </div>
  );
}

export function formatBytes(value: number) {
  if (!value) {
    return '0 o';
  }
  const units = ['o', 'Ko', 'Mo', 'Go'];
  let size = value;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}
