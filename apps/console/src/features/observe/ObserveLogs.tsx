import { useMemo, useState } from 'react';
import { ChevronDown, ChevronUp, Filter as FilterIcon, RefreshCw, Terminal } from 'lucide-react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import {
  OB_ACCENT,
  OB_GREEN,
  OB_ORANGE,
  OB_RED,
  OB_SLATE,
  ObSearchField,
  asRecord,
  asRows,
  tint,
  useObserveResource,
} from './observe-shared';

/* ObserveLogs — ports observe_logs.dart. Structured log stream with a search
 * box, level/source filter popover, a column picker, a live toggle and
 * expandable per-line metadata. */

const ALL_COLS: [string, string][] = [
  ['timestamp', 'Time'],
  ['level', 'Level'],
  ['source', 'Source'],
  ['message', 'Message'],
];

const LEVELS = ['debug', 'info', 'warn', 'error', 'fatal'];

function logLevelColor(level: string): string {
  if (level === 'fatal' || level === 'error') return OB_RED;
  if (level === 'warn') return OB_ORANGE;
  if (level === 'debug') return OB_SLATE;
  return '#94A3B8';
}

export function ObserveLogs({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/logs', projectId, { limit: 100 });
  const [search, setSearch] = useState('');
  const [live, setLive] = useState(true);
  const [filterLevel, setFilterLevel] = useState<string | null>(null);
  const [filterSource, setFilterSource] = useState<string | null>(null);
  const [visibleCols, setVisibleCols] = useState<string[]>([
    'timestamp',
    'level',
    'source',
    'message',
  ]);

  const allLogs = asRows(query.data?.logs);

  const sources = useMemo(
    () =>
      Array.from(
        new Set(allLogs.map((l) => String(l.source ?? '')).filter(Boolean)),
      ).sort(),
    [allLogs],
  );

  const logs = useMemo(() => {
    const q = search.toLowerCase();
    return allLogs.filter((l) => {
      if (filterLevel && l.level !== filterLevel) return false;
      if (filterSource && l.source !== filterSource) return false;
      if (q && !String(l.message ?? '').toLowerCase().includes(q)) return false;
      return true;
    });
  }, [allLogs, search, filterLevel, filterSource]);

  const activeFilterCount = [filterLevel, filterSource].filter(Boolean).length;

  const toggleCol = (key: string) => {
    if (key === 'message') return;
    setVisibleCols((prev) => {
      if (prev.includes(key)) return prev.length > 1 ? prev.filter((c) => c !== key) : prev;
      return [...prev, key];
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-6 py-2.5 md:px-8">
        <ObSearchField value={search} onChange={setSearch} hint="Search logs…" />
        <div className="flex-1" />

        {/* Filter popover */}
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" size="sm">
              <FilterIcon size={13} style={activeFilterCount ? { color: OB_ACCENT } : undefined} />
              <span style={activeFilterCount ? { color: OB_ACCENT } : undefined}>
                {activeFilterCount ? `Filter (${activeFilterCount})` : 'Filter'}
              </span>
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-60 p-3">
            <div className="flex flex-col gap-3">
              <FilterSelect
                label="Level"
                value={filterLevel}
                options={LEVELS}
                onChange={setFilterLevel}
              />
              {sources.length > 0 && (
                <FilterSelect
                  label="Source"
                  value={filterSource}
                  options={sources}
                  onChange={setFilterSource}
                />
              )}
              {activeFilterCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setFilterLevel(null);
                    setFilterSource(null);
                  }}
                >
                  Clear all
                </Button>
              )}
            </div>
          </PopoverContent>
        </Popover>

        {/* Column picker */}
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" size="sm">
              <VerticalBars />
              {visibleCols.length}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-48 p-1.5">
            <div className="px-2 pb-1.5 pt-1 text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
              Columns
            </div>
            {ALL_COLS.map(([key, label]) => {
              const checked = visibleCols.includes(key);
              const isMsg = key === 'message';
              return (
                <button
                  key={key}
                  type="button"
                  disabled={isMsg}
                  onClick={() => toggleCol(key)}
                  className="flex w-full items-center gap-2 rounded-[var(--radius-6)] px-2 py-1.5 text-left transition-colors hover:bg-fill disabled:cursor-default"
                >
                  <span
                    className="flex h-4 w-4 items-center justify-center rounded-[var(--radius-sm)] border"
                    style={{
                      backgroundColor: checked ? OB_ACCENT : 'transparent',
                      borderColor: checked ? OB_ACCENT : 'var(--border)',
                    }}
                  >
                    {checked && <CheckMark />}
                  </span>
                  <span className="text-[length:var(--text-body)] text-text-secondary">{label}</span>
                </button>
              );
            })}
          </PopoverContent>
        </Popover>

        {/* Live toggle */}
        <button
          type="button"
          onClick={() => setLive((v) => !v)}
          className="flex h-8 items-center gap-1.5 rounded-[var(--radius)] border px-2.5 text-[length:var(--text-label)] transition-colors"
          style={
            live
              ? { color: OB_GREEN, backgroundColor: tint(OB_GREEN, 10), borderColor: tint(OB_GREEN, 40) }
              : undefined
          }
        >
          <span
            className="h-1.5 w-1.5 rounded-full"
            style={{ backgroundColor: live ? OB_GREEN : 'var(--text-subtle)' }}
          />
          {live ? 'Live' : 'Paused'}
        </button>

        <Button
          variant="ghost"
          size="icon"
          onClick={() => query.refetch()}
          aria-label="Refresh"
        >
          <RefreshCw size={14} />
        </Button>
      </div>

      {/* Log stream */}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {query.error ? (
          <ErrorState error={query.error} onRetry={query.refetch} />
        ) : query.isLoading ? (
          <div className="flex justify-center py-20">
            <Loader2 className="h-6 w-6 animate-spin" style={{ color: OB_ACCENT }} />
          </div>
        ) : logs.length === 0 ? (
          <EmptyState
            icon={Terminal}
            title="No logs match the current filters"
            subtitle="Try adjusting your search or filter settings."
          />
        ) : (
          logs.map((log, i) => <LogLine key={i} log={log} visibleCols={visibleCols} />)
        )}
      </div>
    </div>
  );
}

function FilterSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string | null;
  options: string[];
  onChange: (v: string | null) => void;
}) {
  const ANY = '__any__';
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
        {label}
      </span>
      <Select
        value={value ?? ANY}
        onValueChange={(v) => onChange(v === ANY ? null : v)}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ANY}>Any</SelectItem>
          {options.map((o) => (
            <SelectItem key={o} value={o}>
              {o}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function LogLine({
  log,
  visibleCols,
}: {
  log: Record<string, unknown>;
  visibleCols: string[];
}) {
  const [expanded, setExpanded] = useState(false);
  const level = String(log.level ?? 'info');
  const ts = String(log.timestamp ?? '');
  const msg = String(log.message ?? '');
  const src = String(log.source ?? '');
  const meta = log.meta && typeof log.meta === 'object' ? asRecord(log.meta) : null;
  const lc = logLevelColor(level);
  const tsShort = ts.includes('T') ? ts.split('T')[1].split('.')[0] : ts;

  return (
    <div className="border-b border-white/[0.04]">
      <button
        type="button"
        disabled={!meta}
        onClick={() => meta && setExpanded((v) => !v)}
        className="flex w-full items-start gap-0 px-4 py-1.5 text-left disabled:cursor-default"
      >
        {visibleCols.includes('timestamp') && (
          <span className="w-[76px] shrink-0 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-[#64748B]">
            {tsShort}
          </span>
        )}
        {visibleCols.includes('level') && (
          <span
            className="w-11 shrink-0 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-bold"
            style={{ color: lc }}
          >
            {level.toUpperCase()}
          </span>
        )}
        {visibleCols.includes('source') && src && (
          <span
            className="mr-2.5 shrink-0 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)]"
            style={{ color: OB_ACCENT }}
          >
            {src}
          </span>
        )}
        <span className="flex-1 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-[#E2E8F0]">
          {msg}
        </span>
        {meta &&
          (expanded ? (
            <ChevronUp size={12} className="text-[#64748B]" />
          ) : (
            <ChevronDown size={12} className="text-[#64748B]" />
          ))}
      </button>
      {expanded && meta && (
        <pre className="ml-[136px] mr-4 mb-1.5 overflow-x-auto rounded-[var(--radius-sm)] bg-white/[0.04] p-2.5 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] leading-[1.5] text-[#94A3B8]">
          {Object.entries(meta)
            .map(([k, v]) => `${k}: ${String(v)}`)
            .join('\n')}
        </pre>
      )}
    </div>
  );
}

function VerticalBars() {
  const heights = [10, 8, 11];
  return (
    <span className="flex items-end gap-0.5">
      {heights.map((h, i) => (
        <span
          key={i}
          className="w-0.5 rounded-[1px] bg-current"
          style={{ height: h }}
        />
      ))}
    </span>
  );
}

function CheckMark() {
  return (
    <svg width={10} height={10} viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth={3}>
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}
