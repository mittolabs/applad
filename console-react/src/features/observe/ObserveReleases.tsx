import { useMemo, useState } from 'react';
import {
  AlertCircle,
  ArrowLeft,
  CheckCircle2,
  Circle,
  Tag,
  TrendingUp,
} from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import {
  OB_ACCENT,
  OB_GREEN,
  OB_ORANGE,
  OB_RED,
  OB_SLATE,
  ObEmptyCard,
  ObMetaBadge,
  ObSectionTitle,
  asRows,
  num,
  obTimeAgo,
  useObserveResource,
} from './observe-shared';

/* ObserveReleases — ports observe_releases.dart. Release list correlating
 * crash-free rate and issue counts, plus a commit + issue detail view. */

const COLUMNS: DataTableColumn[] = [
  { key: 'version', label: 'Version', flex: 4 },
  { key: 'crashFree', label: 'Crash-free', flex: 2, sortable: false },
  { key: 'newIssues', label: 'New issues', flex: 2, sortable: false },
  { key: 'regressed', label: 'Regressed', flex: 2, sortable: false },
  { key: 'fixed', label: 'Fixed', flex: 2, sortable: false },
  { key: 'commits', label: 'Commits', flex: 2, sortable: false },
  { key: 'created', label: 'Created', flex: 2, sortable: false },
];

export function ObserveReleases({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/releases', projectId);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  const allReleases = asRows(query.data?.releases);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allReleases;
    return allReleases.filter(
      (r) =>
        String(r.version ?? '').toLowerCase().includes(q) ||
        String(r.environment ?? '').toLowerCase().includes(q),
    );
  }, [allReleases, search]);

  if (selectedId) {
    const rel = allReleases.find((r) => r.$id === selectedId || r.version === selectedId);
    return <ReleaseDetail release={rel} onBack={() => setSelectedId(null)} />;
  }

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'version':
              return String(row.version ?? '');
            case 'crashFree':
              return `${num(row.crashFreeSessionsPct, 100).toFixed(2)}%`;
            case 'newIssues':
              return String(row.newIssues ?? 0);
            case 'regressed':
              return String(row.regressedIssues ?? 0);
            case 'fixed':
              return String(row.fixedIssues ?? 0);
            case 'commits':
              return String(row.commitCount ?? 0);
            case 'created':
              return obTimeAgo(row.createdAt);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'version') {
            const v = String(row.version ?? '—');
            const env = String(row.environment ?? '');
            return (
              <span className="inline-flex items-center gap-2">
                <Tag size={12} className="text-text-subtle" />
                <span
                  className="font-[family-name:var(--font-mono)] text-[length:var(--text-body)] font-medium"
                  style={{ color: OB_ACCENT }}
                >
                  {v}
                </span>
                {env && <ObMetaBadge label={env} color={OB_ACCENT} />}
              </span>
            );
          }
          if (key === 'crashFree') {
            const pct = num(row.crashFreeSessionsPct, 100);
            const c = pct >= 99 ? OB_GREEN : pct >= 95 ? OB_ORANGE : OB_RED;
            return (
              <span className="text-[length:var(--text-label)] font-semibold" style={{ color: c }}>
                {pct.toFixed(2)}%
              </span>
            );
          }
          if (key === 'newIssues') return <CountCell n={num(row.newIssues)} pos={OB_ORANGE} />;
          if (key === 'regressed') return <CountCell n={num(row.regressedIssues)} pos={OB_RED} />;
          if (key === 'fixed') return <CountCell n={num(row.fixedIssues)} pos={OB_GREEN} />;
          return undefined;
        }}
        rowIcon={() => Tag}
        rowIconColor={() => OB_ACCENT}
        onRowClick={(row) => {
          const id = String(row.$id ?? row.version ?? '');
          if (id) setSelectedId(id);
        }}
        searchHint="Search releases…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="releases"
        emptyIcon={Tag}
        emptyTitle="No releases yet"
        emptySubtitle="Use the SDK to create a release and track its health"
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />
    </div>
  );
}

function CountCell({ n, pos }: { n: number; pos: string }) {
  return (
    <span
      className="text-[length:var(--text-label)]"
      style={{ color: n > 0 ? pos : 'var(--text-secondary)' }}
    >
      {n}
    </span>
  );
}

/* ── Release detail ───────────────────────────────────────────────────────── */

function ReleaseDetail({
  release,
  onBack,
}: {
  release: Record<string, unknown> | undefined;
  onBack: () => void;
}) {
  if (!release) {
    return (
      <div className="flex justify-center py-20 text-[length:var(--text-body)] text-text-secondary">
        Release not found
      </div>
    );
  }

  const version = String(release.version ?? '—');
  const commits = asRows(release.commits);
  const issues = asRows(release.issues);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center gap-4 border-b border-border px-6 py-3.5 md:px-8">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-[length:var(--text-body)] text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft size={14} />
          All releases
        </button>
        <Tag size={14} style={{ color: OB_ACCENT }} />
        <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-subhead)] font-semibold text-text-primary">
          {version}
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-6 md:px-8 lg:flex-row">
        <div className="min-w-0 flex-[2]">
          <ObSectionTitle title={`Commits (${commits.length})`} />
          <div className="mt-3">
            {commits.length === 0 ? (
              <ObEmptyCard message="No commits linked" />
            ) : (
              commits.map((c, i) => <CommitRow key={i} commit={c} />)
            )}
          </div>
        </div>

        <div className="min-w-0 flex-1">
          <ObSectionTitle title="Issues" />
          <div className="mt-3 flex flex-col gap-1.5">
            {issues.length === 0 ? (
              <ObEmptyCard message="No issues for this release" />
            ) : (
              issues.map((iss, i) => <IssueRow key={i} issue={iss} />)
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function CommitRow({ commit }: { commit: Record<string, unknown> }) {
  const sha = String(commit.sha ?? '').slice(0, 7);
  const message = String(commit.message ?? '');
  const author = String(commit.author ?? '');
  return (
    <div className="mb-1.5 flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-3.5 py-2.5">
      <span
        className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-semibold"
        style={{ color: OB_ACCENT }}
      >
        {sha}
      </span>
      <span className="flex-1 truncate text-[length:var(--text-label)] text-text-primary">
        {message}
      </span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">{author}</span>
    </div>
  );
}

function IssueRow({ issue }: { issue: Record<string, unknown> }) {
  const title = String(issue.title ?? 'Issue');
  const type = String(issue.type ?? 'new');
  const color =
    type === 'new' ? OB_ORANGE : type === 'regressed' ? OB_RED : type === 'fixed' ? OB_GREEN : OB_SLATE;
  const Icon =
    type === 'new' ? AlertCircle : type === 'regressed' ? TrendingUp : type === 'fixed' ? CheckCircle2 : Circle;
  return (
    <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-surface p-3">
      <Icon size={13} style={{ color }} />
      <span className="flex-1 truncate text-[length:var(--text-label)] text-text-primary">
        {title}
      </span>
      <span className="text-[length:var(--text-caption)] font-medium" style={{ color }}>
        {type}
      </span>
    </div>
  );
}
