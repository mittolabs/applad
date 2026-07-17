import { useNavigate } from 'react-router-dom';
import {
  AlertTriangle,
  CheckCircle2,
  Gauge,
  HeartPulse,
  type LucideIcon,
  Terminal,
  Timer,
} from 'lucide-react';
import { Loader2 } from 'lucide-react';
import { EmptyState } from '@/components/empty-state';
import {
  OB_ACCENT,
  OB_GREEN,
  OB_ORANGE,
  OB_PURPLE,
  OB_RED,
  ObSectionTitle,
  apdexColor,
  asRecord,
  asRows,
  levelColor,
  num,
  obFmtNum,
  obTimeAgo,
  tint,
  useObserveResource,
} from './observe-shared';

/* ObserveOverview — ports observe_overview.dart. Summary dashboard combining
 * the overview + errors resources. On error it renders nothing (matching the
 * Flutter SizedBox), on empty sections it shows placeholder EmptyStates. */

export function ObserveOverview({ projectId }: { projectId?: string }) {
  const overview = useObserveResource('/observe/overview', projectId);
  const errorsQ = useObserveResource('/observe/errors', projectId, { limit: 50 });

  if (overview.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin" style={{ color: OB_ACCENT }} />
      </div>
    );
  }
  if (overview.error) return null;

  const data = overview.data ?? {};
  const stats = asRecord(data.stats);
  const services = asRows(data.services);
  const vitals = asRecord(data.vitals);
  const recentErrors = asRows(errorsQ.data?.errors).slice(0, 5);

  return (
    <div className="overflow-y-auto px-6 py-6 md:px-8">
      {/* Stat cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <StatCard
          label="Errors (24h)"
          value={obFmtNum(stats.errorsToday ?? 0)}
          color={OB_RED}
          icon={AlertTriangle}
        />
        <StatCard
          label="P95 Latency"
          value={`${num(stats.p95Ms)}ms`}
          color={OB_ACCENT}
          icon={Timer}
        />
        <StatCard
          label="Uptime"
          value={`${num(stats.uptimePct, 100)}%`}
          color={OB_GREEN}
          icon={HeartPulse}
        />
        <StatCard
          label="Apdex"
          value={num(stats.apdex, 1.0).toFixed(2)}
          color={apdexColor(stats.apdex)}
          icon={Gauge}
        />
        <StatCard
          label="Log volume (1h)"
          value={obFmtNum(stats.logsLastHour ?? 0)}
          color={OB_PURPLE}
          icon={Terminal}
        />
      </div>

      {/* Web Vitals */}
      {Object.keys(vitals).length > 0 && (
        <div className="mt-7">
          <ObSectionTitle title="Web Vitals" />
          <div className="mt-3 flex flex-wrap gap-2.5">
            {WEB_VITALS.map((v) => (
              <VitalCard key={v.name} spec={v} value={vitals[v.key]} />
            ))}
          </div>
        </div>
      )}

      {/* Service health */}
      <div className="mt-7">
        <ObSectionTitle title="Service Health" />
        <div className="mt-3">
          {services.length === 0 ? (
            <EmptyState
              icon={HeartPulse}
              title="No services configured yet"
              subtitle="Register uptime monitors to track your service health here."
            />
          ) : (
            <div className="flex flex-wrap gap-3">
              {services.map((s, i) => (
                <ServiceCard key={i} service={s} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Recent errors */}
      <RecentErrors projectId={projectId} errors={recentErrors} />
    </div>
  );
}

function StatCard({
  label,
  value,
  color,
  icon: Icon,
}: {
  label: string;
  value: string;
  color: string;
  icon: LucideIcon;
}) {
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-4 py-3">
      <div
        className="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded-[var(--radius)]"
        style={{ backgroundColor: tint(color, 10), color }}
      >
        <Icon size={15} />
      </div>
      <div className="flex min-w-0 flex-col">
        <span className="text-[length:var(--text-title)] font-bold text-text-primary">{value}</span>
        <span className="truncate text-[length:var(--text-caption)] text-text-secondary">
          {label}
        </span>
      </div>
    </div>
  );
}

interface VitalSpec {
  name: string;
  key: string;
  unit: string;
  good: number;
  poor: number;
  description: string;
}

const WEB_VITALS: VitalSpec[] = [
  { name: 'LCP', key: 'lcp', unit: 'ms', good: 2500, poor: 4000, description: 'Largest Contentful Paint' },
  { name: 'FID', key: 'fid', unit: 'ms', good: 100, poor: 300, description: 'First Input Delay' },
  { name: 'CLS', key: 'cls', unit: '', good: 0.1, poor: 0.25, description: 'Cumulative Layout Shift' },
  { name: 'TTFB', key: 'ttfb', unit: 'ms', good: 800, poor: 1800, description: 'Time to First Byte' },
  { name: 'FCP', key: 'fcp', unit: 'ms', good: 1800, poor: 3000, description: 'First Contentful Paint' },
];

function vitalColor(spec: VitalSpec, value: unknown): string {
  const v = typeof value === 'number' ? value : 0;
  if (v <= spec.good) return OB_GREEN;
  if (v <= spec.poor) return OB_ORANGE;
  return OB_RED;
}
function vitalRating(spec: VitalSpec, value: unknown): string {
  const v = typeof value === 'number' ? value : 0;
  if (v <= spec.good) return 'Good';
  if (v <= spec.poor) return 'Needs improvement';
  return 'Poor';
}
function vitalDisplay(spec: VitalSpec, value: unknown): string {
  if (value == null) return '—';
  const v = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(v)) return '—';
  if (spec.unit === '') return v.toFixed(3);
  return `${Math.round(v)}${spec.unit}`;
}

function VitalCard({ spec, value }: { spec: VitalSpec; value: unknown }) {
  const color = vitalColor(spec, value);
  return (
    <div className="min-w-[150px] flex-1 rounded-[var(--radius)] border border-border bg-surface p-3.5">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-body)] font-semibold text-text-primary">
          {spec.name}
        </span>
        <span
          className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-semibold"
          style={{ color, backgroundColor: tint(color, 10) }}
        >
          {vitalRating(spec, value)}
        </span>
      </div>
      <div
        className="mt-1.5 font-[family-name:var(--font-mono)] text-[length:var(--text-h2)] font-bold"
        style={{ color }}
      >
        {vitalDisplay(spec, value)}
      </div>
      <div className="mt-0.5 text-[length:var(--text-caption)] text-text-subtle">
        {spec.description}
      </div>
    </div>
  );
}

