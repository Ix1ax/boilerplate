'use client';

import type { ComponentProps } from 'react';
import { XIcon } from 'lucide-react';
import { Dialog as SheetPrimitive } from 'radix-ui';

import { cn } from './utils';
import { Button } from './Button';

type SheetProps = ComponentProps<typeof SheetPrimitive.Root>;

const Sheet = ({ ...props }: SheetProps) => {
  return <SheetPrimitive.Root data-slot="sheet" {...props} />;
};

type SheetTriggerProps = ComponentProps<typeof SheetPrimitive.Trigger>;

const SheetTrigger = ({ ...props }: SheetTriggerProps) => {
  return <SheetPrimitive.Trigger data-slot="sheet-trigger" {...props} />;
};

type SheetCloseProps = ComponentProps<typeof SheetPrimitive.Close>;

const SheetClose = ({ ...props }: SheetCloseProps) => {
  return <SheetPrimitive.Close data-slot="sheet-close" {...props} />;
};

type SheetPortalProps = ComponentProps<typeof SheetPrimitive.Portal>;

const SheetPortal = ({ ...props }: SheetPortalProps) => {
  return <SheetPrimitive.Portal data-slot="sheet-portal" {...props} />;
};

type SheetOverlayProps = ComponentProps<typeof SheetPrimitive.Overlay>;

const SheetOverlay = ({ className, ...props }: SheetOverlayProps) => {
  return (
    <SheetPrimitive.Overlay
      data-slot="sheet-overlay"
      className={cn(
        'fixed inset-0 z-50 bg-overlay duration-100 supports-backdrop-filter:backdrop-blur-xs data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0',
        className,
      )}
      {...props}
    />
  );
};

interface SheetContentProps extends ComponentProps<
  typeof SheetPrimitive.Content
> {
  side?: 'top' | 'right' | 'bottom' | 'left';
  showCloseButton?: boolean;
}

const SheetContent = ({
  className,
  children,
  side = 'right',
  showCloseButton = true,
  ...props
}: SheetContentProps) => {
  return (
    <SheetPortal>
      <SheetOverlay />
      <SheetPrimitive.Content
        data-slot="sheet-content"
        data-side={side}
        className={cn(
          'fixed z-50 flex flex-col gap-4 bg-popover bg-clip-padding text-sm text-popover-foreground shadow-lg transition duration-200 data-[side=right]:rounded-l-2xl data-[side=left]:rounded-r-2xl data-[side=bottom]:rounded-t-2xl data-[side=top]:rounded-b-2xl ease-in-out data-[side=bottom]:inset-x-0 data-[side=bottom]:bottom-0 data-[side=bottom]:h-auto data-[side=bottom]:border-t data-[side=left]:inset-y-0 data-[side=left]:left-0 data-[side=left]:h-full data-[side=left]:w-3/4 data-[side=left]:border-r data-[side=right]:inset-y-0 data-[side=right]:right-0 data-[side=right]:h-full data-[side=right]:w-3/4 data-[side=right]:border-l data-[side=top]:inset-x-0 data-[side=top]:top-0 data-[side=top]:h-auto data-[side=top]:border-b data-[side=left]:sm:max-w-sm data-[side=right]:sm:max-w-sm data-open:animate-in data-open:fade-in-0 data-[side=bottom]:data-open:slide-in-from-bottom-10 data-[side=left]:data-open:slide-in-from-left-10 data-[side=right]:data-open:slide-in-from-right-10 data-[side=top]:data-open:slide-in-from-top-10 data-closed:animate-out data-closed:fade-out-0 data-[side=bottom]:data-closed:slide-out-to-bottom-10 data-[side=left]:data-closed:slide-out-to-left-10 data-[side=right]:data-closed:slide-out-to-right-10 data-[side=top]:data-closed:slide-out-to-top-10',
          className,
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <SheetPrimitive.Close data-slot="sheet-close" asChild>
            <Button
              variant="ghost"
              className="absolute top-3 right-3"
              size="icon-sm"
            >
              <XIcon />
              <span className="sr-only">Закрыть</span>
            </Button>
          </SheetPrimitive.Close>
        )}
      </SheetPrimitive.Content>
    </SheetPortal>
  );
};

type SheetHeaderProps = ComponentProps<'div'>;

const SheetHeader = ({ className, ...props }: SheetHeaderProps) => {
  return (
    <div
      data-slot="sheet-header"
      className={cn('flex flex-col gap-0.5 p-6', className)}
      {...props}
    />
  );
};

type SheetFooterProps = ComponentProps<'div'>;

const SheetFooter = ({ className, ...props }: SheetFooterProps) => {
  return (
    <div
      data-slot="sheet-footer"
      className={cn('mt-auto flex flex-col gap-2 p-6', className)}
      {...props}
    />
  );
};

type SheetTitleProps = ComponentProps<typeof SheetPrimitive.Title>;

const SheetTitle = ({ className, ...props }: SheetTitleProps) => {
  return (
    <SheetPrimitive.Title
      data-slot="sheet-title"
      className={cn(
        'font-heading text-base font-medium text-foreground',
        className,
      )}
      {...props}
    />
  );
};

type SheetDescriptionProps = ComponentProps<typeof SheetPrimitive.Description>;

const SheetDescription = ({ className, ...props }: SheetDescriptionProps) => {
  return (
    <SheetPrimitive.Description
      data-slot="sheet-description"
      className={cn('text-sm text-muted-foreground', className)}
      {...props}
    />
  );
};

export {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
};
