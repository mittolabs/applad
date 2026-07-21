import { useState } from 'react';
import { useTabIndex } from '@/hooks/use-tab-param';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  Check,
  ChevronDown,
  Copy,
  Database,
  FolderClosed,
  GitBranch,
  Link as LinkIcon,
  Loader2,
  Mail,
  Rocket,
  Users,
  Zap,
  type LucideIcon,
} from 'lucide-react';
import { api } from '@/api/client';
import { useProject } from '@/api/queries';
import { PageTabs } from '@/components/page-tabs';
import { IdText } from '@/components/id-text';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { UsageChart } from './UsageChart';

/* Ports console/lib/features/overview/overview_page.dart.
 * Read-only project dashboard: usage charts, stat cards, info cards,
 * recent deployments, and a services grid. Two tabs (Overview / Activity).
 * All numbers come from a fan-out of best-effort GETs (each failure = 0). */

const TABS = ['Overview', 'Activity'];

// --- Data --------------------------------------------------------------------

interface Stats {
  databases: number;
  functions: number;
  buckets: number;
  users: number;
  deployments: number;
  workflows: number;
  usage: Record<string, unknown>;
  releases: Record<string, unknown>[];
}

function countKey(data: unknown, key: string): number {
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    if (Array.isArray(obj[key])) return (obj[key] as unknown[]).length;
    if (typeof obj.total === 'number') return obj.total;
  }
  return 0;
}

/** Pull the first array found under any of the given keys. */
function firstList(data: unknown, keys: string[]): Record<string, unknown>[] {
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    for (const k of keys) {
      if (Array.isArray(obj[k])) return obj[k] as Record<string, unknown>[];
    }
  }
  return [];
}

function useProjectStats(projectId: string) {
  return useQuery<Stats>({
    queryKey: ['overview-stats', projectId],
    queryFn: async () => {
      const safeFetch = async (path: string): Promise<unknown> => {
        try {
          const res = await api.get(path);
          return res.data;
        } catch {
          return null;
        }
      };

      const [
        databases,
        functions,
        buckets,
        users,
        targets,
        workflows,
        usage,
        deploy,
      ] = await Promise.all([
        safeFetch('/databases'),
        safeFetch('/functions'),
        safeFetch('/storage/buckets'),
        safeFetch('/account/users'),
        safeFetch('/deploy/targets'),
        safeFetch('/workflows'),
        safeFetch(`/projects/${projectId}/usage`),
        safeFetch('/deploy/releases?limit=5'),
      ]);

      return {
        databases: countKey(databases, 'databases'),
        functions: countKey(functions, 'functions'),
        buckets: countKey(buckets, 'buckets'),
        users: countKey(users, 'users'),
        deployments: countKey(targets, 'targets'),
        workflows: countKey(workflows, 'workflows'),
        usage: (usage as Record<string, unknown>) ?? {},
        releases: firstList(deploy, ['deployments', 'releases', 'targets']),
      };
    },
  });
}

// --- Formatting helpers ------------------------------------------------------

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return `${n}`;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(4, Math.floor(Math.log(bytes) / Math.log(1024)));
  const v = bytes / Math.pow(1024, i);
  return `${v.toFixed(v < 10 ? 2 : 1)} ${units[i]}`;
}

function toNum(v: unknown): number {
  return typeof v === 'number' ? v : 0;
}

function graphData(history: unknown, points: number): number[] {
  if (Array.isArray(history) && history.length > 0) {
    return history
      .slice(0, points)
      .map((e) => (typeof e === 'number' ? e : 0));
  }
  return Array<number>(points).fill(0);
}

