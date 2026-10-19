import * as React from 'react';
import { cn } from 'cn';

const Textarea = ({
  className,
  ...props
}: React.ComponentProps<'textarea'>) => {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        'flex field-sizing-content min-h-24 resize-y py-3.5 leading-normal w-full min-w-0 rounded-lg border-[1.5px] border-input bg-card px-4 text-[17px] text-foreground transition-colors outline-none placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-2 aria-invalid:border-danger',
        className,
      )}
      {...props}
    />
  );
};

export { Textarea };
