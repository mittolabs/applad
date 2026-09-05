import { type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';

/* ────────────────────────────────────────────────────────────────────────────
 * analytics-shared — palette, formatting helpers, a hand-rolled SVG chart and
 * the project-scoped fetch hook shared by the Analytics views.
 *
 * The colours are the theme's status tokens rather than the fixed hex values
 * the old Observe pages carried, so these views follow the light theme instead
 * of staying dark inside it.
 * ──────────────────────────────────────────────────────────────────────────── */

export const ACCENT = 'var(--color-accent)';
export const PURPLE = 'var(--color-accent-3)';
export const GREEN = 'var(--status-success)';
export const ORANGE = 'var(--status-warning)';
export const RED = 'var(--status-danger)';
export const INFO = 'var(--status-info)';
export const SLATE = 'var(--status-neutral)';

/** color-mix tint helper — `color` at `pct`% over transparent. */
export function tint(color: string, pct: number): string {
  return `color-mix(in srgb, ${color} ${pct}%, transparent)`;
}

/*
 * Rendering a measurement that may not exist.
 *
 * The observe pages defaulted uptime and apdex to their perfect values, so an
 * instance with no data reported 100.00% uptime and a 1.00 apdex in green. A
 * number nobody measured is shown as absent.
 */
export function metric(v: unknown, opts: { suffix?: string; digits?: number } = {}): string {
  if (typeof v !== 'number' || !Number.isFinite(v)) return '—';
  const { suffix = '', digits } = opts;
  return (digits == null ? String(v) : v.toFixed(digits)) + suffix;
}

/** The colour for a percentage where higher is better, grey when unmeasured. */
export function healthColor(v: unknown, good = 99.9, fair = 99): string {
  if (typeof v !== 'number' || !Number.isFinite(v)) return SLATE;
  if (v >= good) return GREEN;
  if (v >= fair) return ORANGE;
  return RED;
}

export function apdexColor(v: unknown): string {
  // No score is not a perfect score.
  if (typeof v !== 'number') return SLATE;
  if (v >= 0.9) return GREEN;
  if (v >= 0.7) return ORANGE;
  return RED;
}

export function timeAgo(v: unknown): string {
  if (v == null) return '—';
  const dt = new Date(String(v));
  if (Number.isNaN(dt.getTime())) return String(v);
  const s = Math.floor((Date.now() - dt.getTime()) / 1000);
  if (s < 60) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

export function fmtNum(v: unknown): string {
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
 * A project-scoped GET that stays disabled until a project is selected.
 * Individual views own their retries and post-mutation refetch. */
export function useAnalyticsResource(
  path: string,
  projectId: string | undefined,
  params?: Record<string, string | number>,
) {
  return useQuery({
    queryKey: ['analytics', path, projectId, params ?? null],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get(path, params ? { params } : undefined);
      return res.data as Record<string, unknown>;
    },
  });
}

/* ─── Presentational primitives ─────────────────────────────────────────── */

export function SectionTitle({ title, trailing }: { title: string; trailing?: ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[length:var(--text-control)] font-semibold text-text-primary">
        {title}
      </span>
      {trailing}
    </div>
  );
}

/* Hand-rolled SVG area+line chart — no chart libraries. */
export function LineChart({ points, color }: { points: number[]; color: string }) {
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

/* Horizontal bar used by the top-events list and funnel steps. */
export function Bar({ pct, color }: { pct: number; color: string }) {
  const clamped = Math.max(0, Math.min(100, pct));
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full" style={{ backgroundColor: tint(color, 12) }}>
      <div className="h-full rounded-full" style={{ width: `${clamped}%`, backgroundColor: color }} />
    </div>
  );
}
