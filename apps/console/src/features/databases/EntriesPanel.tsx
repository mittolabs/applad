import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FileText, Globe, History, MoreHorizontal, Plus } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { SearchListHeader } from '@/components/search-list';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FormDialog, ConfirmDialog, TextField, TextAreaField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { ChipGroup, fmtDate, localFilter, str, type Json } from './shared';

/* Editorial view over a content-enabled table. Same rows API as the raw grid,
 * but presented as entries: draft/published, slug, locale and versions. */

const SYSTEM_FIELDS = ['id', 'created_at', 'updated_at', 'status', 'slug', 'locale', 'published_at', 'entry_group'];
const STATUS_FILTERS = ['All', 'Draft', 'Published'] as const;

export function EntriesPanel({ dbId, tableId }: { dbId: string; tableId: string }) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<(typeof STATUS_FILTERS)[number]>('All');
  const [locale, setLocale] = useState('');
  const [editing, setEditing] = useState<Json | null>(null);
  const [creating, setCreating] = useState(false);
  const [versionsFor, setVersionsFor] = useState<Json | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const columnsQuery = useQuery({
    queryKey: ['db-columns', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/columns`);
      return (res.data as { columns?: Json[] }).columns ?? [];
    },
  });
  const columns = columnsQuery.data ?? [];

  const entriesQuery = useQuery({
    queryKey: ['db-entries', dbId, tableId, statusFilter, locale],
    queryFn: async () => {
      const params: Record<string, string | number> = { limit: 100 };
      if (statusFilter !== 'All') params.status = statusFilter.toLowerCase();
      if (locale.trim()) params.locale = locale.trim();
      const res = await api.get(`${base}/rows`, { params });
      const d = res.data as { documents?: Json[]; rows?: Json[] };
      return d.rows ?? d.documents ?? [];
    },
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ['db-entries', dbId, tableId] });

  const publish = useMutation({
    mutationFn: ({ id, next }: { id: string; next: boolean }) =>
      api.post(`${base}/rows/${id}/${next ? 'publish' : 'unpublish'}`),
    onSuccess: invalidate,
    onError: (e) => toast.error(friendlyError(e)),
  });

  const del = async () => {
    if (!deleteTarget) return;
    await api.delete(`${base}/rows/${deleteTarget}`);
    setDeleteTarget(null);
    invalidate();
  };

  const entries = localFilter(entriesQuery.data ?? [], search);
  const error = entriesQuery.error ?? columnsQuery.error;

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        searchHint="Search entries"
        value={search}
        onChange={setSearch}
        trailing={
          <div className="flex items-center gap-2">
            <div className="relative">
              <Globe
                size={14}
                className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-text-subtle"
              />
              <Input
                value={locale}
                onChange={(e) => setLocale(e.target.value)}
                placeholder="All locales"
                className="w-[140px] pl-8"
              />
            </div>
            <Button size="sm" onClick={() => setCreating(true)}>
              <Plus size={14} />
              New entry
            </Button>
          </div>
        }
      />

      <ChipGroup
        options={STATUS_FILTERS.map((s) => ({ value: s, label: s }))}
        value={statusFilter}
        onChange={setStatusFilter}
      />

      {error ? (
        <ErrorState error={error} onRetry={entriesQuery.refetch} />
      ) : entriesQuery.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">
          Loading…
        </div>
      ) : entries.length === 0 ? (
        <EmptyState
          icon={FileText}
          title="No entries yet"
          subtitle="Create your first entry to start publishing."
          actionLabel="New entry"
          onAction={() => setCreating(true)}
        />
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
          {entries.map((entry) => {
            const id = str(entry['id']) || str(entry['$id']);
            const published = str(entry['status']) === 'published';
            return (
              <div
                key={id}
                className="group flex items-center gap-3 border-b border-[var(--fill)] px-4 py-3.5 last:border-0 hover:bg-fill"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setEditing(entry)}
                      className="truncate text-left text-[length:var(--text-body)] text-text-primary hover:underline"
                    >
                      {entryTitle(entry, columns)}
                    </button>
                    <StatusPill published={published} />
                  </div>
                  <div className="mt-0.5 truncate text-[length:var(--text-caption)] text-text-subtle">
                    {str(entry['slug']) || 'no slug'} · {str(entry['locale']) || 'en'} · updated{' '}
                    {fmtDate(entry['updated_at'] ?? entry['$updatedAt'])}
                  </div>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  loading={publish.isPending && publish.variables?.id === id}
                  onClick={() => publish.mutate({ id, next: !published })}
                >
                  {published ? 'Unpublish' : 'Publish'}
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      type="button"
                      className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-text-primary group-hover:opacity-100"
                      aria-label="Entry actions"
                    >
                      <MoreHorizontal size={14} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={() => setEditing(entry)}>Edit</DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => setVersionsFor(entry)}>
                      Version history
                    </DropdownMenuItem>
                    <DropdownMenuItem destructive onSelect={() => setDeleteTarget(id)}>
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            );
          })}
        </div>
      )}

      {(creating || editing) && (
        <EntryDialog
          base={base}
          columns={columns}
          entry={editing ?? undefined}
          onClose={() => {
            setCreating(false);
            setEditing(null);
          }}
          onSaved={() => {
            setCreating(false);
            setEditing(null);
            invalidate();
          }}
        />
      )}

      {versionsFor && (
        <VersionsDialog
          base={base}
          entry={versionsFor}
          onClose={() => setVersionsFor(null)}
        />
      )}

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete entry"
        message="This permanently deletes the entry. This cannot be undone."
        confirmLabel="Delete"
        destructive
        onConfirm={del}
      />
    </div>
  );
}

function StatusPill({ published }: { published: boolean }) {
  const color = published ? 'var(--status-success)' : 'var(--color-text-muted)';
  return (
    <span
      className="shrink-0 rounded-full px-2 py-0.5 text-[length:var(--text-caption)] font-medium"
      style={{
        color,
        background: `color-mix(in srgb, ${color} 14%, transparent)`,
      }}
    >
      {published ? 'Published' : 'Draft'}
    </span>
  );
}

/** Best-effort display name: the first user-defined text field, else the slug/id. */
function entryTitle(entry: Json, columns: Json[]): string {
  for (const col of columns) {
    const key = str(col['key']);
    if (!key || SYSTEM_FIELDS.includes(key)) continue;
    const v = str(entry[key]);
    if (v) return v;
  }
  return str(entry['slug']) || str(entry['id']) || 'Untitled';
}

function EntryDialog({
  base,
  columns,
  entry,
  onClose,
  onSaved,
}: {
  base: string;
  columns: Json[];
  entry?: Json;
  onClose: () => void;
  onSaved: () => void;
}) {
  const editing = Boolean(entry);
  const fields = columns
    .map((c) => ({ key: str(c['key']), type: str(c['type']) }))
    .filter((c) => c.key && !SYSTEM_FIELDS.includes(c.key));

  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(fields.map((f) => [f.key, str(entry?.[f.key])])),
  );
  const [slug, setSlug] = useState(str(entry?.['slug']));
  const [locale, setLocale] = useState(str(entry?.['locale']) || 'en');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      const data: Record<string, unknown> = { ...values, slug, locale };
      if (editing) await api.patch(`${base}/rows/${str(entry?.['id']) || str(entry?.['$id'])}`, { data });
      else await api.post(`${base}/rows`, { data });
      onSaved();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={editing ? 'Edit entry' : 'New entry'}
      submitLabel={editing ? 'Save' : 'Create'}
      loading={saving}
      width={620}
      onSubmit={submit}
    >
      {fields.map((f) =>
        f.type === 'richtext' ? (
          <TextAreaField
            key={f.key}
            label={f.key}
            value={values[f.key] ?? ''}
            onChange={(e) => setValues((p) => ({ ...p, [f.key]: e.target.value }))}
            rows={8}
            placeholder="Write your content…"
          />
        ) : (
          <TextField
            key={f.key}
            label={f.key}
            value={values[f.key] ?? ''}
            onChange={(e) => setValues((p) => ({ ...p, [f.key]: e.target.value }))}
            placeholder={f.type === 'media' ? 'File ID or URL' : `Enter ${f.key}`}
          />
        ),
      )}
      <TextField
        label="Slug"
        value={slug}
        onChange={(e) => setSlug(e.target.value)}
        placeholder="my-first-post"
      />
      <TextField
        label="Locale"
        value={locale}
        onChange={(e) => setLocale(e.target.value)}
        placeholder="en"
      />
    </FormDialog>
  );
}

function VersionsDialog({
  base,
  entry,
  onClose,
}: {
  base: string;
  entry: Json;
  onClose: () => void;
}) {
  const id = str(entry['id']) || str(entry['$id']);
  const { data, isLoading } = useQuery({
    queryKey: ['db-row-versions', base, id],
    queryFn: async () => {
      const res = await api.get(`${base}/rows/${id}/versions`);
      return (res.data as { versions?: Json[] }).versions ?? [];
    },
  });
  const versions = data ?? [];

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Version history"
      submitLabel="Close"
      width={560}
      onSubmit={async () => onClose()}
    >
      {isLoading ? (
        <div className="py-8 text-center text-[length:var(--text-body)] text-text-subtle">
          Loading…
        </div>
      ) : versions.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-8 text-center">
          <History size={22} className="text-text-subtle" />
          <Label>No versions recorded yet.</Label>
        </div>
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
          {versions.map((v) => (
            <div
              key={str(v['$id'])}
              className="border-b border-[var(--fill)] px-4 py-3 last:border-0"
            >
              <div className="flex items-center justify-between">
                <span className="text-[length:var(--text-body)] text-text-primary">
                  Version {str(v['version'])}
                </span>
                <span className="text-[length:var(--text-caption)] text-text-subtle">
                  {fmtDate(v['$createdAt'])}
                </span>
              </div>
              <pre className="mt-2 max-h-32 overflow-auto rounded-[var(--radius-6)] bg-[var(--fill)] p-2 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
                {JSON.stringify(v['data'] ?? {}, null, 2)}
              </pre>
            </div>
          ))}
        </div>
      )}
    </FormDialog>
  );
}
