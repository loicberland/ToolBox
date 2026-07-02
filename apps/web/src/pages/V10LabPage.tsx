import React from 'react';
import type { BeforeLeaveHandler } from '../app/App';
import { V10LabModule } from '../modules/v10-lab/V10LabModule';

export function V10LabPage({ onBeforeLeaveChange }: { onBeforeLeaveChange?: (handler: BeforeLeaveHandler | null) => void }) {
  return <V10LabModule onBeforeLeaveChange={onBeforeLeaveChange} />;
}
