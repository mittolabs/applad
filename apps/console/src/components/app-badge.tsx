import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

/* Ports app_badge.dart — neutral compact tag pill.
 * radius 4, padding H7 V2, optional 10px leading icon + 4px gap, label 11/w500.
 * Default fg = neutral grey; bg = fg @ ~12%. */
export function AppBadge({
  label,
  icon,
  color,
  backgroundColor,
  className,
}: {
  label: string;
  icon?: ReactNode;
  color?: string;
  backgroundColor?: string;
  className?: string;
}) {
  const fg = color ?? 'var(--status-neutral)';
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-[var(--radius-sm)] px-[7px] py-[2px] text-[length:var(--text-caption)] font-medium',
        className,
      )}
      style={{
        color: fg,
        backgroundColor:
          backgroundColor ?? `color-mix(in srgb, ${fg} 12%, transparent)`,
      }}
    >
      {icon && <span className="flex items-center [&>svg]:size-[10px]">{icon}</span>}
      {label}
    </span>
  );
}
