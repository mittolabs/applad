import { useState } from 'react';
import { AlertTriangle, Info, X, XCircle } from 'lucide-react';
import { cn } from '../lib/utils';
import {
  useDismissNotice,
  useNotices,
  type Notice,
  type NoticeRegion,
  type Theme,
} from '../hooks/use-entitlements';

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

export function Notices({
  region,
  className,
  flush,
}: {
  region: NoticeRegion;
  className?: string;
  /** Full-bleed bar with square corners, for a banner that spans the whole app
   *  rather than sitting inside a page's padding. */
  flush?: boolean;
}) {
  const notices = useNotices(region);
  const dismiss = useDismissNotice();
  // Local state only hides it instantly; the server is what makes it stay
  // dismissed across a refresh and on the user's other devices.
  const [hidden, setHidden] = useState<string[]>([]);
  const visible = notices.filter((n) => !hidden.includes(n.id));
  if (visible.length === 0) return null;

  return (
    <div className={cn('flex flex-col', flush ? 'gap-0' : 'gap-2', className)}>
      {visible.map((n) => (
        <NoticeBar
          key={n.id}
          notice={n}
          flush={flush}
          onDismiss={() => {
            setHidden((d) => [...d, n.id]);
            dismiss(n.id);
          }}
        />
      ))}
    </div>
  );
}

/* A theme is a set of values the server already validated against a fixed
 * vocabulary (colours are hex, images are https, effects are named). It is
 * turned into inline styles here; nothing from it is ever treated as markup. */
const PAD: Record<string, string> = {
  compact: 'py-1.5',
  normal: 'py-3',
  tall: 'py-6',
};

function themeStyle(t?: Theme): React.CSSProperties | undefined {
  if (!t) return undefined;
  const s: React.CSSProperties = {};
  if (t.background && t.gradientTo) {
    s.backgroundImage = `linear-gradient(${t.gradientAngle ?? 135}deg, ${t.background}, ${t.gradientTo})`;
  } else if (t.background) {
    s.backgroundColor = t.background;
  }
  if (t.image) {
    const layer = `url("${t.image}")`;
    s.backgroundImage = s.backgroundImage ? `${layer}, ${s.backgroundImage}` : layer;
    s.backgroundSize = 'cover';
    s.backgroundPosition = 'center';
  }
  if (t.textColor) s.color = t.textColor;
  if (t.background || t.image) s.borderColor = 'transparent';
  return s;
}

function NoticeBar({
  notice,
  onDismiss,
  flush,
}: {
  notice: Notice;
  onDismiss: () => void;
  flush?: boolean;
}) {
  const level = LEVEL[notice.level] ?? LEVEL.info;
  const { Icon } = level;
  const t = notice.theme;
  const themed = !!(t?.background || t?.image);
  return (
    <div
      className={cn(
        'relative overflow-hidden border',
        t?.align === 'center' ? 'flex items-center justify-center gap-2.5' : 'flex items-start gap-2.5',
        flush
          ? 'rounded-none border-x-0 border-t-0 px-6'
          : 'rounded-[var(--radius)] px-3.5',
        PAD[t?.height ?? 'normal'] ?? PAD.normal,
        !themed && level.cls,
        t?.effect && `notice-fx notice-fx-${t.effect}`,
      )}
      style={themeStyle(t)}
      role={notice.level === 'critical' ? 'alert' : 'status'}
    >
      {t?.icon ? (
        <span className="mt-px shrink-0 text-[15px] leading-none">{t.icon}</span>
      ) : (
        <Icon size={16} className={cn('mt-px shrink-0', ICON_COLOR[notice.level] ?? ICON_COLOR.info)} />
      )}
      <div className="relative z-[1] min-w-0 flex-1">
        <div
          className={cn(
            'text-[length:var(--text-body)] font-medium',
            themed ? '' : 'text-text-primary',
          )}
        >
          {notice.title}
        </div>
        {notice.body && (
          <div
            className={cn(
              'mt-0.5 text-[length:var(--text-label)]',
              themed ? 'opacity-90' : 'text-text-secondary',
            )}
          >
            {notice.body}
          </div>
        )}
      </div>
      {notice.action && (
        <a
          href={notice.action.href}
          style={t?.accentColor ? { backgroundColor: t.accentColor } : undefined}
          className={cn(
            'relative z-[1] shrink-0 rounded-[var(--radius-6)] px-2.5 py-1 text-[length:var(--text-label)] font-medium text-white transition-opacity hover:opacity-90',
            !t?.accentColor && 'bg-[var(--color-accent)]',
          )}
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