function timeAgo(iso: unknown): string {
  if (typeof iso !== 'string' || !iso) return '';
  const dt = new Date(iso);
  if (Number.isNaN(dt.getTime())) return '';
  const mins = Math.floor((Date.now() - dt.getTime()) / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return `${Math.floor(days / 7)}w ago`;
}

// --- Page --------------------------------------------------------------------

export function OverviewPage() {
  const { projectId } = useParams<{ projectId: string }>();
  // In the URL so a refresh stays on the tab somebody was reading.
  const [tab, setTab] = useTabIndex(TABS, undefined, 'view');
  const { data: project } = useProject(projectId);

  if (!projectId) {
    return (
      <div className="flex h-full items-center justify-center p-8 text-[length:var(--text-subhead)] text-text-muted">
        Select a project
      </div>
    );
  }

  const projectName = project?.name ?? 'Project';

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-end gap-3">
        <h1 className="text-[length:var(--text-h2)] font-bold text-text-primary">
          {projectName}
        </h1>
        <div className="mb-[3px] flex items-center gap-1">
          <FolderClosed size={13} className="text-text-subtle" />
          <IdText id={projectId} />
        </div>
      </div>

      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      {tab === 0 ? (
        <OverviewTab projectId={projectId} />
      ) : (
        <ActivityTab projectId={projectId} />
      )}
    </div>
  );
}

// --- Overview tab ------------------------------------------------------------

function OverviewTab({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const { data: stats, isLoading, error, refetch } = useProjectStats(projectId);

  if (isLoading) {
    return (
      <div className="flex justify-center pt-20">
        <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
      </div>
    );
  }
  if (error) return <ErrorState error={error} onRetry={refetch} />;
  if (!stats) return null;

  const { usage } = stats;
  const go = (seg: string) => navigate(`/project/${projectId}/${seg}`);

  return (
    <div className="flex flex-col gap-4">
      {/* Usage charts */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <UsageGraph
          title="Requests"
          value={formatNumber(toNum(usage.requests))}
          data={graphData(usage.requestsHistory, 30)}
        />
        <UsageGraph
          title="Bandwidth"
          value={formatBytes(toNum(usage.bandwidth))}
          data={graphData(usage.bandwidthHistory, 30)}
        />
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          icon={Database}
          label="DATABASE"
          value={`${stats.databases}`}
          sublabel="Databases"
          onClick={() => go('databases')}
        />
        <StatCard
          icon={FolderClosed}
          label="STORAGE"
          value={formatBytes(toNum(usage.storageBytes))}
          sublabel="Storage"
          onClick={() => go('storage')}
        />
        <StatCard
          icon={Users}
          label="AUTH"
          value={`${stats.users}`}
          sublabel="Users"
          onClick={() => go('auth')}
        />
        <StatCard
          icon={Zap}
          label="FUNCTIONS"
          value={`${stats.functions}`}
          sublabel="Executions"
          onClick={() => go('functions')}
        />
      </div>

      {/* Info cards */}
      <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
        <InfoCard icon={FolderClosed} label="Project ID" value={projectId} />
        <InfoCard
          icon={LinkIcon}
          label="API Endpoint"
          value={`${window.location.origin}/v1`}
        />
      </div>

      <RecentDeployments releases={stats.releases} onViewAll={() => go('deploy')} />

      <ServicesGrid stats={stats} onNavigate={go} />
    </div>
  );
}

// --- Usage graph -------------------------------------------------------------

function UsageGraph({
  title,
  value,
  data,
  period = '30d',
}: {
  title: string;
  value: string;
  data: number[];
  period?: string;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <div className="flex items-start">
        <div>
          <div className="text-[28px] font-bold leading-none text-text-primary">
            {value}
          </div>
          <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
            {title}
          </div>
        </div>
        <div className="ml-auto flex items-center gap-1 rounded-[var(--radius-6)] border border-border bg-fill px-2.5 py-1 text-[length:var(--text-label)] text-text-secondary">
          {period}
          <ChevronDown size={12} />
        </div>
      </div>
      <div className="mt-5">
        <UsageChart data={data} height={100} />
      </div>
    </div>
  );
}

// --- Stat card ---------------------------------------------------------------

function StatCard({
  icon: Icon,
  label,
  value,
  sublabel,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  sublabel: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col items-start rounded-[var(--radius)] border border-border bg-surface p-5 text-left transition-colors hover:border-[var(--fill-active)] hover:bg-fill-hover"
    >
      <div className="flex items-center gap-2">
        <Icon size={14} className="text-[var(--color-accent)]" />
        <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
          {label}
        </span>
      </div>
      <div className="mt-4 text-[28px] font-bold leading-none text-text-primary">
        {value}
      </div>
      <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
        {sublabel}
      </div>
    </button>
  );
}

// --- Info card ---------------------------------------------------------------

function InfoCard({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className="flex items-center gap-2.5 rounded-[var(--radius)] border border-border bg-surface p-4">
      <Icon size={14} className="text-text-secondary" />
      <div className="min-w-0 flex-1">
        <div className="text-[length:var(--text-caption)] font-medium text-text-subtle">
          {label}
        </div>
        <div className="mt-0.5 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary">
          {value}
        </div>
      </div>
      <button
        type="button"
        onClick={copy}
        aria-label={`Copy ${label}`}
        className="text-text-subtle transition-colors hover:text-text-primary"
      >
        {copied ? (
          <Check size={14} className="text-status-success" />
        ) : (
          <Copy size={14} />
        )}
      </button>
    </div>
  );
}

// --- Recent deployments ------------------------------------------------------

/* Deployment-status pill — mirrors overview_page.dart `_statusColor`/`_statusLabel`.
 * Deliberately NOT the shared StatusChip: the overview colors in-progress builds
 * amber and rollbacks purple, which the generic StatusChip map doesn't do. */
function deployStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return '#22C55E';
    case 'failed':
      return '#EF4444';
    case 'building':
    case 'deploying':
      return '#F59E0B';
    case 'rolled_back':
      return '#8B5CF6';
    default:
      return '#6B7280';
  }
}

