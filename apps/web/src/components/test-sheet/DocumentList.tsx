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
          <div>
            <strong>{document.originalName}</strong>
            <p className="muted">{document.mimeType || 'Type inconnu'} - {formatBytes(document.sizeBytes)}</p>
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
