import React, { useState } from 'react';
import { ExportOptions, testSheetApi } from '../../api/testSheet';
import { messages } from '../../i18n';
import { Button } from '../ui/Button';

type Props = {
  planId: number;
  planName: string;
  onClose: () => void;
  onError: (message: string) => void;
};

export function TestPlanExportDialog({ planId, planName, onClose, onError }: Props) {
  const [options, setOptions] = useState<ExportOptions>({
    includeGroups: true,
    includeSheets: true,
    includeSteps: true,
    includeDocuments: true,
    includeHistory: false,
    includeEvidences: false,
  });
  const [exporting, setExporting] = useState(false);

  const setOption = (key: keyof ExportOptions, checked: boolean) => {
    setOptions((current) => normalizeExportOptions({ ...current, [key]: checked }));
  };

  const exportPlan = async () => {
    setExporting(true);
    try {
      const blob = await testSheetApi.exportPlan(planId, normalizeExportOptions(options));
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `test-sheet-plan-${planId}.zip`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      onClose();
    } catch (err) {
      onError((err as Error).message);
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="dialog-backdrop" role="presentation">
      <div className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="export-plan-title">
        <h3 id="export-plan-title">Exporter {planName}</h3>
        <div className="export-option-list">
          <label><input type="checkbox" checked readOnly /> <span>Plan</span></label>
          <label><input type="checkbox" checked={options.includeGroups} onChange={(event) => setOption('includeGroups', event.target.checked)} /> <span>Sous-plans / groupes</span></label>
          <label><input type="checkbox" checked={options.includeSheets} onChange={(event) => setOption('includeSheets', event.target.checked)} /> <span>Fiches</span></label>
          <label><input type="checkbox" checked={options.includeSteps} onChange={(event) => setOption('includeSteps', event.target.checked)} /> <span>Actions</span></label>
          <label><input type="checkbox" checked={options.includeDocuments} onChange={(event) => setOption('includeDocuments', event.target.checked)} /> <span>Documents</span></label>
          <label><input type="checkbox" checked={options.includeHistory} onChange={(event) => setOption('includeHistory', event.target.checked)} /> <span>Historique d'exécution</span></label>
          <label><input type="checkbox" checked={options.includeEvidences} onChange={(event) => setOption('includeEvidences', event.target.checked)} /> <span>Preuves / evidences</span></label>
        </div>
        <div className="button-row end">
          <Button type="button" variant="secondary" onClick={onClose}>{messages.common.cancel}</Button>
          <Button type="button" disabled={exporting} onClick={exportPlan}>{exporting ? 'Export...' : 'Exporter'}</Button>
        </div>
      </div>
    </div>
  );
}

function normalizeExportOptions(options: ExportOptions): ExportOptions {
  const next = { ...options };
  if (next.includeSteps) {
    next.includeSheets = true;
    next.includeGroups = true;
  }
  if (next.includeSheets) {
    next.includeGroups = true;
  }
  if (next.includeEvidences) {
    next.includeHistory = true;
  }
  return next;
}
