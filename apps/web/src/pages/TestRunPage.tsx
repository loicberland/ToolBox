import React, { useEffect, useState } from 'react';
import { RunGroup, testSheetApi, TestDocument, TestRun, TestRunSheet } from '../api/testSheet';
import { DocumentList } from '../components/test-sheet/DocumentList';
import { TestRunProgress } from '../components/test-sheet/TestRunProgress';
import { TestRunSheetDetail } from '../components/test-sheet/TestRunSheetDetail';
import { TestRunSheetList } from '../components/test-sheet/TestRunSheetList';
import { Button } from '../components/ui/Button';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { PageHeader } from '../components/ui/PageHeader';
import { StatusBadge } from '../components/test-sheet/StatusBadge';
import { getGroupStatus, getRunSheetProgress, isRunEditable, isRunReadOnly } from '../components/test-sheet/runStatus';
import { messages } from '../i18n';

type Props = {
  runId: number;
  onBack: () => void;
  onReport: (runId: number) => void;
};

export function TestRunPage({ runId, onBack, onReport }: Props) {
  const [run, setRun] = useState<TestRun | undefined>();
  const [error, setError] = useState('');
  const [selectedSheetId, setSelectedSheetId] = useState<number | undefined>();
  const [selectedGroupId, setSelectedGroupId] = useState<number | undefined>();
  const [confirmFinish, setConfirmFinish] = useState(false);

  const load = () => testSheetApi.getRun(runId).then(setRun).catch((err: Error) => setError(err.message));

  useEffect(() => {
    load();
  }, [runId]);

  useEffect(() => {
    const groups = getRunGroups(run);
    if (!groups.length) {
      setSelectedGroupId(undefined);
      setSelectedSheetId(undefined);
      return;
    }

    const selectedGroupStillExists = groups.some((group) => group.id === selectedGroupId);
    const nextGroup = selectedGroupStillExists
      ? groups.find((group) => group.id === selectedGroupId)!
      : (groups.find((group) => getGroupStatus(group.sheets ?? []) === 'pending') ?? groups[0]);
    if (nextGroup.id !== selectedGroupId) {
      setSelectedGroupId(nextGroup.id);
    }
    const sheets = nextGroup.sheets ?? [];
    const selectedSheetStillExists = sheets.some((sheet) => sheet.id === selectedSheetId);
    if (selectedSheetStillExists) {
      return;
    }

    const firstPending = sheets.find((sheet) => getRunSheetProgress(sheet).status === 'pending');
    setSelectedSheetId((firstPending ?? sheets[0])?.id);
  }, [run, selectedGroupId, selectedSheetId]);

  const runGroups = getRunGroups(run);
  const selectedGroup = runGroups.find((group) => group.id === selectedGroupId);
  const visibleSheets = selectedGroup?.sheets ?? [];
  const selectedSheet = visibleSheets.find((sheet) => sheet.id === selectedSheetId);
  const readOnly = run ? isRunReadOnly(run.status) : false;
  const runEditable = run ? isRunEditable(run.status) : false;
  const finish = async () => {
    if (!run || !runEditable) {
      return;
    }
    if (hasPendingWork(run)) {
      setConfirmFinish(true);
      return;
    }
    await testSheetApi.finishRun(runId);
    await load();
  };

  return (
    <section className="workspace">
      <PageHeader
        eyebrow={messages.testSheet.run.eyebrow}
        title={run ? `${messages.testSheet.plans.executionNumber}${run.runNumber} - ${run.planName}` : messages.testSheet.run.eyebrow}
        description={selectedSheet ? `${messages.testSheet.run.selectedTest} : ${selectedSheet.name}` : undefined}
        onBack={onBack}
        actions={(
          <div className="button-row">
            {run && <StatusBadge status={run.status} />}
            <Button variant="secondary" type="button" onClick={() => onReport(runId)}>{messages.testSheet.run.report}</Button>
            {runEditable && <Button type="button" onClick={finish}>{messages.testSheet.run.finish}</Button>}
          </div>
        )}
      />
      {error && <p className="error">{error}</p>}
      {run && (
        <div className="test-run-execution">
          <div className="test-run-progress">
            <TestRunProgress status={run.status} sheets={run.sheets} groups={runGroups} />
          </div>
          <div className="test-run-layout">
            <div className="test-run-sidebar">
              <RunGroupList
                groups={runGroups}
                selectedGroupId={selectedGroupId}
                onSelect={(group) => {
                  setSelectedGroupId(group.id);
                  const sheets = group.sheets ?? [];
                  const firstPending = sheets.find((sheet) => getRunSheetProgress(sheet).status === 'pending');
                  setSelectedSheetId((firstPending ?? sheets[0])?.id);
                }}
              />
              {selectedGroup && <SelectedGroupProgress group={selectedGroup} />}
              <TestRunSheetList sheets={visibleSheets} selectedSheetId={selectedSheetId} onSelect={setSelectedSheetId} />
              {selectedSheet && selectedGroup && (
                <RunSheetDocumentsCard
                  documents={selectedSheet.documents ?? []}
                  zipFilename={buildDocumentsZipFilename(run.planName, selectedGroup.name, selectedSheet.name)}
                />
              )}
            </div>
            <aside className="test-run-detail">
              {selectedSheet && (
                <TestRunSheetDetail
                  sheet={selectedSheet}
                  readOnly={readOnly}
                  onSaveSheet={async (sheetId, input) => {
                    if (readOnly) {
                      return;
                    }
                    await testSheetApi.updateRunSheet(runId, sheetId, input);
                    await load();
                  }}
                  onSaveStep={async (stepId, input) => {
                    if (readOnly) {
                      return;
                    }
                    await testSheetApi.updateRunStep(runId, stepId, input);
                    await load();
                  }}
                  onEvidenceChanged={load}
                />
              )}
            </aside>
          </div>
        </div>
      )}
      <ConfirmDialog
        open={confirmFinish}
        title={messages.testSheet.run.finishTitle}
        message={messages.testSheet.run.finishMessage}
        confirmLabel={messages.testSheet.run.finishAnyway}
        onCancel={() => setConfirmFinish(false)}
        onConfirm={async () => {
          if (!runEditable) {
            setConfirmFinish(false);
            return;
          }
          await testSheetApi.finishRun(runId);
          setConfirmFinish(false);
          await load();
        }}
      />
    </section>
  );
}

