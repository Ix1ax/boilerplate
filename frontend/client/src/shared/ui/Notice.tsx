import type { ComponentProps, ReactNode } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from 'cn';

const noticeVariants = cva('flex flex-col gap-1 rounded-[1.25rem] p-4', {
  variants: {
    tone: {
      success: 'bg-success-soft text-success',
      warning: 'bg-warning-soft text-warning',
      danger: 'bg-danger-soft text-danger',
      neutral: 'bg-neutral-soft text-neutral',
      interest: 'bg-interest-soft text-interest',
      gold: 'bg-gold-soft text-ink',
      ink: 'bg-ink text-paper',
      outline: 'border-2 border-warning text-foreground',
    },
  },
  defaultVariants: { tone: 'neutral' },
});

interface NoticeProps
  extends
    Omit<ComponentProps<'div'>, 'title'>,
    VariantProps<typeof noticeVariants> {
  title?: ReactNode;
}

/** Плашка состояния: «Вы записаны», «Ожидает решения», «Снято». */
export const Notice = ({
  tone,
  title,
  className,
  children,
  ...props
}: NoticeProps) => {
  return (
    <div className={cn(noticeVariants({ tone }), className)} {...props}>
      {title ? <b className="text-xl leading-snug">{title}</b> : null}
      {children}
    </div>
  );
};
