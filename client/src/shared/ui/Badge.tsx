import { mergeProps } from '@base-ui/react/merge-props';
import { useRender } from '@base-ui/react/use-render';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from 'cn';

// Метки-«пилюли». Статусы записи различаются и цветом, и значком в тексте (✓ ◷ ✕ ⊘ ◎).
const badgeVariants = cva(
  'group/badge inline-flex w-fit shrink-0 items-center justify-center gap-1.5 overflow-hidden rounded-full border border-transparent px-3 py-1 text-sm leading-tight font-semibold whitespace-nowrap transition-colors has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2 [&>svg]:pointer-events-none [&>svg]:size-3.5!',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground [a]:hover:bg-blue-hover',
        secondary: 'bg-muted text-foreground [a]:hover:bg-line-strong',
        soft: 'bg-background text-foreground',
        gold: 'bg-gold font-bold text-ink',
        ink: 'bg-ink text-paper dark:bg-paper dark:text-ink',
        outline: 'border-[1.5px] border-foreground text-foreground',
        dashed:
          'border-[1.5px] border-dashed border-muted-foreground font-bold text-ink-soft dark:text-muted-foreground',
        success: 'bg-success-soft font-bold text-success',
        warning: 'bg-warning-soft font-bold text-warning',
        destructive: 'bg-danger-soft font-bold text-danger',
        neutral: 'bg-neutral-soft font-bold text-neutral',
        interest: 'bg-interest-soft font-bold text-interest',
        ghost: 'hover:bg-muted hover:text-muted-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
);

const Badge = ({
  className,
  variant = 'default',
  render,
  ...props
}: useRender.ComponentProps<'span'> & VariantProps<typeof badgeVariants>) => {
  return useRender({
    defaultTagName: 'span',
    props: mergeProps<'span'>(
      {
        className: cn(badgeVariants({ variant }), className),
      },
      props,
    ),
    render,
    state: {
      slot: 'badge',
      variant,
    },
  });
};

export { Badge, badgeVariants };
