import React from 'react';
import { testSheetApi, TestDocument } from '../../api/testSheet';
import { Button } from '../ui/Button';

type Props = {
  documents?: TestDocument[];
  emptyText?: string;
  onRemove?: (document: TestDocument) => void;
  onDelete?: (document: TestDocument) => void;
};

export function DocumentList({ documents = [], emptyText = 'Aucun document', onRemove, onDelete }: Props) {
  if (documents.length === 0) {
    return <p className="muted">{emptyText}</p>;
  }

  return (
    <div className="document-list">
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
            <a className="ui-button secondary sm" href={testSheetApi.documentDownloadUrl(document.id)}>Telecharger</a>
            {onRemove && <Button type="button" size="sm" variant="secondary" onClick={() => onRemove(document)}>Retirer</Button>}
            {onDelete && <Button type="button" size="sm" variant="danger" onClick={() => onDelete(document)}>Supprimer</Button>}
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
};

export function DocumentFilePicker({ id, file, inputRef, onFileChange, label = 'Choisir un fichier' }: DocumentFilePickerProps) {
  return (
    <div className="document-file-picker">
      <input
        ref={inputRef}
        id={id}
        className="document-file-input"
        type="file"
        onChange={(event) => onFileChange(event.target.files?.[0])}
      />
      <label className="ui-button secondary document-file-button" htmlFor={id}>
        {label}
      </label>
      <span className={file ? 'document-file-name' : 'document-file-name muted'}>
        {file?.name ?? 'aucun fichier selectionne'}
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
