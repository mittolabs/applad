import { useMemo, useState } from 'react';
import {
  AlertCircle,
  ArrowLeft,
  ArrowUpDown,
  Circle,
  Keyboard,
  MousePointer2,
  Navigation,
  Play,
  Video,
} from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import {
  OB_ACCENT,
  OB_GREEN,
  OB_ORANGE,
  OB_RED,
  OB_SLATE,
  ObMetaBadge,
  ObSectionTitle,
  asRows,
  num,
  obTimeAgo,
  useObserveResource,
} from './observe-shared';

/* ObserveReplays — ports observe_replays.dart. Session replay list plus a
 * placeholder player + events/network/console detail (backendless). */

const COLUMNS: DataTableColumn[] = [
  { key: 'user', label: 'User', flex: 3 },
  { key: 'url', label: 'Page', flex: 5 },
  { key: 'duration', label: 'Duration', flex: 2, sortable: false },
  { key: 'errors', label: 'Errors', flex: 2, sortable: false },
  { key: 'flags', label: 'Flags', flex: 2 },
  { key: 'browser', label: 'Browser', flex: 2, sortable: false },
  { key: 'started', label: 'Started', flex: 2, sortable: false },
];

function fmtDur(secs: number): string {
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

export function ObserveReplays({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/replays', projectId);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Record<string, string | null>>({});

  const allReplays = asRows(query.data?.replays);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return allReplays.filter((r) => {
      if (
        q &&
        !String(r.user ?? '').toLowerCase().includes(q) &&
        !String(r.url ?? '').toLowerCase().includes(q)
      )
        return false;
      if (filters.flags === 'has_errors' && num(r.errorCount) === 0) return false;
      if (filters.flags === 'rage_click' && r.hasRageClick !== true) return false;
      return true;
    });
  }, [allReplays, search, filters]);

  if (selectedId) {
    const rep = allReplays.find((r) => r.$id === selectedId);
    return <ReplayDetail replay={rep} onBack={() => setSelectedId(null)} />;
  }

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'user':
              return String(row.user ?? 'Anonymous');
            case 'url':
              return String(row.url ?? '');
            case 'duration':
              return fmtDur(num(row.durationSecs));
            case 'errors':
              return String(row.errorCount ?? 0);
            case 'browser':
              return String(row.browser ?? '');
            case 'started':
              return obTimeAgo(row.startedAt);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'errors') {
            const n = num(row.errorCount);
            return n === 0 ? <span /> : <ObMetaBadge label={String(n)} color={OB_RED} />;
          }
          if (key === 'flags') {
            const rage = row.hasRageClick === true;
            const dead = row.hasDeadClick === true;
            if (!rage && !dead) return <span />;
            return (
              <span className="inline-flex items-center gap-1">
                {rage && <ObMetaBadge label="Rage" color={OB_ORANGE} />}
                {dead && <ObMetaBadge label="Dead" color={OB_SLATE} />}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => Play}
        rowIconColor={() => OB_ACCENT}
        onRowClick={(row) => {
          const id = String(row.$id ?? '');
          if (id) setSelectedId(id);
        }}
        filters={[
          {
            key: 'flags',
            label: 'Flags',
            options: [
              { value: 'has_errors', label: 'Has errors' },
              { value: 'rage_click', label: 'Rage click' },
            ],
          },
        ]}
        filterValues={filters}
        onFiltersChange={setFilters}
        searchHint="Search by user, URL…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="replays"
        emptyIcon={Video}
        emptyTitle="No session replays"
        emptySubtitle="Integrate the SDK to capture user sessions"
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />
    </div>
  );
}

/* ── Replay detail ────────────────────────────────────────────────────────── */

