import { useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  FileText,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  Search,
  Trash2,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { ConfirmDialog } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import {
  contentKeys,
  FilterChip,
  parseFields,
  TypeBadge,
  type ContentRow,
} from './shared';
import { TypeFormDialog } from './TypeFormDialog';

type TypeFilter = '' | 'versioned' | 'localized';

export function TypesView({
  projectId,
  onSelectType,
}: {
  projectId?: string;
  onSelectType: (t: ContentRow) => void;
}) {
  const qc = useQueryClient();
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<TypeFilter>('');
  const [isGrid, setIsGrid] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<ContentRow | null>(null);
  const [deleting, setDeleting] = useState<ContentRow | null>(null);
  const [removing, setRemoving] = useState(false);

  const typesQuery = useQuery({
    queryKey: contentKeys.types(projectId),
    queryFn: async () => {
      const res = await api.get('/content/types');
      return ((res.data as ContentRow)?.types as ContentRow[]) ?? [];
    },
  });

  const all = typesQuery.data ?? [];
  const types = useMemo(() => {
    const q = search.trim().toLowerCase();
    return all.filter((t) => {
      if (q) {
        const name = String(t.name ?? '').toLowerCase();
        const slug = String(t.slug ?? '').toLowerCase();
        if (!name.includes(q) && !slug.includes(q)) return false;
      }
      if (filter === 'versioned' && t.versioning !== true) return false;
      if (filter === 'localized' && t.localization !== true) return false;
      return true;
    });
  }, [all, search, filter]);

  const openCreate = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (t: ContentRow) => {
    setEditing(t);
    setDialogOpen(true);
  };
  const invalidate = () => qc.invalidateQueries({ queryKey: contentKeys.types(projectId) });

  const confirmDelete = async () => {
    if (!deleting) return;
    setRemoving(true);
    try {
      await api.delete(`/content/types/${String(deleting.$id)}`);
      toast.success('Content type deleted');
      setDeleting(null);
      invalidate();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRemoving(false);
    }
  };

  const filtered = search.trim() !== '' || filter !== '';

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Content</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Structured content types with custom fields, versioning and localization
        </p>
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2.5">
        <div className="relative w-56">
          <Search
            size={14}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
          />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search types…"
            className="h-[34px] pl-8"
          />
        </div>
        {(['', 'versioned', 'localized'] as TypeFilter[]).map((f) => (
          <FilterChip
            key={f || 'all'}
            label={f === '' ? 'All' : f === 'versioned' ? 'Versioned' : 'Localized'}
            active={filter === f}
            onClick={() => setFilter(f)}
          />
        ))}
        <div className="ml-auto flex items-center gap-3">
          <span className="text-[length:var(--text-label)] text-text-muted">
            {types.length} type{types.length === 1 ? '' : 's'}
          </span>
          <div className="flex items-center rounded-[var(--radius)] border border-border">
            <ViewToggleButton
              icon={LayoutGrid}
              active={isGrid}
              onClick={() => setIsGrid(true)}
            />
            <ViewToggleButton icon={ListIcon} active={!isGrid} onClick={() => setIsGrid(false)} />
          </div>
          <Button size="sm" onClick={openCreate}>
            <Plus size={14} />
            New type
          </Button>
        </div>
      </div>

      {/* Content */}
      {typesQuery.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : typesQuery.error ? (
        <ErrorState error={typesQuery.error} onRetry={() => typesQuery.refetch()} />
      ) : types.length === 0 ? (
        <EmptyState
          icon={FileText}
          title={filtered ? 'No types match' : 'No content types'}
          subtitle={
            filtered
              ? 'Try a different search or filter.'
              : 'Define structured content types with custom fields, versioning, and localization.'
          }
          actionLabel={filtered ? undefined : 'Create type'}
          onAction={filtered ? undefined : openCreate}
        />
      ) : isGrid ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(240px,1fr))] gap-3">
          {types.map((t) => (
            <TypeCard
              key={String(t.$id)}
              type={t}
              onOpen={() => onSelectType(t)}
              onEdit={() => openEdit(t)}
              onDelete={() => setDeleting(t)}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col">
          {types.map((t) => (
            <TypeListRow
              key={String(t.$id)}
              type={t}
              onOpen={() => onSelectType(t)}
              onEdit={() => openEdit(t)}
              onDelete={() => setDeleting(t)}
            />
          ))}
        </div>
      )}

      <TypeFormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        type={editing}
        onSaved={invalidate}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(o) => !o && setDeleting(null)}
        title="Delete type"
        message={`This will permanently delete "${String(deleting?.name ?? '')}" and every entry it contains. This cannot be undone.`}
        loading={removing}
        onConfirm={confirmDelete}
      />
    </div>
  );
}