function ServiceCard({ service }: { service: Record<string, unknown> }) {
  const status = String(service.status ?? 'healthy');
  const map: Record<string, [string, string]> = {
    healthy: [OB_GREEN, 'Healthy'],
    degraded: [OB_ORANGE, 'Degraded'],
    down: [OB_RED, 'Down'],
  };
  const [dot, label] = map[status] ?? [OB_GREEN, 'Unknown'];
  return (
    <div className="w-[180px] rounded-[var(--radius)] border border-border bg-surface p-3.5">
      <div className="flex items-center gap-1.5">
        <span className="h-2 w-2 rounded-full" style={{ backgroundColor: dot }} />
        <span className="text-[length:var(--text-caption)] font-medium" style={{ color: dot }}>
          {label}
        </span>
      </div>
      <div className="mt-2 text-[length:var(--text-body)] font-medium text-text-primary">
        {String(service.name ?? 'Service')}
      </div>
      <div className="mt-0.5 text-[length:var(--text-caption)] text-text-subtle">
        {num(service.latencyMs)}ms &nbsp;•&nbsp; {num(service.uptime, 100)}% uptime
      </div>
    </div>
  );
}

function RecentErrors({
  projectId,
  errors,
}: {
  projectId?: string;
  errors: Record<string, unknown>[];
}) {
  const navigate = useNavigate();
  return (
    <div className="mt-7">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-control)] font-semibold text-text-primary">
          Recent Errors
        </span>
        <button
          type="button"
          onClick={() => projectId && navigate(`/project/${projectId}/errors`)}
          className="text-[length:var(--text-label)]"
          style={{ color: OB_ACCENT }}
        >
          View all
        </button>
      </div>
      <div className="mt-3">
        {errors.length === 0 ? (
          <EmptyState
            icon={CheckCircle2}
            title="No errors — great job!"
            subtitle="Errors captured by the Applad SDK will appear here."
          />
        ) : (
          <div className="flex flex-col gap-1.5">
            {errors.map((e, i) => {
              const c = levelColor(String(e.level ?? 'error'));
              return (
                <div
                  key={i}
                  className="flex items-center gap-2.5 rounded-[var(--radius)] border border-border bg-surface px-3.5 py-2.5"
                >
                  <span
                    className="h-[7px] w-[7px] shrink-0 rounded-full"
                    style={{ backgroundColor: c }}
                  />
                  <span className="flex-1 truncate text-[length:var(--text-body)] text-text-primary">
                    {String(e.title ?? 'Unknown error')}
                  </span>
                  <span className="text-[length:var(--text-label)] text-text-subtle">
                    {obFmtNum(e.count ?? 0)}
                  </span>
                  <span className="text-[length:var(--text-caption)] text-text-subtle">
                    {obTimeAgo(e.lastSeen)}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
