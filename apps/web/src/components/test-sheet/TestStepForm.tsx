import React, { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
import { StepInput, TestSheetStep } from '../../api/testSheet';
import { messages } from '../../i18n';
import { Button } from '../ui/Button';
import { MarkdownTextarea } from '../ui/MarkdownTextarea';

type Props = {
  step?: TestSheetStep;
  nextOrder: number;
  onSubmit: (input: StepInput) => Promise<void>;
  onSubmitAndCreateAnother?: (input: StepInput) => Promise<void>;
  onCancel?: () => void;
};

export type TestStepFormHandle = {
  submit: () => Promise<void>;
};

export const TestStepForm = forwardRef<TestStepFormHandle, Props>(function TestStepForm({ step, nextOrder, onSubmit, onSubmitAndCreateAnother, onCancel }, ref) {
  const formRef = useRef<HTMLFormElement | null>(null);
  const [value, setValue] = useState<StepInput>(newStep(nextOrder));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setValue(step ? {
      action: step.action,
      field: step.field,
      expectedResult: step.expectedResult,
      executionOrder: step.executionOrder,
    } : newStep(nextOrder));
  }, [step, nextOrder]);

  const focusFirstField = () => {
    requestAnimationFrame(() => {
      formRef.current?.querySelector('textarea')?.focus();
    });
  };

  const submitCurrent = async (createAnother = false) => {
    setSaving(true);
    try {
      if (createAnother && !step && onSubmitAndCreateAnother) {
        await onSubmitAndCreateAnother(value);
        setValue(newStep(nextOrder + 1));
        focusFirstField();
        return;
      }
      await onSubmit(value);
      if (!step) {
        setValue(newStep(nextOrder + 1));
      }
    } finally {
      setSaving(false);
    }
  };

  useImperativeHandle(ref, () => ({
    submit: submitCurrent,
  }));

  return (
    <form
      ref={formRef}
      className="form-grid step-form"
      onSubmit={async (event) => {
        event.preventDefault();
        await submitCurrent();
      }}
    >
      <MarkdownTextarea
        label={messages.testSheet.run.action}
        value={value.action}
        required
        onChange={(action) => setValue({ ...value, action })}
      />
      <MarkdownTextarea
        label={messages.testSheet.edit.specificField}
        value={value.field}
        onChange={(field) => setValue({ ...value, field })}
      />
      <MarkdownTextarea
        label={messages.testSheet.edit.expectedResult}
        value={value.expectedResult}
        onChange={(expectedResult) => setValue({ ...value, expectedResult })}
      />
      <div className="button-row">
        <Button type="submit" disabled={saving}>{saving ? messages.common.saving : step ? messages.common.save : messages.testSheet.edit.addStep}</Button>
        {!step && onSubmitAndCreateAnother && (
          <Button
            type="button"
            variant="secondary"
            disabled={saving}
            onClick={() => submitCurrent(true)}
          >
            {messages.testSheet.edit.addStepAndContinue}
          </Button>
        )}
        {onCancel && <Button variant="secondary" type="button" onClick={onCancel}>{messages.common.cancel}</Button>}
      </div>
    </form>
  );
});

function newStep(order: number): StepInput {
  return {
    action: '',
    field: '',
    expectedResult: '',
    executionOrder: order,
  };
}
