'use client';

import { type ComponentProps, useId, useState } from 'react';
import { CheckIcon, ChevronsUpDownIcon } from 'lucide-react';

import { cn } from './utils';

import { Button } from './Button';
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from './Command';
import { Popover, PopoverContent, PopoverTrigger } from './Popover';

export type ComboboxOption = {
  value: string;
  label: string;
  keywords?: string[];
};

interface ComboboxProps extends Pick<
  ComponentProps<'button'>,
  'id' | 'ref' | 'onBlur' | 'disabled' | 'aria-invalid' | 'aria-describedby'
> {
  options: ComboboxOption[];
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  onSearchChange?: (value: string) => void;
  selectedLabel?: string;
  onOpenChange?: (open: boolean) => void;
}

export const Combobox = ({
  options,
  value,
  onValueChange,
  placeholder = 'Выберите значение',
  searchPlaceholder = 'Поиск…',
  emptyMessage = 'Ничего не найдено.',
  onSearchChange,
  selectedLabel,
  onOpenChange,
  ...props
}: ComboboxProps) => {
  const [open, setOpen] = useState(false);
  const listId = useId();
  const selected = options.find((option) => option.value === value);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        onOpenChange?.(next);
        if (!next) onSearchChange?.('');
      }}
    >
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          className="min-h-11 min-w-0 w-full justify-between font-normal"
          {...props}
        >
          <span className="min-w-0 flex-1 truncate text-left">
            {selectedLabel ?? selected?.label ?? placeholder}
          </span>
          <ChevronsUpDownIcon className="size-4 shrink-0 text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-(--radix-popover-trigger-width) p-0"
      >
        <Command
          defaultValue={value || 'empty-value'}
          shouldFilter={!onSearchChange}
        >
          <CommandInput
            onValueChange={onSearchChange}
            placeholder={searchPlaceholder}
            aria-label={searchPlaceholder}
          />
          <CommandList id={listId}>
            <CommandEmpty>{emptyMessage}</CommandEmpty>
            {options.map((option) => (
              <CommandItem
                key={option.value}
                value={option.value || 'empty-value'}
                keywords={[option.label, ...(option.keywords ?? [])]}
                onSelect={() => {
                  onValueChange(option.value);
                  setOpen(false);
                  onOpenChange?.(false);
                  onSearchChange?.('');
                }}
              >
                <CheckIcon
                  className={cn(
                    'size-4 shrink-0',
                    value !== option.value && 'opacity-0',
                  )}
                />
                {option.label}
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
};