function ReplayDetail({
  replay,
  onBack,
}: {
  replay: Record<string, unknown> | undefined;
  onBack: () => void;
}) {
  if (!replay) {
    return (
      <div className="flex justify-center py-20 text-[length:var(--text-body)] text-text-secondary">
        Replay not found
      </div>
    );
  }

  const user = String(replay.user ?? 'Anonymous');
  const events = asRows(replay.events);
  const network = asRows(replay.network);
  const consoleLines = asRows(replay.console);
  const duration = num(replay.durationSecs);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center gap-4 border-b border-border px-6 py-3.5 md:px-8">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-[length:var(--text-body)] text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft size={14} />
          All replays
        </button>
        <span className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
          Session — {user}
        </span>
        <div className="flex-1" />
        <span className="text-[length:var(--text-label)] text-text-secondary">
          {fmtDur(duration)}
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-6 md:px-8">
        {/* Placeholder player */}
        <div className="flex h-[200px] items-center justify-center rounded-[var(--radius-12)] border border-white/[0.06] bg-[#0D0D10]">
          <div className="flex flex-col items-center gap-2">
            <Play size={32} className="text-white/30" />
            <span className="text-[length:var(--text-body)] text-white/30">Replay player</span>
          </div>
        </div>

        <div className="flex flex-col gap-6 lg:flex-row">
          <div className="min-w-0 flex-1">
            <ObSectionTitle title={`Events (${events.length})`} />
            <div className="mt-3 flex flex-col">
              {events.slice(0, 20).map((e, i) => (
                <EventRow key={i} event={e} />
              ))}
            </div>
          </div>
          <div className="min-w-0 flex-1">
            <ObSectionTitle title={`Network (${network.length})`} />
            <div className="mt-3 flex flex-col">
              {network.slice(0, 20).map((n, i) => (
                <NetworkRow key={i} req={n} />
              ))}
            </div>
          </div>
        </div>

        {consoleLines.length > 0 && (
          <div>
            <ObSectionTitle title={`Console (${consoleLines.length})`} />
            <div className="mt-3 rounded-[var(--radius)] border border-white/[0.06] bg-[#0D0D10] p-3">
              {consoleLines.map((c, i) => (
                <ConsoleLine key={i} entry={c} />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function EventRow({ event }: { event: Record<string, unknown> }) {
  const type = String(event.type ?? 'click');
  const target = String(event.target ?? '');
  const ts = num(event.offsetMs);
  const Icon =
    type === 'click'
      ? MousePointer2
      : type === 'scroll'
        ? ArrowUpDown
        : type === 'input'
          ? Keyboard
          : type === 'nav'
            ? Navigation
            : type === 'error'
              ? AlertCircle
              : Circle;
  const color = type === 'error' ? OB_RED : OB_SLATE;
  return (
    <div className="mb-1.5 flex items-center gap-1.5">
      <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-[#64748B]">
        {ts}ms
      </span>
      <Icon size={12} style={{ color }} />
      <span className="flex-1 truncate text-[length:var(--text-caption)] text-text-secondary">
        {target ? `${type}: ${target}` : type}
      </span>
    </div>
  );
}

function NetworkRow({ req }: { req: Record<string, unknown> }) {
  const method = String(req.method ?? 'GET');
  const url = String(req.url ?? '');
  const status = num(req.status, 200);
  const dur = num(req.durationMs);
  const sc = status >= 400 ? OB_RED : OB_GREEN;
  return (
    <div className="mb-1.5 flex items-center gap-1.5">
      <span
        className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-bold"
        style={{ color: OB_ACCENT }}
      >
        {method}
      </span>
      <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)]" style={{ color: sc }}>
        {status}
      </span>
      <span className="flex-1 truncate text-[length:var(--text-caption)] text-text-secondary">
        {url}
      </span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">{dur}ms</span>
    </div>
  );
}

function ConsoleLine({ entry }: { entry: Record<string, unknown> }) {
  const level = String(entry.level ?? 'log');
  const msg = String(entry.message ?? '');
  const lc = level === 'error' || level === 'fatal' ? OB_RED : level === 'warn' ? OB_ORANGE : '#94A3B8';
  return (
    <div className="mb-1 flex items-start gap-2">
      <span
        className="w-10 shrink-0 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-bold"
        style={{ color: lc }}
      >
        {level.toUpperCase()}
      </span>
      <span className="flex-1 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-[#E2E8F0]">
        {msg}
      </span>
    </div>
  );
}
