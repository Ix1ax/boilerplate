import * as React from 'react';
import { Input as InputPrimitive } from '@base-ui/react/input';
import { cn } from 'cn';

const Input = ({
  className,
  type,
  ...props
}: React.ComponentProps<'input'>) => {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        'min-h-[54px] w-full min-w-0 rounded-lg border-[1.5px] border-input bg-card px-4 text-[17px] text-foreground transition-colors outline-none placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-2 aria-invalid:border-danger file:inline-flex file:h-8 file:border-0 file:bg-transparent file:text-sm file:font-semibold file:text-foreground',
        className,
      )}
      {...props}
    />
  );
};

export { Input };
