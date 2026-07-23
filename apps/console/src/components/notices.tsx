import { useState } from 'react';
import { AlertTriangle, Info, X, XCircle } from 'lucide-react';
import { cn } from '../lib/utils';
import { useNotices, type Notice, type NoticeRegion } from '../hooks/use-entitlements';

/**
 * Banners, rendered by core in core styling from DATA.
 *
 * Nothing injects markup here. A provider supplies a Notice and core decides how
 * it looks, which is what keeps a composed build consistent with the rest of the
 * console and keeps a core restyle from breaking whoever supplied it.
 */
const LEVEL = {
  info: { Icon: Info, cls: 'border-[var(--color-accent)]/30 bg-[var(--color-accent)]/10' },
  warn: { Icon: AlertTriangle, cls: 'border-status-warning/30 bg-status-warning/10' },
  critical: { Icon: XCircle, cls: 'border-status-danger/30 bg-status-danger/10' },
} as const;

const ICON_COLOR = {
  info: 'text-[var(--color-accent)]',
  warn: 'text-status-warning',
  critical: 'text-status-danger',
} as const;

export function Notices({ region, className }: { region: NoticeRegion; className?: string }) {
  const notices = useNotices(region);
  const [dismissed, setDismissed] = useState<string[]>([]);
  const visible = notices.filter((n) => !dismissed.includes(n.id));
  if (visible.length === 0) return null;

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      {visible.map((n) => (
        <NoticeBar key={n.id} notice={n} onDismiss={() => setDismissed((d) => [...d, n.id])} />
      ))}
    </div>
  );
}

function NoticeBar({ notice, onDismiss }: { notice: Notice; onDismiss: () => void }) {
  const level = LEVEL[notice.level] ?? LEVEL.info;
  const { Icon } = level;
  return (
    <div
      className={cn(
        'flex items-start gap-2.5 rounded-[var(--radius)] border px-3.5 py-2.5',
        level.cls,
      )}
      role={notice.level === 'critical' ? 'alert' : 'status'}
    >
      <Icon size={16} className={cn('mt-px shrink-0', ICON_COLOR[notice.level] ?? ICON_COLOR.info)} />
      <div className="min-w-0 flex-1">
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">
          {notice.title}
        </div>
        {notice.body && (
          <div className="mt-0.5 text-[length:var(--text-label)] text-text-secondary">
            {notice.body}
          </div>
        )}
      </div>
      {notice.action && (
        <a
          href={notice.action.href}
          className="shrink-0 rounded-[var(--radius-6)] bg-[var(--color-accent)] px-2.5 py-1 text-[length:var(--text-label)] font-medium text-white transition-opacity hover:opacity-90"
        >
          {notice.action.label}
        </a>
      )}
      {notice.dismissible && (
        <button
          onClick={onDismiss}
          aria-label="Dismiss"
          className="shrink-0 rounded-[var(--radius-sm)] p-1 text-text-muted transition-colors hover:bg-fill hover:text-text-primary"
        >
          <X size={14} />
        </button>
      )}
    </div>
  );
}