function ViewToggleButton({
  icon: Icon,
  active,
  onClick,
}: {
  icon: typeof LayoutGrid;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex h-7 w-7 items-center justify-center rounded-[var(--radius-6)] transition-colors',
        active ? 'bg-fill-active text-[var(--color-accent)]' : 'text-text-muted hover:text-text-secondary',
      )}
    >
      <Icon size={14} />
    </button>
  );
}

function TypeMeta({ type }: { type: ContentRow }) {
  const fieldCount = parseFields(type).length;
  return (
    <>
      <TypeBadge label={`${fieldCount} field${fieldCount === 1 ? '' : 's'}`} />
      {type.versioning === true && <TypeBadge label="versioned" />}
      {type.localization === true && <TypeBadge label="i18n" />}
    </>
  );
}

function RowIconButton({
  icon: Icon,
  onClick,
  label,
}: {
  icon: typeof Pencil;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      className="text-text-muted transition-colors hover:text-text-primary"
    >
      <Icon size={14} />
    </button>
  );
}

function TypeCard({
  type,
  onOpen,
  onEdit,
  onDelete,
}: {
  type: ContentRow;
  onOpen: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="flex h-full flex-col rounded-[var(--radius-10)] border border-border bg-surface p-4 text-left transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_35%,transparent)] hover:bg-fill-hover"
    >
      <div className="flex items-center">
        <div className="flex h-[34px] w-[34px] items-center justify-center rounded-[var(--radius)] bg-fill text-text-subtle">
          <FileText size={16} />
        </div>
        <div className="ml-auto flex items-center gap-1.5">
          <RowIconButton icon={Pencil} onClick={onEdit} label="Edit" />
          <RowIconButton icon={Trash2} onClick={onDelete} label="Delete" />
        </div>
      </div>
      <div className="mt-3 truncate text-[length:var(--text-body)] font-semibold text-text-primary">
        {String(type.name ?? '')}
      </div>
      <div className="mt-0.5 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted">
        {String(type.slug ?? '')}
      </div>
      <div className="mt-auto flex flex-wrap gap-1.5 pt-3">
        <TypeMeta type={type} />
      </div>
    </button>
  );
}

function TypeListRow({
  type,
  onOpen,
  onEdit,
  onDelete,
}: {
  type: ContentRow;
  onOpen: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="group flex items-center gap-3 border-b border-border px-0.5 py-2.5 text-left transition-colors hover:bg-fill"
    >
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius)] bg-fill text-text-subtle">
        <FileText size={14} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[length:var(--text-body)] font-medium text-text-primary">
          {String(type.name ?? '')}
        </div>
        <div className="truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted">
          {String(type.slug ?? '')}
        </div>
      </div>
      <div className="flex items-center gap-1.5">
        <TypeMeta type={type} />
      </div>
      <div className="ml-4 flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
        <RowIconButton icon={Pencil} onClick={onEdit} label="Edit" />
        <RowIconButton icon={Trash2} onClick={onDelete} label="Delete" />
      </div>
    </button>
  );
}
