'use client';

import type { ComponentProps } from 'react';

import { cn } from './utils';

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './Select';

export type SelectOption = {
  value: string | number;
  label: string;
  disabled?: boolean;
};
interface OptionSelectProps extends Pick<
  ComponentProps<typeof SelectTrigger>,
  | 'id'
  | 'ref'
  | 'onBlur'
  | 'disabled'
  | 'aria-label'
  | 'aria-labelledby'
  | 'aria-describedby'
  | 'aria-invalid'
  | 'className'
> {
  value: string | number;
  options: SelectOption[];
  onValueChange: (value: string) => void;
  name?: string;
  placeholder?: string;
}
export const OptionSelect = ({
  value,
  options,
  onValueChange,
  name,
  placeholder = 'Выберите значение',
  className,
  disabled,
  ...props
}: OptionSelectProps) => {
  return (
    <>
      <Select
        value={`option:${value}`}
        onValueChange={(next) => onValueChange(next.slice('option:'.length))}
        disabled={disabled}
      >
        <SelectTrigger
          {...props}
          disabled={disabled}
          className={cn(
            'h-10 data-[size=default]:h-10 w-full min-w-0 bg-background px-3 text-base font-normal md:text-sm [&_[data-slot=select-value]]:min-w-0 [&_[data-slot=select-value]]:truncate [&_[data-slot=select-value]]:text-left',
            className,
          )}
        >
          <SelectValue placeholder={placeholder}>
            <span className="block min-w-0 truncate">
              {options.find((option) => String(option.value) === String(value))
                ?.label ?? placeholder}
            </span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent
          position="popper"
          align="start"
          className="w-(--radix-select-trigger-width) max-w-[calc(100vw-2rem)]"
        >
          <SelectGroup>
            {options.map((option) => (
              <SelectItem
                key={option.value}
                value={`option:${option.value}`}
                disabled={option.disabled}
                className="min-h-9 py-2 pl-3 [&>span:last-child]:min-w-0 [&>span:last-child]:break-words"
              >
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {name && (
        <input type="hidden" name={name} value={value} disabled={disabled} />
      )}
    </>
  );
};
