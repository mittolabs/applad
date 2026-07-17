import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

/*
 * Button — ports app.dart button themes.
 * text 14/w600, radius 8. Variants:
 *   primary   FilledButton   bg accent, white
 *   secondary ElevatedButton bg surface, primary text
 *   outline   OutlinedButton transparent, field border
 *   ghost     TextButton     secondary text, hover fill
 *   destructive               bg danger (#EF4444)
 */
const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[var(--radius)] text-[length:var(--text-control)] font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        primary: 'bg-[var(--color-accent)] text-white hover:opacity-90',
        secondary:
          'bg-surface text-text-primary border border-border hover:bg-fill-hover',
        outline:
          'border border-field-border bg-transparent text-text-primary hover:bg-fill',
        // Toolbar chip — ports app_data_table.dart _ToolbarChip (fieldFill bg + fieldBorder).
        toolbar:
          'border border-field-border bg-field-fill text-text-secondary hover:bg-fill-hover hover:text-text-primary',
        ghost: 'bg-transparent text-text-secondary hover:bg-fill hover:text-text-primary',
        destructive: 'bg-[var(--color-danger)] text-white hover:opacity-90',
        link: 'text-[var(--color-accent)] underline-offset-4 hover:underline',
      },
      size: {
        // `sm` (32px) is the standard control height across the app.
        sm: 'h-[var(--control-h)] px-3 text-[length:var(--text-body)]',
        default: 'h-9 px-5 py-2', // opt-in larger button (rare)
        lg: 'h-10 px-6',
        icon: 'h-[var(--control-h)] w-[var(--control-h)]',
      },
    },
    defaultVariants: {
      variant: 'primary',
      size: 'sm',
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, loading, children, disabled, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        ref={ref}
        className={cn(buttonVariants({ variant, size, className }))}
        disabled={disabled || loading}
        {...props}
      >
        {loading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
        {children}
      </Comp>
    );
  },
);
Button.displayName = 'Button';

export { buttonVariants };
