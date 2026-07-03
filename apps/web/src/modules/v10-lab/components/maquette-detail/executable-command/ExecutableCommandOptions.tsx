import React from 'react';
import { ExecutableCommandOption, ExecutableCommandGroup } from '../../../utils/v10LabUtils';

export function ExecutableCommandOptions({ groups, excludedNames = [], valueFor = (option) => option.name }: { groups: ExecutableCommandGroup[]; excludedNames?: string[]; valueFor?: (option: ExecutableCommandOption) => string }) {
  const excluded = new Set(excludedNames);
  return <>
    {groups.map((group) => {
      const options = group.options.filter((option) => !excluded.has(option.name));
      if (!options.length) {
        return null;
      }
      return (
        <optgroup key={group.label} label={group.label}>
          {options.map((option) => <option key={valueFor(option)} value={valueFor(option)}>{option.label}</option>)}
        </optgroup>
      );
    })}
  </>;
}
