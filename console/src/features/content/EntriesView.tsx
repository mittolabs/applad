import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, EyeOff, FileEdit, Plus, Send, Trash2 } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { StatusChip } from '@/components/status-chip';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { ConfirmDialog } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import { contentKeys, dateOnly, FilterChip, type ContentRow } from './shared';

const STATUS_FILTERS: { label: string; value: string }[] = [
  { label: 'All', value: '' },
  { label: 'Draft', value: 'draft' },
  { label: 'Published', value: 'published' },
  { label: 'Archived', value: 'archived' },
];

export function EntriesView({
  projectId,
  type,
  onBack,
  onSelectEntry,
  onNewEntry,
}: {
  projectId?: string;
  type: ContentRow;
  onBack: () => void;
  onSelectEntry: (e: ContentRow) => void;
  onNewEntry: () => void;
}) {
  const qc = useQueryClient();
  const typeId = String(type.$id);
  const [status, setStatus] = useState('');
  const [deleting, setDeleting] = useState<ContentRow | null>(null);
  const [removing, setRemoving] = useState(false);

  const entriesQuery = useQuery({
    queryKey: contentKeys.entries(projectId, typeId, status),
    queryFn: async () => {
      const res = await api.get(`/content/types/${typeId}/entries`, {
        params: status ? { status } : undefined,
      });
      return ((res.data as ContentRow)?.entries as ContentRow[]) ?? [];
    },
  });

  const entries = entriesQuery.data ?? [];
  const invalidate = () =>
    qc.invalidateQueries({ queryKey: ['content-entries', projectId, typeId] });

  const setEntryStatus = async (entry: ContentRow, publish: boolean) => {
    try {
      await api.patch(
        `/content/types/${typeId}/entries/${String(entry.$id)}/${publish ? 'publish' : 'unpublish'}`,
      );
      toast.success(publish ? 'Entry published' : 'Entry unpublished');
      invalidate();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  const confirmDelete = async () => {
    if (!deleting) return;
    setRemoving(true);
    try {
      await api.delete(`/content/types/${typeId}/entries/${String(deleting.$id)}`);
      toast.success('Entry deleted');
      setDeleting(null);
      invalidate();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRemoving(false);
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-start gap-2">
        <button
          type="button"
          onClick={onBack}
          className="mt-1 text-text-secondary transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={16} />
        </button>
        <div>
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
            {String(type.name ?? 'Content')}
          </h1>
          {type.slug != null && String(type.slug) !== '' && (
            <p className="mt-1 font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-subtle">
              {String(type.slug)}
            </p>
          )}
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2.5">
        {STATUS_FILTERS.map((f) => (
          <FilterChip
            key={f.value || 'all'}
            label={f.label}
            active={status === f.value}
            onClick={() => setStatus(f.value)}
          />
        ))}
        <div className="ml-auto">
          <Button size="sm" onClick={onNewEntry}>
            <Plus size={14} />
            New entry
          </Button>
        </div>
      </div>

      {/* Content */}
      {entriesQuery.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : entriesQuery.error ? (
        <ErrorState error={entriesQuery.error} onRetry={() => entriesQuery.refetch()} />
      ) : entries.length === 0 ? (
        <EmptyState
          icon={FileEdit}
          title="No entries"
          subtitle={
            status === ''
              ? 'Create your first entry for this content type.'
              : `No ${status} entries found.`
          }
          actionLabel={status === '' ? 'New entry' : undefined}
          onAction={status === '' ? onNewEntry : undefined}
        />
      ) : (
        <div className="flex flex-col">
          <div className="flex items-center border-b border-border py-2 text-[length:var(--text-caption)] font-semibold text-text-muted">
            <div className="flex-[4]">Slug</div>
            <div className="flex-[2]">Status</div>
            <div className="flex-[2]">Locale</div>
            <div className="flex-[1]">Ver.</div>
            <div className="flex-[3]">Updated</div>
            <div className="w-20" />
          </div>
          {entries.map((e) => (
            <EntryRow
              key={String(e.$id)}
              entry={e}
              onOpen={() => onSelectEntry(e)}
              onPublish={() => setEntryStatus(e, true)}
              onUnpublish={() => setEntryStatus(e, false)}
              onDelete={() => setDeleting(e)}
            />
          ))}
        </div>
      )}

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(o) => !o && setDeleting(null)}
        title="Delete entry"
        message="Delete this entry? This cannot be undone."
        loading={removing}
        onConfirm={confirmDelete}
      />
    </div>
  );
}

function EntryRow({
  entry,
  onOpen,
  onPublish,
  onUnpublish,
  onDelete,
}: {
  entry: ContentRow;
  onOpen: () => void;
  onPublish: () => void;
  onUnpublish: () => void;
  onDelete: () => void;
}) {
  const status = String(entry.status ?? 'draft');
  const slug = String(entry.slug ?? entry.$id ?? '—');
  const locale = String(entry.locale ?? 'en');
  const version = String(entry.version ?? '1');
  const updated = dateOnly(entry.$updatedAt ?? entry.updatedAt);

  const stop = (fn: () => void) => (ev: React.MouseEvent) => {
    ev.stopPropagation();
    fn();
  };

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(ev) => (ev.key === 'Enter' || ev.key === ' ') && onOpen()}
      className="flex cursor-pointer items-center border-b border-border py-2.5 transition-colors hover:bg-fill"
    >
      <div className="flex-[4] truncate pr-2 font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary">
        {slug}
      </div>
      <div className="flex-[2]">
        <StatusChip label={status} />
      </div>
      <div className="flex-[2] text-[length:var(--text-label)] text-text-secondary">{locale}</div>
      <div className="flex-[1] text-[length:var(--text-label)] text-text-muted">v{version}</div>
      <div className="flex-[3] text-[length:var(--text-label)] text-text-secondary">{updated}</div>
      <div className="flex w-20 items-center justify-end gap-2">
        {status === 'draft' && (
          <IconAction icon={Send} label="Publish" onClick={stop(onPublish)} />
        )}
        {status === 'published' && (
          <IconAction icon={EyeOff} label="Unpublish" onClick={stop(onUnpublish)} />
        )}
        <IconAction icon={Trash2} label="Delete" onClick={stop(onDelete)} />
      </div>
    </div>
  );
}

function IconAction({
  icon: Icon,
  label,
  onClick,
  className,
}: {
  icon: typeof Send;
  label: string;
  onClick: (ev: React.MouseEvent) => void;
  className?: string;
}) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      onClick={onClick}
      className={cn('text-text-muted transition-colors hover:text-text-primary', className)}
    >
      <Icon size={14} />
    </button>
  );
}
