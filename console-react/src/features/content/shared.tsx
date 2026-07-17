import { cn } from '@/lib/utils';

/* Shared types, constants and helpers for the Content (headless CMS) feature.
 * Ports the model bits of console/lib/features/content/content_page.dart. */

export type ContentRow = Record<string, unknown>;

export const FIELD_TYPES = [
  'text',
  'richtext',
  'number',
  'boolean',
  'date',
  'media',
  'reference',
  'slug',
  'seo',
] as const;

export type FieldType = (typeof FIELD_TYPES)[number];

export interface FieldDef {
  key: string;
  label: string;
  type: string;
  required: boolean;
}

/** Parse a content type's `fields` array into typed field definitions. */
export function parseFields(type: ContentRow | null | undefined): FieldDef[] {
  const raw = (type?.fields as unknown[]) ?? [];
  return raw.map((f) => {
    const o = (f ?? {}) as Record<string, unknown>;
    return {
      key: String(o.key ?? ''),
      label: String(o.label ?? ''),
      type: String(o.type ?? 'text'),
      required: o.required === true,
    };
  });
}

/** Derive a URL-safe slug from an arbitrary string. */
export function slugify(raw: string): string {
  return raw
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');
}

/** Strip the time component from an ISO timestamp (`2024-01-01T…` → `2024-01-01`). */
export function dateOnly(v: unknown): string {
  return String(v ?? '').split('T')[0];
}

/** React-query keys, scoped by project. */
export const contentKeys = {
  types: (projectId?: string) => ['content-types', projectId] as const,
  entries: (projectId: string | undefined, typeId: string, status: string) =>
    ['content-entries', projectId, typeId, status] as const,
  versions: (projectId: string | undefined, typeId: string, entryId: string) =>
    ['content-versions', projectId, typeId, entryId] as const,
};

/** Pill filter chip — ports _FilterChip. */
export function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'rounded-[var(--radius-6)] border px-2.5 py-1 text-[length:var(--text-label)] transition-colors',
        active
          ? 'border-[var(--color-accent)] bg-[color-mix(in_srgb,var(--color-accent)_12%,transparent)] font-medium text-[var(--color-accent)]'
          : 'border-border bg-fill text-text-secondary hover:text-text-primary',
      )}
    >
      {label}
    </button>
  );
}

/** Neutral tag pill — ports _Badge. */
export function TypeBadge({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center rounded-[var(--radius-sm)] border border-border bg-fill px-[7px] py-[3px] text-[length:var(--text-caption)] text-text-muted">
      {label}
    </span>
  );
}
