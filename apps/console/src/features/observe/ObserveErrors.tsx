import { useMemo, useState } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  BellOff,
  CheckCircle2,
  Circle,
  Globe,
  MessageCircle,
  MousePointer2,
  Terminal,
  UserCheck,
} from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  OB_ACCENT,
  OB_GREEN,
  OB_ORANGE,
  OB_PURPLE,
  OB_RED,
  OB_SLATE,
  ObActionBtn,
  ObContextPanel,
  ObMetaBadge,
  asRecord,
  asRows,
  cap,
  levelColor,
  obFmtNum,
  obTimeAgo,
  useObserveResource,
} from './observe-shared';

/* ObserveErrors — ports observe_errors.dart. Error list with triage actions and
 * a full-page issue detail (stack trace, breadcrumbs, context panels). */

const COLUMNS: DataTableColumn[] = [
  { key: 'level', label: 'Level', flex: 2 },
  { key: 'title', label: 'Title', flex: 6 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'count', label: 'Events', flex: 2, sortable: false },
  { key: 'users', label: 'Users', flex: 2, sortable: false },
  { key: 'lastSeen', label: 'Last seen', flex: 2, sortable: false },
];

function statusColor(s: string): string {
  if (s === 'resolved') return OB_GREEN;
  if (s === 'ignored') return OB_SLATE;
  return OB_ORANGE;
}

