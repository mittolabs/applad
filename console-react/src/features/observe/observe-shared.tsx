import { type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';

/* ────────────────────────────────────────────────────────────────────────────
 * observe-shared — palette, helpers, hand-rolled SVG line chart and the data
 * fetch hook. Ports console/lib/features/observe/observe_shared.dart.
 * The Observe colour palette is bespoke (not part of the design-token set), so
 * these hex values match the Flutter source exactly and are applied inline.
 * ──────────────────────────────────────────────────────────────────────────── */

export const OB_ACCENT = '#3472A4';
export const OB_GREEN = '#10B981';
export const OB_RED = '#EF4444';
export const OB_ORANGE = '#F59E0B';
export const OB_PURPLE = '#8B5CF6';
export const OB_SLATE = '#64748B';

/** color-mix tint helper — `color` at `pct`% over transparent. */
export function tint(color: string, pct: number): string {
  return `color-mix(in srgb, ${color} ${pct}%, transparent)`;
}

export function levelColor(level: string): string {
  switch (level) {
    case 'fatal':
    case 'error':
      return OB_RED;
    case 'warn':
    case 'warning':
      return OB_ORANGE;
    case 'debug':
      return OB_SLATE;
    default:
      return OB_ACCENT;
  }
}

export function apdexColor(v: unknown): string {
  const d = typeof v === 'number' ? v : 1.0;
  if (d >= 0.9) return OB_GREEN;
  if (d >= 0.7) return OB_ORANGE;
  return OB_RED;
}

export function obTimeAgo(v: unknown): string {
  if (v == null) return '—';
  const dt = new Date(String(v));
  if (Number.isNaN(dt.getTime())) return String(v);
  const diffMs = Date.now() - dt.getTime();
  const s = Math.floor(diffMs / 1000);
  if (s < 60) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

export function obFmtNum(v: unknown): string {
  const n = typeof v === 'number' ? Math.trunc(v) : parseInt(String(v), 10) || 0;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return `${n}`;
}

export function cap(s: string): string {
  return s.length === 0 ? s : s[0].toUpperCase() + s.slice(1);
}

/** Read a nested record safely as `Record<string, unknown>`. */
export function asRecord(v: unknown): Record<string, unknown> {
  return v && typeof v === 'object' ? (v as Record<string, unknown>) : {};
}

/** Read a value as an array of records. */
export function asRows(v: unknown): Record<string, unknown>[] {
  return Array.isArray(v) ? (v as Record<string, unknown>[]) : [];
}

/** Numeric coercion with fallback. */
export function num(v: unknown, fallback = 0): number {
  if (typeof v === 'number') return v;
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

/* ─── Data hook ─────────────────────────────────────────────────────────────
 * Ports the observe_providers.dart FutureProviders — a project-scoped GET that
 * is disabled until a project is selected. Individual views own retries + the
 * post-mutation refetch. Endpoints live under /observe/* (see report). */
export function useObserveResource(
  path: string,
  projectId: string | undefined,
  params?: Record<string, string | number>,
) {
  return useQuery({
    queryKey: ['observe', path, projectId, params ?? null],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get(path, params ? { params } : undefined);
      return res.data as Record<string, unknown>;
    },
  });
}

/* ─── Presentational primitives ─────────────────────────────────────────── */

export function ObSectionTitle({
  title,
  trailing,
}: {
  title: string;
  trailing?: ReactNode;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[length:var(--text-control)] font-semibold text-text-primary">
        {title}
      </span>
      {trailing}
    </div>
  );
}

export function ObEmptyCard({ message }: { message: string }) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5 text-[length:var(--text-body)] text-text-secondary">
      {message}
    </div>
  );
}

export function ObMetaBadge({ label, color }: { label: string; color: string }) {
  return (
    <span
      className="rounded-[var(--radius-sm)] px-[7px] py-0.5 text-[length:var(--text-caption)] font-medium"
      style={{ color, backgroundColor: tint(color, 8) }}
    >
      {label}
    </span>
  );
}

export function ObActionBtn({
  label,
  color,
  onClick,
}: {
  label: string;
  color: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-[var(--radius-sm)] border px-2 py-[3px] text-[length:var(--text-caption)] font-medium transition-colors"
      style={{ color, borderColor: tint(color, 40) }}
    >
      {label}
    </button>
  );
}

/* Right-rail key/value context panel (USER / REQUEST / RUNTIME). */
export function ObContextPanel({
  title,
  data,
}: {
  title: string;
  data: Record<string, unknown>;
}) {
  const entries = Object.entries(data);
  if (entries.length === 0) return null;
  return (
    <div className="flex flex-col">
      <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
        {title}
      </span>
      <div className="mt-2 rounded-[var(--radius)] border border-border bg-surface p-3">
        {entries.map(([k, v]) => (
          <div key={k} className="mb-1.5 flex items-start gap-2 last:mb-0">
            <span className="w-28 shrink-0 text-[length:var(--text-label)] text-text-subtle">
              {k}
            </span>
            <span className="flex-1 break-all font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
              {String(v)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

/* Hand-rolled SVG area+line chart — no chart libraries. */
export function ObLineChart({ points, color }: { points: number[]; color: string }) {
  if (points.length < 2) return null;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = Math.max(max - min, 1);
  const step = 100 / (points.length - 1);
  const coords = points.map((p, i) => [i * step, 100 - ((p - min) / range) * 100] as const);
  const line = coords.map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x},${y}`).join(' ');
  const fill = `M0,100 ${coords.map(([x, y]) => `L${x},${y}`).join(' ')} L100,100 Z`;
  return (
    <svg className="h-full w-full" viewBox="0 0 100 100" preserveAspectRatio="none">
      <path d={fill} fill={color} fillOpacity={0.08} />
      <path
        d={line}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

/* Compact toolbar search field used by the Logs view (240px, leading icon). */
export function ObSearchField({
  value,
  onChange,
  hint,
  width = 280,
}: {
  value: string;
  onChange: (v: string) => void;
  hint: string;
  width?: number;
}) {
  return (
    <div
      className="flex h-8 items-center gap-2 rounded-[var(--radius)] border border-border bg-surface px-2.5"
      style={{ width }}
    >
      <SearchIcon />
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={hint}
        className="w-full bg-transparent text-[length:var(--text-body)] text-text-primary outline-none placeholder:text-text-subtle"
      />
    </div>
  );
}

function SearchIcon() {
  return (
    <svg
      width={14}
      height={14}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="shrink-0 text-text-subtle"
    >
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.3-4.3" />
    </svg>
  );
}
