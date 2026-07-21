/* Shared formatting helpers for deploy targets (Sites + Containers).
 * Ports the utility helpers at the bottom of sites_page.dart. */

export function timeAgo(ts: unknown): string {
  if (ts == null || ts === '') return '';
  try {
    const dt = new Date(String(ts));
    if (Number.isNaN(dt.getTime())) return '';
    const diffMs = Date.now() - dt.getTime();
    const days = Math.floor(diffMs / 86_400_000);
    const hours = Math.floor(diffMs / 3_600_000);
    const minutes = Math.floor(diffMs / 60_000);
    if (days > 30) return `${Math.floor(days / 30)}mo ago`;
    if (days > 0) return `${days}d ago`;
    if (hours > 0) return `${hours}h ago`;
    if (minutes > 0) return `${minutes}m ago`;
    return 'just now';
  } catch {
    return '';
  }
}

export function formatTimestamp(ts: unknown): string {
  if (ts == null) return 'N/A';
  const str = String(ts);
  if (str === '') return 'N/A';
  const dt = new Date(str);
  if (Number.isNaN(dt.getTime())) return str;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${dt.getFullYear()}-${p(dt.getMonth() + 1)}-${p(dt.getDate())} ${p(dt.getHours())}:${p(dt.getMinutes())}`;
}

/** Date-only (YYYY-MM-DD) from an ISO timestamp. */
export function shortDate(v: unknown): string {
  if (v == null) return '—';
  const str = String(v);
  if (str === '') return '—';
  return str.split('T')[0];
}

/** Duration given in whole seconds. */
/**
 * Renders a duration given in milliseconds.
 *
 * It previously read its argument as seconds while every caller passed
 * milliseconds, so a one-second build was reported as sixteen minutes.
 */
export function formatDuration(ms: unknown): string {
  if (ms == null) return '--';
  const total = typeof ms === 'number' ? Math.trunc(ms) : 0;
  if (total <= 0) return '--';
  if (total < 1000) return `${total}ms`;

  const s = Math.round(total / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  const r = s % 60;
  return r > 0 ? `${m}m ${r}s` : `${m}m`;
}

export function formatBytes(bytes: unknown): string {
  if (bytes == null) return '--';
  const b = typeof bytes === 'number' ? bytes : 0;
  if (b <= 0) return '--';
  if (b < 1024) return `${Math.trunc(b)} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`;
  return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function formatNumber(n: unknown): string {
  if (n == null) return '--';
  const v = typeof n === 'number' ? n : 0;
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}M`;
  if (v >= 1_000) return `${(v / 1_000).toFixed(1)}K`;
  return `${Math.trunc(v)}`;
}

/** Coerce a possibly-numeric JSON value to a number (0 fallback). */
export function asNumber(v: unknown): number {
  return typeof v === 'number' ? v : Number(v) || 0;
}

/** Read the Appwrite-style id from a JSON row. */
export function rowId(row: Record<string, unknown>): string {
  return String(row['$id'] ?? row['id'] ?? '');
}
