import type { LucideIcon } from 'lucide-react';
import {
  ArrowLeft,
  Calendar,
  Hash,
  Link as LinkIcon,
  Link2,
  List,
  Mail,
  MapPin,
  ToggleLeft,
  Type,
} from 'lucide-react';
import { IdText } from '@/components/id-text';

/** Untyped JSON payload with Appwrite-style keys ($id, $createdAt). */
export type Json = Record<string, unknown>;

export function str(v: unknown): string {
  return v == null ? '' : String(v);
}

export function fmtDate(v: unknown): string {
  if (!v) return '—';
  const d = new Date(String(v));
  if (Number.isNaN(d.getTime())) return String(v);
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function formatIndexType(value: string): string {
  switch (value) {
    case 'unique':
      return 'Unique';
    case 'fulltext':
      return 'GIN full-text';
    default:
      return 'B-tree';
  }
}

const TYPE_ICONS: Record<string, LucideIcon> = {
  string: Type,
  integer: Hash,
  float: Hash,
  boolean: ToggleLeft,
  datetime: Calendar,
  email: Mail,
  url: LinkIcon,
  enum: List,
  point: MapPin,
  relationship: Link2,
};

export function columnTypeIcon(type: string): LucideIcon {
  return TYPE_ICONS[type] ?? Type;
}

/** Case-insensitive client-side filter across a row's stringified values. */
export function localFilter<T extends Json>(rows: T[], search: string): T[] {
  const q = search.trim().toLowerCase();
  if (!q) return rows;
  return rows.filter((r) => JSON.stringify(r).toLowerCase().includes(q));
}

/* Back header used by the database + table detail views. */
export function BackHeader({
  title,
  subtitle,
  icon: Icon,
  onBack,
}: {
  title: string;
  subtitle: string;
  icon: LucideIcon;
  onBack: () => void;
}) {
  return (
    <div className="flex items-center gap-3">
      <button
        type="button"
        onClick={onBack}
        className="text-text-muted transition-colors hover:text-text-primary"
        aria-label="Back"
      >
        <ArrowLeft size={20} />
      </button>
      <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">{title}</h1>
      <Icon size={13} className="text-text-subtle" />
      <IdText id={subtitle} />
    </div>
  );
}

/* Rounded metric pill used by the SQL console + schema browser. */
export function MetricPill({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center rounded-full border border-border bg-surface-alt px-2.5 py-1 text-[length:var(--text-label)] text-text-primary">
      {label}
    </span>
  );
}

/* Wrap of selectable chips (index type, relationship type, permission action). */
export function ChipGroup<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((o) => {
        const sel = value === o.value;
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            className={
              sel
                ? 'rounded-[var(--radius-6)] border border-[var(--color-accent)] bg-fill-active px-3 py-1.5 text-[length:var(--text-label)] text-text-primary transition-colors'
                : 'rounded-[var(--radius-6)] border border-border px-3 py-1.5 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-secondary'
            }
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
