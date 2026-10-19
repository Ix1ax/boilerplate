import { Button as ButtonPrimitive } from '@base-ui/react/button';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from 'cn';

// Кнопки-«пилюли» системы «Вместе»: высота не меньше 44 px, жёлтый — только с чёрным текстом.
const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center gap-2 rounded-full border border-transparent bg-clip-padding font-bold whitespace-nowrap transition-[background-color,color,border-color,transform] duration-200 outline-none select-none active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none aria-disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-5",
  {
    variants: {
      variant: {
        default:
          'bg-primary text-primary-foreground hover:bg-blue-hover disabled:bg-blue-pale disabled:text-ink',
        gold: 'bg-gold text-ink hover:bg-[#f2c832] disabled:opacity-60',
        ink: 'bg-ink text-paper hover:bg-[#2b2f37] disabled:opacity-60 dark:bg-paper dark:text-ink dark:hover:bg-white',
        outline:
          'border-[1.5px] border-foreground bg-card text-foreground hover:bg-muted aria-expanded:bg-muted disabled:opacity-50',
        'outline-light':
          'border-[1.5px] border-paper/50 bg-transparent text-paper hover:border-paper hover:bg-paper/10 disabled:opacity-50',
        secondary:
          'bg-secondary text-secondary-foreground hover:bg-line-strong aria-expanded:bg-line-strong disabled:opacity-50 dark:hover:bg-ink-soft',
        ghost:
          'font-semibold text-foreground hover:bg-muted aria-expanded:bg-muted disabled:opacity-50',
        destructive:
          'bg-danger text-white hover:bg-[#6f1808] disabled:opacity-50 dark:bg-[#8a1f0b] dark:hover:bg-[#a52a12]',
        link: 'h-auto! min-h-11 rounded-none px-0! text-primary underline-offset-4 hover:underline dark:text-interest',
      },
      size: {
        default: 'min-h-12 px-5 text-base',
        xs: "min-h-9 gap-1 px-3 text-sm [&_svg:not([class*='size-'])]:size-4",
        sm: "min-h-11 gap-1.5 px-4 text-[15px] [&_svg:not([class*='size-'])]:size-4",
        lg: 'min-h-14 px-7 text-[17px]',
        xl: 'min-h-16 px-8 text-[19px]',
        icon: 'size-12',
        'icon-xs': "size-9 [&_svg:not([class*='size-'])]:size-4",
        'icon-sm': "size-11 [&_svg:not([class*='size-'])]:size-4",
        'icon-lg': 'size-14',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
);

const Button = ({
  className,
  variant = 'default',
  size = 'default',
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) => {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
};

export { Button, buttonVariants };
