import { cn } from '@/lib/utils';

/* Ports console/lib/core/widgets/status_chip.dart.
 * Pill radius 4, padding H8 V3, 5x5 dot + 5px gap, label 11/w500.
 * Colors come from --status-* tokens (light/dark pairs); bg is a ~12% tint. */

export type StatusVariant = 'success' | 'warning' | 'danger' | 'info' | 'neutral';

const VAR_MAP: Record<string, StatusVariant> = {};
const add = (variant: StatusVariant, words: string[]) =>
  words.forEach((w) => (VAR_MAP[w] = variant));
add('success', ['verified', 'active', 'completed', 'success', 'deployed', 'published', 'enabled', 'healthy', 'online', 'ready', 'passed']);
add('warning', ['unverified', 'pending', 'draft', 'paused', 'idle', 'scheduled', 'warning']);
add('danger', ['disabled', 'failed', 'error', 'suspended', 'banned', 'inactive', 'deleted', 'offline', 'unhealthy']);
add('info', ['running', 'processing', 'building', 'deploying', 'queued', 'info', 'pending_review']);

/** Map an arbitrary status string to a variant (defaults to neutral). */
export function statusVariant(status: string): StatusVariant {
  return VAR_MAP[status.trim().toLowerCase()] ?? 'neutral';
}

const COLOR_VAR: Record<StatusVariant, string> = {
  success: 'var(--status-success)',
  warning: 'var(--status-warning)',
  danger: 'var(--status-danger)',
  info: 'var(--status-info)',
  neutral: 'var(--status-neutral)',
};

export function StatusChip({
  label,
  variant,
  className,
}: {
  label: string;
  variant?: StatusVariant;
  className?: string;
}) {
  const v = variant ?? statusVariant(label);
  const color = COLOR_VAR[v];
  return (
    <span
      className={cn(
        'inline-flex items-center gap-[5px] rounded-[var(--radius-sm)] px-2 py-[3px] text-[length:var(--text-caption)] font-medium capitalize',
        className,
      )}
      style={{
        color,
        backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)`,
      }}
    >
      <span
        className="h-[5px] w-[5px] rounded-full"
        style={{ backgroundColor: color }}
      />
      {label.replace(/_/g, ' ')}
    </span>
  );
}
