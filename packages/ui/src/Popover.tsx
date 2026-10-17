'use client';

import type { ComponentProps } from 'react';
import { Popover as PopoverPrimitive } from 'radix-ui';

import { cn } from './utils';

type PopoverProps = ComponentProps<typeof PopoverPrimitive.Root>;

const Popover = ({ ...props }: PopoverProps) => {
  return <PopoverPrimitive.Root data-slot="popover" {...props} />;
};

type PopoverTriggerProps = ComponentProps<typeof PopoverPrimitive.Trigger>;

const PopoverTrigger = ({ ...props }: PopoverTriggerProps) => {
  return <PopoverPrimitive.Trigger data-slot="popover-trigger" {...props} />;
};

type PopoverContentProps = ComponentProps<typeof PopoverPrimitive.Content>;

const PopoverContent = ({
  className,
  align = 'center',
  sideOffset = 4,
  ...props
}: PopoverContentProps) => {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        data-slot="popover-content"
        align={align}
        sideOffset={sideOffset}
        className={cn(
          'z-50 flex w-72 origin-(--radix-popover-content-transform-origin) flex-col gap-3 rounded-2xl bg-popover p-4 text-sm text-popover-foreground shadow-md ring-1 ring-foreground/10 outline-hidden duration-100 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95',
          className,
        )}
        {...props}
      />
    </PopoverPrimitive.Portal>
  );
};

type PopoverAnchorProps = ComponentProps<typeof PopoverPrimitive.Anchor>;

const PopoverAnchor = ({ ...props }: PopoverAnchorProps) => {
  return <PopoverPrimitive.Anchor data-slot="popover-anchor" {...props} />;
};

type PopoverHeaderProps = ComponentProps<'div'>;

const PopoverHeader = ({ className, ...props }: PopoverHeaderProps) => {
  return (
    <div
      data-slot="popover-header"
      className={cn('flex flex-col gap-0.5 text-sm', className)}
      {...props}
    />
  );
};

type PopoverTitleProps = ComponentProps<'h2'>;

const PopoverTitle = ({ className, ...props }: PopoverTitleProps) => {
  return (
    <div
      data-slot="popover-title"
      className={cn('font-medium', className)}
      {...props}
    />
  );
};

type PopoverDescriptionProps = ComponentProps<'p'>;

const PopoverDescription = ({
  className,
  ...props
}: PopoverDescriptionProps) => {
  return (
    <p
      data-slot="popover-description"
      className={cn('text-muted-foreground', className)}
      {...props}
    />
  );
};

export {
  Popover,
  PopoverAnchor,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
};
