import type { LucideIcon } from 'lucide-react';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';

/* Ports app_empty_state.dart — centered icon-box + title + subtitle +
 * optional accent action button. Reused by DataTable's empty state. */
export function EmptyState({
  icon: Icon,
  title,
  subtitle,
  actionLabel,
  onAction,
  className,
}: {
  icon?: LucideIcon;
  title: string;
  subtitle?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center px-6 py-16 text-center',
        className,
      )}
    >
      {Icon && (
        <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-[var(--radius-12)] border border-border bg-fill text-text-muted">
          <Icon size={22} />
        </div>
      )}
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        {title}
      </div>
      {subtitle && (
        <div className="mt-1 max-w-sm text-[length:var(--text-body)] text-text-muted">
          {subtitle}
        </div>
      )}
      {actionLabel && onAction && (
        <Button className="mt-5" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
