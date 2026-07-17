import { Clock } from 'lucide-react';
import { AppBadge } from '@/components/app-badge';
import { actionColor, typeColor } from './credentials';

/* Compact tag pills for the vault list & detail views. */

export function TypeBadge({ type }: { type: string }) {
  return (
    <AppBadge label={type.replace(/_/g, ' ')} color={typeColor(type)} className="uppercase tracking-wide" />
  );
}

export function ActionBadge({ action }: { action: string }) {
  return <AppBadge label={action} color={actionColor(action)} className="uppercase tracking-wide" />;
}

/** Relative expiry indicator — red when expired, amber within a week, muted otherwise. */
export function ExpiryChip({ expiresAt }: { expiresAt: string }) {
  const dt = new Date(expiresAt);
  if (Number.isNaN(dt.getTime())) return null;
  const now = Date.now();
  const expired = dt.getTime() < now;
  const days = Math.floor((dt.getTime() - now) / 86_400_000);
  const label = expired ? 'Expired' : days === 0 ? 'Expires today' : `Expires in ${days}d`;
  const color = expired ? '#EF4444' : days < 7 ? '#F59E0B' : 'var(--text-muted)';
  return (
    <span
      className="inline-flex items-center gap-1 text-[length:var(--text-caption)]"
      style={{ color }}
    >
      <Clock size={11} />
      {label}
    </span>
  );
}
