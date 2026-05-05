import React from 'react';

type Props = {
  markdown: string;
};

export function ReportPreview({ markdown }: Props) {
  return <pre className="report-preview">{markdown}</pre>;
}
