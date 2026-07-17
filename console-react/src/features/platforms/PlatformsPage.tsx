import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Layers } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { AppBadge } from '@/components/app-badge';
import { FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import { PlatformDetail } from './PlatformDetail';
import {
  PLATFORM_TYPES,
  fmtDate,
  identityHint,
  identityLabel,
  identityValue,
  platformId,
  typeBadgeColor,
  typeIconFor,
  typeLabel,
} from './platform-utils';

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'type', label: 'Type', flex: 2 },
  { key: 'identity', label: 'Identifier', flex: 4 },
  { key: 'created', label: 'Created', flex: 2 },
];

const PER_PAGE_DEFAULT = 12;

export function PlatformsPage() {
  const { projectId = '' } = useParams<{ projectId: string }>();
  const [selected, setSelected] = useState<Row | null>(null);
  const [creating, setCreating] = useState(false);

  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(PER_PAGE_DEFAULT);
  const [filters, setFilters] = useState<Record<string, string | null>>({});

  const query = useQuery({
    queryKey: ['platforms', projectId],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get(`/projects/${projectId}/platforms`);
      return ((res.data as Record<string, unknown>)['platforms'] as Row[] | undefined) ?? [];
    },
  });

  const all = useMemo(() => query.data ?? [], [query.data]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    const typeFilter = filters['type'];
    return all.filter((p) => {
      if (typeFilter && String(p['type'] ?? '') !== typeFilter) return false;
      if (!q) return true;
      return (
        String(p['name'] ?? '').toLowerCase().includes(q) ||
        String(p['type'] ?? '').toLowerCase().includes(q) ||
        identityValue(p).toLowerCase().includes(q)
      );
    });
  }, [all, search, filters]);

  const total = filtered.length;
  const paged = useMemo(
    () => filtered.slice((page - 1) * perPage, page * perPage),
    [filtered, page, perPage],
  );

  if (selected) {
    return (
      <PlatformDetail
        platform={selected}
        projectId={projectId}
        onBack={() => setSelected(null)}
        onChange={(next) => {
          setSelected(next);
          query.refetch();
        }}
        onDeleted={() => {
          setSelected(null);
          query.refetch();
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Platforms</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Register your applications to enable API access and optionally link deployments.
        </p>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={paged}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row['name'] ?? 'Unnamed');
            case 'type':
              return typeLabel(String(row['type'] ?? ''));
            case 'identity':
              return identityValue(row);
            case 'created':
              return fmtDate(row['$createdAt'] ?? row['createdAt']);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'type') {
            const type = String(row['type'] ?? '');
            const Icon = typeIconFor(type);
            return (
              <AppBadge label={typeLabel(type)} icon={<Icon />} color={typeBadgeColor(type)} />
            );
          }
          if (key === 'identity') {
            const val = identityValue(row);
            return val ? (
              <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-secondary">
                {val}
              </span>
            ) : (
              <span className="text-text-subtle">—</span>
            );
          }
          return undefined;
        }}
        rowIcon={(row) => typeIconFor(String(row['type'] ?? ''))}
        rowIconColor={(row) => typeBadgeColor(String(row['type'] ?? ''))}
        onRowClick={(row) => setSelected(row)}
        onDeleteRow={async (row) => {
          try {
            await api.delete(`/projects/${projectId}/platforms/${platformId(row)}`);
            toast.success('Platform removed');
            query.refetch();
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
        gridCard={(row) => <GridCard row={row} />}
        persistKey="platforms_view"
        createLabel="Add platform"
        onCreate={() => setCreating(true)}
        searchHint="Search by name, type, or identifier…"
        searchValue={search}
        onSearchChange={setSearch}
        onSearch={() => setPage(1)}
        total={total}
        perPage={perPage}
        page={page}
        onPerPageChange={(n) => {
          setPerPage(n);
          setPage(1);
        }}
        onPrev={() => setPage((p) => Math.max(1, p - 1))}
        onNext={() => setPage((p) => p + 1)}
        itemLabel="platforms"
        filters={[
          {
            key: 'type',
            label: 'Type',
            options: PLATFORM_TYPES.map((t) => ({ value: t.id, label: t.label })),
          },
        ]}
        filterValues={filters}
        onFiltersChange={(values) => {
          setFilters(values);
          setPage(1);
        }}
        emptyIcon={Layers}
        emptyTitle="No platforms registered"
        emptySubtitle="Register your web, mobile, desktop, or server platforms to enable API access."
        loading={query.isLoading}
        error={query.error}
        onRetry={() => query.refetch()}
      />

      <AddPlatformDialog
        open={creating}
        onOpenChange={setCreating}
        projectId={projectId}
        onCreated={() => query.refetch()}
      />
    </div>
  );
}

// ── Grid card ─────────────────────────────────────────────────────────────────────

function GridCard({ row }: { row: Row }) {
  const type = String(row['type'] ?? '');
  const Icon = typeIconFor(type);
  const accent = typeBadgeColor(type);
  const name = String(row['name'] ?? 'Unnamed');
  const identity = identityValue(row);
  return (
    <div className="flex h-full flex-col rounded-[var(--radius-10)] border border-border bg-surface p-4 transition-colors hover:border-field-border hover:bg-fill-hover">
      <div className="flex items-center justify-between">
        <span
          className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)]"
          style={{ backgroundColor: `color-mix(in srgb, ${accent} 10%, transparent)`, color: accent }}
        >
          <Icon size={18} />
        </span>
        <span
          className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium"
          style={{ backgroundColor: `color-mix(in srgb, ${accent} 12%, transparent)`, color: accent }}
        >
          {typeLabel(type)}
        </span>
      </div>
      <div className="mt-3 truncate text-[length:var(--text-control)] font-medium text-text-primary">{name}</div>
      {identity ? (
        <div className="mt-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
          {identity}
        </div>
      ) : (
        <div className="mt-1 text-[length:var(--text-caption)] text-text-subtle">No identifier set</div>
      )}
      <div className="mt-auto pt-3 text-[length:var(--text-caption)] text-text-subtle">
        {identityLabel(type)}
      </div>
    </div>
  );
}