export function ObserveErrors({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/errors', projectId, { limit: 50 });
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Record<string, string | null>>({});

  const allErrors = asRows(query.data?.errors);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return allErrors.filter((e) => {
      if (q && !String(e.title ?? '').toLowerCase().includes(q)) return false;
      if (filters.status && String(e.status ?? 'unresolved') !== filters.status) return false;
      if (filters.level && String(e.level ?? 'error') !== filters.level) return false;
      return true;
    });
  }, [allErrors, search, filters]);

  if (selectedId) {
    const err = allErrors.find((e) => e.$id === selectedId);
    return (
      <ErrorDetail
        err={err}
        onBack={() => setSelectedId(null)}
        onAction={async (action) => {
          await api.patch(`/observe/errors/${selectedId}/${action}`);
          await query.refetch();
          setSelectedId(null);
        }}
      />
    );
  }

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'level':
              return String(row.level ?? 'error');
            case 'title':
              return String(row.title ?? '');
            case 'status':
              return String(row.status ?? 'unresolved');
            case 'count':
              return String(row.count ?? 0);
            case 'users':
              return String(row.affectedUsers ?? 0);
            case 'lastSeen':
              return obTimeAgo(row.lastSeen);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'level') {
            const l = String(row.level ?? 'error');
            const c = levelColor(l);
            return (
              <span className="inline-flex items-center gap-1.5" style={{ color: c }}>
                <span className="h-[7px] w-[7px] rounded-full" style={{ backgroundColor: c }} />
                <span className="text-[length:var(--text-label)] font-medium">{cap(l)}</span>
              </span>
            );
          }
          if (key === 'status') {
            const s = String(row.status ?? 'unresolved');
            const c = statusColor(s);
            return <ObMetaBadge label={cap(s)} color={c} />;
          }
          if (key === 'count') {
            return (
              <span className="text-[length:var(--text-label)] text-text-secondary">
                {obFmtNum(row.count ?? 0)}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => AlertCircle}
        rowIconColor={(row) => levelColor(String(row.level ?? 'error'))}
        onRowClick={(row) => {
          const id = String(row.$id ?? '');
          if (id) setSelectedId(id);
        }}
        onDeleteRow={async (row) => {
          try {
            await api.patch(`/observe/errors/${row.$id}/ignore`);
            await query.refetch();
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
        filters={[
          {
            key: 'status',
            label: 'Status',
            options: ['unresolved', 'resolved', 'ignored'].map((v) => ({ value: v, label: cap(v) })),
          },
          {
            key: 'level',
            label: 'Level',
            options: ['fatal', 'error', 'warning', 'info'].map((v) => ({ value: v, label: cap(v) })),
          },
        ]}
        filterValues={filters}
        onFiltersChange={setFilters}
        searchHint="Search errors…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="errors"
        emptyIcon={CheckCircle2}
        emptyTitle="No errors found"
        emptySubtitle="Your project is running clean"
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />
    </div>
  );
}

/* ── Error detail ─────────────────────────────────────────────────────────── */

function ErrorDetail({
  err,
  onBack,
  onAction,
}: {
  err: Record<string, unknown> | undefined;
  onBack: () => void;
  onAction: (action: 'resolve' | 'ignore') => void;
}) {
  if (!err) {
    return (
      <div className="flex justify-center py-20 text-[length:var(--text-body)] text-text-secondary">
        Error not found
      </div>
    );
  }

  const title = String(err.title ?? 'Unknown error');
  const level = String(err.level ?? 'error');
  const status = String(err.status ?? 'unresolved');
  const stack = String(err.stackTrace ?? '');
  const breadcrumbs = asRows(err.breadcrumbs);
  const userCtx = asRecord(err.userContext);
  const reqCtx = asRecord(err.requestContext);
  const runtimeCtx = asRecord(err.runtimeContext);
  const tags = asRecord(err.tags);
  const activity = asRows(err.activity);
  const lc = levelColor(level);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center gap-4 border-b border-border px-6 py-3.5 md:px-8">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-[length:var(--text-body)] text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft size={14} />
          All errors
        </button>
        <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: lc }} />
        <span className="flex-1 truncate text-[length:var(--text-subhead)] font-semibold text-text-primary">
          {title}
        </span>
        {status === 'unresolved' ? (
          <div className="flex items-center gap-2">
            <ObActionBtn label="Resolve" color={OB_GREEN} onClick={() => onAction('resolve')} />
            <ObActionBtn label="Ignore" color={OB_SLATE} onClick={() => onAction('ignore')} />
          </div>
        ) : (
          <ObMetaBadge label={cap(status)} color={status === 'resolved' ? OB_GREEN : OB_SLATE} />
        )}
      </div>

      {/* Body */}
      <div className="flex flex-1 flex-col gap-6 overflow-y-auto p-6 lg:flex-row">
        {/* Left: stats + stack + breadcrumbs */}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap gap-6">
            <DetailStat label="Events" value={obFmtNum(err.count ?? 0)} />
            <DetailStat label="Affected users" value={String(err.affectedUsers ?? 0)} />
            <DetailStat label="First seen" value={obTimeAgo(err.firstSeen)} />
            <DetailStat label="Last seen" value={obTimeAgo(err.lastSeen)} />
          </div>

          {stack && (
            <div className="mt-6">
              <SectionLabel>Stack trace</SectionLabel>
              <pre className="mt-2 overflow-x-auto rounded-[var(--radius)] border border-white/10 bg-[#0D0D10] p-4 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] leading-[1.7] text-[#E2E8F0]">
                {stack}
              </pre>
            </div>
          )}

          {breadcrumbs.length > 0 && (
            <div className="mt-6">
              <SectionLabel>Breadcrumbs</SectionLabel>
              <div className="mt-2">
                {breadcrumbs.map((b, i) => (
                  <BreadcrumbRow key={i} crumb={b} />
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Right: context panels */}
        <div className="flex w-full flex-col gap-4 lg:w-[280px] lg:shrink-0">
          {Object.keys(tags).length > 0 && <TagsPanel tags={tags} />}
          <ObContextPanel title="USER" data={userCtx} />
          <ObContextPanel title="REQUEST" data={reqCtx} />
          <ObContextPanel title="RUNTIME" data={runtimeCtx} />
          {activity.length > 0 && (
            <div>
              <SectionLabel>Activity</SectionLabel>
              <div className="mt-2 flex flex-col gap-2.5">
                {activity.map((a, i) => (
                  <ActivityItem key={i} item={a} />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function DetailStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-[length:var(--text-title)] font-bold text-text-primary">{value}</span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">{label}</span>
    </div>
  );
}

function SectionLabel({ children }: { children: string }) {
  return (
    <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
      {children}
    </span>
  );
}

function BreadcrumbRow({ crumb }: { crumb: Record<string, unknown> }) {
  const type = String(crumb.type ?? 'default');
  const message = String(crumb.message ?? '');
  const color =
    type === 'error'
      ? OB_RED
      : type === 'warning'
        ? OB_ORANGE
        : type === 'http'
          ? OB_ACCENT
          : type === 'ui'
            ? OB_PURPLE
            : OB_SLATE;
  const Icon =
    type === 'error'
      ? AlertCircle
      : type === 'warning'
        ? AlertTriangle
        : type === 'http'
          ? Globe
          : type === 'ui'
            ? MousePointer2
            : type === 'console'
              ? Terminal
              : Circle;
  return (
    <div className="mb-1.5 flex gap-2.5">
      <div className="flex flex-col items-center">
        <Icon size={13} style={{ color }} />
        <span className="mt-0.5 h-5 w-px bg-border" />
      </div>
      <div className="flex flex-col">
        <span className="text-[length:var(--text-label)] text-text-primary">{message}</span>
        <span className="text-[length:var(--text-caption)] text-text-subtle">
          {obTimeAgo(crumb.timestamp)}
        </span>
      </div>
    </div>
  );
}

function TagsPanel({ tags }: { tags: Record<string, unknown> }) {
  return (
    <div className="flex flex-col">
      <SectionLabel>Tags</SectionLabel>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {Object.entries(tags).map(([k, v]) => (
          <span
            key={k}
            className="rounded-[var(--radius-sm)] border border-border bg-surface px-2 py-1 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary"
          >
            {k}: {String(v)}
          </span>
        ))}
      </div>
    </div>
  );
}

function ActivityItem({ item }: { item: Record<string, unknown> }) {
  const type = String(item.type ?? 'note');
  const user = String(item.user ?? 'System');
  const text = String(item.text ?? '');
  const Icon =
    type === 'resolved'
      ? CheckCircle2
      : type === 'ignored'
        ? BellOff
        : type === 'assigned'
          ? UserCheck
          : MessageCircle;
  const color = type === 'resolved' ? OB_GREEN : type === 'ignored' ? OB_SLATE : OB_ACCENT;
  return (
    <div className="flex gap-2">
      <Icon size={13} style={{ color, marginTop: 2 }} />
      <div className="flex flex-col">
        <div className="flex items-center gap-1.5">
          <span className="text-[length:var(--text-label)] font-medium text-text-primary">
            {user}
          </span>
          <span className="text-[length:var(--text-caption)] text-text-subtle">
            {obTimeAgo(item.timestamp)}
          </span>
        </div>
        {text && <span className="text-[length:var(--text-label)] text-text-secondary">{text}</span>}
      </div>
    </div>
  );
}
