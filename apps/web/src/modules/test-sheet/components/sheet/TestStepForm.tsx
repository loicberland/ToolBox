import React, { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
import { StepInput, TestSheetStep } from '../../api/testSheet';
import { messages } from '../../../../i18n';
import { Button } from '../../../../shared/components/ui/Button';
import { MarkdownTextarea } from '../../../../shared/components/ui/MarkdownTextarea';

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
  const [showSpecificField, setShowSpecificField] = useState(false);

  useEffect(() => {
    const nextValue = step ? {
      action: step.action,
      field: step.field,
      expectedResult: step.expectedResult,
      executionOrder: step.executionOrder,
    } : newStep(nextOrder);
    setValue(nextValue);
    setShowSpecificField(nextValue.field.trim() !== '');
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
        setShowSpecificField(false);
        focusFirstField();
        return;
      }
      await onSubmit(value);
      if (!step) {
        setValue(newStep(nextOrder + 1));
        setShowSpecificField(false);
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
      <div className="specific-field-toggle">
        <span>{messages.testSheet.edit.specificField}</span>
        {!showSpecificField && (
          <Button type="button" size="sm" variant="secondary" onClick={() => setShowSpecificField(true)}>
            +
          </Button>
        )}
      </div>
      {showSpecificField && (
        <MarkdownTextarea
          value={value.field}
          onChange={(field) => setValue({ ...value, field })}
        />
      )}
      <MarkdownTextarea
        label={messages.testSheet.edit.expectedResult}
        value={value.expectedResult}
        onChange={(expectedResult) => setValue({ ...value, expectedResult })}
      />
      <div className="button-row">
        {!step && onSubmitAndCreateAnother && (
          <Button
            type="submit"
            disabled={saving}
            onClick={async (event) => {
              event.preventDefault();
              await submitCurrent(true);
            }}
          >
            {messages.testSheet.edit.addStepAndContinue}
          </Button>
        )}
        <Button type="submit" variant={!step && onSubmitAndCreateAnother ? 'secondary' : undefined} disabled={saving}>
          {saving ? messages.common.saving : step ? messages.common.save : messages.testSheet.edit.addStep}
        </Button>
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