// ── Add platform dialog ─────────────────────────────────────────────────────────

function AddPlatformDialog({
  open,
  onOpenChange,
  projectId,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  onCreated: () => void;
}) {
  const [type, setType] = useState('web');
  const [name, setName] = useState('');
  const [hostname, setHostname] = useState('');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setType('web');
    setName('');
    setHostname('');
  };

  const submit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api.post(`/projects/${projectId}/platforms`, {
        type,
        name: name.trim(),
        hostname: hostname.trim(),
      });
      onOpenChange(false);
      reset();
      onCreated();
      toast.success('Platform added');
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Add platform"
      subtitle="Register an application platform"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
      width={460}
    >
      <FormField label="Type">
        <div className="flex flex-wrap gap-2">
          {PLATFORM_TYPES.map((t) => {
            const sel = t.id === type;
            const Icon = t.icon;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => {
                  setType(t.id);
                  setHostname('');
                }}
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-[var(--radius)] border px-3 py-2 text-[length:var(--text-label)] transition-colors',
                  sel
                    ? 'border-[var(--color-accent)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] text-text-primary'
                    : 'border-field-border bg-fill text-text-secondary hover:text-text-primary',
                )}
              >
                <Icon size={14} style={sel ? { color: 'var(--color-accent)' } : undefined} />
                {t.label}
              </button>
            );
          })}
        </div>
      </FormField>
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="My app"
        autoFocus
      />
      <TextField
        label={identityLabel(type)}
        value={hostname}
        onChange={(e) => setHostname(e.target.value)}
        placeholder={identityHint(type)}
      />
    </FormDialog>
  );
}
