import { useNavigate } from 'react-router-dom';
import {
  Activity,
  BarChart3,
  Gauge,
  HeartPulse,
  Loader2,
  type LucideIcon,
  Timer,
  Users,
} from 'lucide-react';
import { EmptyState } from '@/components/empty-state';
import {
  ACCENT,
  Bar,
  GREEN,
  INFO,
  LineChart,
  PURPLE,
  SectionTitle,
  apdexColor,
  asRecord,
  asRows,
  fmtNum,
  healthColor,
  metric,
  num,
  tint,
  useAnalyticsResource,
} from './analytics-shared';

/* AnalyticsOverview — the Analytics landing page: how much the product is
 * being used, and whether the platform underneath it is answering. */

export function AnalyticsOverview({ projectId }: { projectId?: string }) {
  const overview = useAnalyticsResource('/analytics/overview', projectId);

  if (overview.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin" style={{ color: ACCENT }} />
      </div>
    );
  }
  if (overview.error) return null;

  const data = overview.data ?? {};
  const stats = asRecord(data.stats);
  const topEvents = asRows(data.topEvents);
  const dau = asRows(data.dau);

  return (
    <div className="overflow-y-auto px-6 py-6 md:px-8">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <StatCard
          label="Events (24h)"
          value={fmtNum(stats.eventsToday ?? 0)}
          color={ACCENT}
          icon={BarChart3}
        />
        <StatCard
          label="Active users (24h)"
          value={fmtNum(stats.activeUsers ?? 0)}
          color={PURPLE}
          icon={Users}
        />
        <StatCard
          label="Sessions (24h)"
          value={fmtNum(stats.activeSessions ?? 0)}
          color={INFO}
          icon={Activity}
        />
        <StatCard
          label="P95 latency"
          value={metric(stats.p95Ms, { suffix: 'ms', digits: 0 })}
          color={ACCENT}
          icon={Timer}
        />
        <StatCard
          label="Uptime"
          value={metric(stats.uptimePct, { suffix: '%', digits: 2 })}
          color={healthColor(stats.uptimePct)}
          icon={HeartPulse}
        />
      </div>

      {stats.apdex != null && (
        <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          <StatCard
            label="Apdex"
            value={metric(stats.apdex, { digits: 2 })}
            color={apdexColor(stats.apdex)}
            icon={Gauge}
          />
        </div>
      )}

      <DailyActiveUsers dau={dau} />
      <TopEvents projectId={projectId} events={topEvents} />
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

function DailyActiveUsers({ dau }: { dau: Record<string, unknown>[] }) {
  const points = dau.map((d) => num(d.users));
  return (
    <div className="mt-7">
      <SectionTitle title="Daily active users" />
      <div className="mt-3 rounded-[var(--radius)] border border-border bg-surface p-4">
        {points.length < 2 ? (
          <div className="py-6 text-center text-[length:var(--text-body)] text-text-muted">
            Not enough history yet — two days of events draw a line here.
          </div>
        ) : (
          <>
            <div className="h-28">
              <LineChart points={points} color={GREEN} />
            </div>
            <div className="mt-2 flex justify-between text-[length:var(--text-caption)] text-text-subtle">
              <span>{String(dau[0]?.date ?? '')}</span>
              <span>{String(dau[dau.length - 1]?.date ?? '')}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function TopEvents({
  projectId,
  events,
}: {
  projectId?: string;
  events: Record<string, unknown>[];
}) {
  const navigate = useNavigate();
  const max = events.reduce((m, e) => Math.max(m, num(e.count)), 0);
  return (
    <div className="mt-7">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-control)] font-semibold text-text-primary">
          Top events (7d)
        </span>
        <button
          type="button"
          onClick={() => projectId && navigate(`/project/${projectId}/events`)}
          className="text-[length:var(--text-label)]"
          style={{ color: ACCENT }}
        >
          View all
        </button>
      </div>
      <div className="mt-3">
        {events.length === 0 ? (
          <EmptyState
            icon={BarChart3}
            title="No events yet"
            subtitle="Call analytics.trackEvent() from your app and the events land here."
          />
        ) : (
          <div className="flex flex-col gap-2.5">
            {events.map((e, i) => (
              <div
                key={i}
                className="rounded-[var(--radius)] border border-border bg-surface px-3.5 py-2.5"
              >
                <div className="flex items-center gap-2.5">
                  <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary">
                    {String(e.event ?? '')}
                  </span>
                  <span className="text-[length:var(--text-label)] text-text-subtle">
                    {fmtNum(e.count ?? 0)}
                  </span>
                </div>
                <div className="mt-2">
                  <Bar pct={max > 0 ? (num(e.count) / max) * 100 : 0} color={ACCENT} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
