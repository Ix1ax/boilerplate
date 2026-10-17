'use client';

import { type ComponentProps, useId, useState } from 'react';
import { PaperclipIcon } from 'lucide-react';

import { cn } from './utils';

import { buttonVariants } from './Button';

interface FileInputProps extends Omit<
  ComponentProps<'input'>,
  'type' | 'className'
> {
  /** Имя выбранного файла, если выбор хранит родитель (например, после сброса). */
  fileName?: string | null;
  placeholder?: string;
  buttonLabel?: string;
  className?: string;
}

/** Поле выбора файла: кнопка и имя файла вместо системного контрола браузера. */
export const FileInput = ({
  id,
  fileName,
  placeholder = 'Файл не выбран',
  buttonLabel = 'Выбрать файл',
  disabled,
  onChange,
  className,
  ...props
}: FileInputProps) => {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const [selected, setSelected] = useState('');
  const name = fileName === undefined ? selected : (fileName ?? '');

  return (
    <div
      data-slot="file-input"
      className={cn(
        'flex min-h-10 w-full min-w-0 items-center gap-3 rounded-lg border border-dashed border-input bg-background p-1 pr-3',
        'has-focus-visible:border-ring has-focus-visible:ring-3 has-focus-visible:ring-ring/50',
        disabled && 'opacity-60',
        className,
      )}
    >
      <input
        {...props}
        id={inputId}
        type="file"
        disabled={disabled}
        className="peer sr-only"
        onChange={(event) => {
          setSelected(
            Array.from(event.target.files ?? [])
              .map((file) => file.name)
              .join(', '),
          );
          onChange?.(event);
        }}
      />
      <label
        htmlFor={inputId}
        aria-hidden="true"
        className={cn(
          buttonVariants({ variant: 'outline', size: 'sm' }),
          'shrink-0 cursor-pointer',
          disabled && 'pointer-events-none',
        )}
      >
        <PaperclipIcon />
        {buttonLabel}
      </label>
      <span
        className={cn(
          'min-w-0 truncate text-sm',
          name ? 'text-foreground' : 'text-muted-foreground',
        )}
      >
        {name || placeholder}
      </span>
    </div>
  );
};