function RunSheetDocumentsCard({ documents, zipFilename }: { documents: TestDocument[]; zipFilename: string }) {
  const hasDocuments = documents.length > 0;

  return (
    <div className="run-sheet-list-card ui-card">
      <div className="ui-card-header">
        <div>
          <span className="section-kicker">{messages.testSheet.run.sheetDocuments}</span>
          <h3>{documents.length} document{documents.length > 1 ? 's' : ''}</h3>
        </div>
        {hasDocuments && (
          <a
            className="ui-button secondary sm"
            href={testSheetApi.documentsZipDownloadUrl(documents.map((document) => document.id), zipFilename)}
          >
            {messages.testSheet.documents.downloadAll}
          </a>
        )}
      </div>
      <DocumentList
        documents={documents}
        emptyText={messages.testSheet.documents.noSheetDocument}
      />
    </div>
  );
}

function buildDocumentsZipFilename(planName: string, groupName: string, sheetName: string) {
  return `documents-${slugFilenamePart(planName)}-${slugFilenamePart(groupName)}-${slugFilenamePart(sheetName)}.zip`;
}

function slugFilenamePart(value: string) {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '') || 'sans-nom';
}

function SelectedGroupProgress({ group }: { group: RunGroup }) {
  const sheets = group.sheets ?? [];
  const done = sheets.filter((sheet) => getRunSheetProgress(sheet).status !== 'pending').length;
  const total = sheets.length;
  const percent = total === 0 ? 0 : Math.round((done / total) * 100);
  return (
    <div className="run-sheet-list-card ui-card">
      <div className="ui-card-header">
        <div>
          <span className="section-kicker">Progression du sous-plan</span>
          <h3>{group.name}</h3>
        </div>
        <StatusBadge status={getGroupStatus(sheets)} />
      </div>
      <strong>{done} / {total} fiches traitées</strong>
      <div className="progress-track" aria-label={`Progression ${percent}%`}>
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

function hasPendingWork(run: TestRun) {
  return run.sheets.some((sheet) => {
    const steps = sheet.steps ?? [];
    if (steps.length === 0) {
      return sheet.status === 'pending';
    }
    return steps.some((step) => step.status === 'pending');
  });
}

function getRunGroups(run?: TestRun): RunGroup[] {
  if (!run) {
    return [];
  }
  if (run.groups && run.groups.length > 0) {
    return run.groups;
  }
  return [{
    id: 0,
    runId: run.id,
    name: run.groupName || run.planName,
    description: '',
    executionOrder: 1,
    createdAt: run.startedAt,
    sheets: run.sheets,
  }];
}

function RunGroupList({
  groups,
  selectedGroupId,
  onSelect,
}: {
  groups: RunGroup[];
  selectedGroupId?: number;
  onSelect: (group: RunGroup) => void;
}) {
  return (
    <div className="run-sheet-list-card ui-card">
      <div className="ui-card-header">
        <div>
          <span className="section-kicker">Sous-plans</span>
          <h3>{groups.length} sous-plan{groups.length > 1 ? 's' : ''}</h3>
        </div>
      </div>
      <div className="run-sheet-list">
        {groups.map((group) => {
          const sheets = group.sheets ?? [];
          const done = sheets.filter((sheet) => getRunSheetProgress(sheet).status !== 'pending').length;
          return (
            <div
              key={group.id}
              className={`run-sheet-list-item ${group.id === selectedGroupId ? 'active' : ''}`}
              role="button"
              tabIndex={0}
              onClick={() => onSelect(group)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  onSelect(group);
                }
              }}
            >
              <span className="run-list-order">{group.executionOrder}</span>
              <div className="run-sheet-list-main">
                <div className="run-sheet-list-title">
                  <strong className="run-sheet-name">{group.name}</strong>
                  <StatusBadge status={getGroupStatus(sheets)} />
                </div>
                <div className="run-sheet-progress-summary">{done} / {sheets.length} fiches traitées</div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
