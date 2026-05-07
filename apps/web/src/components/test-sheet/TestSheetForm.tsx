import React, { forwardRef, useEffect, useImperativeHandle, useState } from 'react';
import { SheetInput, TestSheet } from '../../api/testSheet';
import { messages } from '../../i18n';
import { Button } from '../ui/Button';
import { MarkdownTextarea } from '../ui/MarkdownTextarea';

type Props = {
  sheet?: TestSheet;
  nextOrder: number;
  onSubmit: (input: SheetInput) => Promise<void>;
  onCancel?: () => void;
  formId?: string;
  hideActions?: boolean;
};

export type TestSheetFormHandle = {
  submit: () => Promise<void>;
};

export const TestSheetForm = forwardRef<TestSheetFormHandle, Props>(function TestSheetForm({ sheet, nextOrder, onSubmit, onCancel, formId, hideActions = false }, ref) {
  const [value, setValue] = useState<SheetInput>(newSheet(nextOrder));
  const [saving, setSaving] = useState(false);
  const isEditing = Boolean(sheet?.id);

  useEffect(() => {
    setValue(sheet ? {
      name: sheet.name,
      description: sheet.description,
      prerequisites: sheet.prerequisites,
      config: sheet.config,
      command: sheet.command,
      notes: sheet.notes,
      action: sheet.action,
      expectedResult: sheet.expectedResult,
      executionOrder: sheet.executionOrder,
      mockupSettings: sheet.mockupSettings,
    } : newSheet(nextOrder));
  }, [sheet, nextOrder]);

  const submitCurrent = async () => {
    setSaving(true);
    try {
      await onSubmit(value);
    } finally {
      setSaving(false);
    }
  };

  useImperativeHandle(ref, () => ({
    submit: submitCurrent,
  }));

  return (
    <form
      id={formId}
      className="form-grid sheet-form"
      onSubmit={async (event) => {
        event.preventDefault();
        await submitCurrent();
      }}
    >
      <label>
        {messages.testSheet.edit.name}
        <input value={value.name} onChange={(event) => setValue({ ...value, name: event.target.value })} required />
      </label>
      <label>
        {messages.testSheet.edit.description}
        <textarea value={value.description} onChange={(event) => setValue({ ...value, description: event.target.value })} />
      </label>
      <MarkdownTextarea
        label={messages.testSheet.edit.prerequisites}
        value={value.prerequisites}
        onChange={(prerequisites) => setValue({ ...value, prerequisites })}
      />
      <MarkdownTextarea
        label={messages.testSheet.edit.configuration}
        value={value.config}
        onChange={(config) => setValue({ ...value, config })}
      />
      <MarkdownTextarea
        label={messages.testSheet.edit.command}
        value={value.command}
        onChange={(command) => setValue({ ...value, command })}
      />
      <MarkdownTextarea
        label={messages.testSheet.edit.notes}
        value={value.notes}
        onChange={(notes) => setValue({ ...value, notes })}
      />
      {!hideActions && (
        <div className="button-row">
          <Button type="submit" disabled={saving}>{saving ? messages.common.saving : isEditing ? messages.common.save : messages.common.add}</Button>
          {onCancel && <Button variant="secondary" type="button" onClick={onCancel}>{messages.common.cancel}</Button>}
        </div>
      )}
    </form>
  );
});

function newSheet(order: number): SheetInput {
  return {
    name: '',
    description: '',
    prerequisites: '',
    config: '',
    command: '',
    notes: '',
    action: '',
    expectedResult: '',
    executionOrder: order,
    mockupSettings: '',
  };
}
