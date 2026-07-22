import * as React from 'react';
import { cn } from '@/lib/utils';

/* Input — ports app.dart inputDecorationTheme: fill field-fill, radius 8,
 * padding H14 V10, border field-border, focus accent, text 13, hint subtle. */
export const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, type, ...props }, ref) => {
  return (
    <input
      type={type}
      ref={ref}
      className={cn(
        'flex h-[var(--control-h)] w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-3 text-[length:var(--text-body)] text-text-primary transition-colors',
        'placeholder:text-text-subtle',
        'focus-visible:border-[var(--color-accent)] focus-visible:outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        'file:border-0 file:bg-transparent file:text-[length:var(--text-body)] file:font-medium',
        className,
      )}
      {...props}
    />
  );
});
Input.displayName = 'Input';
