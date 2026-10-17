'use client';

import type { ComponentProps } from 'react';
import { Command as CommandPrimitive } from 'cmdk';
import { SearchIcon } from 'lucide-react';

import { cn } from './utils';

type CommandProps = ComponentProps<typeof CommandPrimitive>;

const Command = ({ className, ...props }: CommandProps) => {
  return (
    <CommandPrimitive
      className={cn(
        'flex w-full flex-col overflow-hidden rounded-lg bg-popover text-popover-foreground',
        className,
      )}
      {...props}
    />
  );
};

type CommandInputProps = ComponentProps<typeof CommandPrimitive.Input>;

const CommandInput = ({ className, ...props }: CommandInputProps) => {
  return (
    <div className="flex items-center gap-2 border-b px-3">
      <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
      <CommandPrimitive.Input
        className={cn(
          'h-10 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground disabled:opacity-50',
          className,
        )}
        {...props}
      />
    </div>
  );
};

type CommandListProps = ComponentProps<typeof CommandPrimitive.List>;

const CommandList = ({ className, ...props }: CommandListProps) => {
  return (
    <CommandPrimitive.List
      className={cn(
        'max-h-64 overflow-x-hidden overflow-y-auto p-1',
        className,
      )}
      {...props}
    />
  );
};

type CommandEmptyProps = ComponentProps<typeof CommandPrimitive.Empty>;

const CommandEmpty = ({ className, ...props }: CommandEmptyProps) => {
  return (
    <CommandPrimitive.Empty
      className={cn(
        'px-3 py-6 text-center text-sm text-muted-foreground',
        className,
      )}
      {...props}
    />
  );
};

type CommandItemProps = ComponentProps<typeof CommandPrimitive.Item>;

const CommandItem = ({ className, ...props }: CommandItemProps) => {
  return (
    <CommandPrimitive.Item
      className={cn(
        'relative flex cursor-default items-center gap-2 rounded-md px-2 py-2 text-sm outline-none select-none data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground data-[disabled=true]:pointer-events-none data-[disabled=true]:opacity-50',
        className,
      )}
      {...props}
    />
  );
};

export { Command, CommandEmpty, CommandInput, CommandItem, CommandList };