function deployStatusLabel(status: string): string {
  if (status === 'rolled_back') return 'Rolled back';
  return status ? status[0].toUpperCase() + status.slice(1) : status;
}

function DeployStatusChip({ status }: { status: string }) {
  const color = deployStatusColor(status);
  return (
    <span
      className="inline-flex items-center gap-[5px] rounded-[var(--radius-sm)] px-2 py-[3px] text-[length:var(--text-caption)] font-medium"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
    >
      <span className="h-[5px] w-[5px] rounded-full" style={{ backgroundColor: color }} />
      {deployStatusLabel(status)}
    </span>
  );
}

function RecentDeployments({
  releases,
  onViewAll,
}: {
  releases: Record<string, unknown>[];
  onViewAll: () => void;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <div className="flex items-center">
        <h2 className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
          Recent Deployments
        </h2>
        <button
          type="button"
          onClick={onViewAll}
          className="ml-auto text-[length:var(--text-body)] text-[var(--color-accent)] hover:underline"
        >
          View all
        </button>
      </div>

      {releases.length === 0 ? (
        <div className="flex items-center gap-2.5 py-4">
          <Rocket size={16} className="text-text-muted" />
          <span className="text-[length:var(--text-body)] text-text-muted">
            No deployments yet
          </span>
        </div>
      ) : (
        <div className="mt-4 flex flex-col gap-2.5">
          {releases.map((rel, i) => {
            const status = String(rel.status ?? 'pending');
            const name = String(rel.name ?? rel.$id ?? 'Release');
            const time = timeAgo(rel.createdAt ?? rel.$createdAt);
            return (
              <div key={String(rel.$id ?? i)} className="flex items-center gap-3">
                <span className="min-w-0 flex-1 truncate text-[length:var(--text-body)] text-text-primary">
                  {name}
                </span>
                <DeployStatusChip status={status} />
                {time && (
                  <span className="text-[length:var(--text-label)] text-text-muted">
                    {time}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// --- Services grid -----------------------------------------------------------

interface ServiceDef {
  icon: LucideIcon;
  label: string;
  value: string;
  sublabel: string;
  seg: string;
}

function ServicesGrid({
  stats,
  onNavigate,
}: {
  stats: Stats;
  onNavigate: (seg: string) => void;
}) {
  const services: ServiceDef[] = [
    { icon: Users, label: 'Auth', value: `${stats.users}`, sublabel: 'users', seg: 'auth' },
    { icon: Database, label: 'Databases', value: `${stats.databases}`, sublabel: 'databases', seg: 'databases' },
    { icon: FolderClosed, label: 'Storage', value: `${stats.buckets}`, sublabel: 'buckets', seg: 'storage' },
    { icon: Zap, label: 'Functions', value: `${stats.functions}`, sublabel: 'functions', seg: 'functions' },
    { icon: Rocket, label: 'Deploy', value: `${stats.deployments}`, sublabel: 'targets', seg: 'deploy' },
    { icon: GitBranch, label: 'Workflows', value: `${stats.workflows}`, sublabel: 'workflows', seg: 'workflows' },
    { icon: Mail, label: 'Messaging', value: '—', sublabel: 'email · sms · push', seg: 'messaging' },
  ];

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <h2 className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        Services
      </h2>
      <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {services.map((s) => (
          <button
            key={s.seg}
            type="button"
            onClick={() => onNavigate(s.seg)}
            className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-fill px-3.5 py-3 text-left transition-colors hover:border-[var(--fill-active)] hover:bg-fill-hover"
          >
            <span
              className="flex h-8 w-8 items-center justify-center rounded-[7px]"
              style={{
                backgroundColor: 'color-mix(in srgb, var(--color-accent) 10%, transparent)',
              }}
            >
              <s.icon size={15} className="text-[var(--color-accent)]" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-[length:var(--text-body)] font-medium text-text-primary">
                {s.label}
              </div>
              <div className="text-[length:var(--text-caption)] text-text-muted">
                {s.sublabel}
              </div>
            </div>
            <span className="text-[length:var(--text-subhead)] font-semibold text-text-secondary">
              {s.value}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

// --- Activity tab ------------------------------------------------------------

function ActivityTab({ projectId }: { projectId: string }) {
  const { data: stats, isLoading } = useProjectStats(projectId);

  if (isLoading) {
    return (
      <div className="flex justify-center pt-20">
        <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
      </div>
    );
  }
  if (!stats) return null;

  const defs: [number, string, string, string, LucideIcon][] = [
    [stats.databases, 'database created', 'databases created', 'Databases', Database],
    [stats.users, 'user registered', 'users registered', 'Auth', Users],
    [stats.buckets, 'bucket created', 'buckets created', 'Storage', FolderClosed],
    [stats.functions, 'function deployed', 'functions deployed', 'Functions', Zap],
    [stats.deployments, 'deployment active', 'deployments active', 'Deploy', Rocket],
    [stats.workflows, 'workflow configured', 'workflows configured', 'Workflows', GitBranch],
  ];

  const items = defs
    .filter(([count]) => count > 0)
    .map(([count, singular, plural, service, icon]) => ({
      title: `${count} ${count === 1 ? singular : plural}`,
      subtitle: service,
      icon,
    }));

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Activity}
        title="No activity yet"
        subtitle="Activity will appear here as you use your project"
        className="pt-20"
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {items.map((item) => (
        <div
          key={item.subtitle}
          className="flex items-center gap-3.5 rounded-[var(--radius)] border border-border bg-surface p-4"
        >
          <span
            className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)]"
            style={{
              backgroundColor: 'color-mix(in srgb, var(--color-accent) 10%, transparent)',
            }}
          >
            <item.icon size={18} className="text-[var(--color-accent)]" />
          </span>
          <div>
            <div className="text-[length:var(--text-control)] font-medium text-text-primary">
              {item.title}
            </div>
            <div className="mt-0.5 text-[length:var(--text-label)] text-text-secondary">
              {item.subtitle}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
