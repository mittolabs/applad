import * as React from 'react';
import { cn } from '@/lib/utils';

export const Textarea = React.forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement>
>(({ className, ...props }, ref) => {
  return (
    <textarea
      ref={ref}
      className={cn(
        'flex min-h-[80px] w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2.5 text-[length:var(--text-body)] text-text-primary transition-colors',
        'placeholder:text-text-subtle',
        'focus-visible:border-[var(--color-accent)] focus-visible:outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  );
});
Textarea.displayName = 'Textarea';
