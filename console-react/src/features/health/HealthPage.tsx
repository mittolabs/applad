import { useParams } from 'react-router-dom';
import { HeartPulse, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ErrorState } from '@/components/error-state';
import { HealthCheckCard } from './HealthCheckCard';
import { useHealth, type HealthCheck } from './useHealth';

function OverviewCard({
  status,
  checks,
  timestamp,
}: {
  status: string;
  checks: HealthCheck[];
  timestamp: string;
}) {
  const passCount = checks.filter((c) => c.status === 'pass').length;
  const color =
    status === 'pass'
      ? 'var(--status-success)'
      : status === 'warn'
        ? 'var(--status-warning)'
        : 'var(--status-danger)';
  const headline =
    status === 'pass'
      ? 'All critical services are healthy'
      : status === 'warn'
        ? 'Some services are degraded'
        : 'Health checks are failing';
  const stamp = timestamp ? new Date(timestamp).toLocaleTimeString() : '';

  return (
    <div className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface p-5">
      <div
        className="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-[var(--radius-12)]"
        style={{ color, backgroundColor: `color-mix(in srgb, ${color} 14%, transparent)` }}
      >
        <HeartPulse size={24} />
      </div>
      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="text-[length:var(--text-title)] font-semibold text-text-primary">{headline}</div>
        <div className="text-[length:var(--text-label)] text-text-secondary">
          {passCount} of {checks.length} checks passed{stamp ? ` · Last refresh: ${stamp}` : ''}
        </div>
      </div>
    </div>
  );
}

export function HealthPage() {
  const { projectId } = useParams();
  const health = useHealth();

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-start gap-4">
        <div className="flex flex-1 flex-col gap-1">
          <h1 className="text-[length:var(--text-h2)] font-bold text-text-primary">Health</h1>
          <p className="text-[length:var(--text-body)] text-text-secondary">
            {projectId
              ? `Infrastructure status for project ${projectId}.`
              : 'Infrastructure status for the current workspace.'}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => health.refetch()} loading={health.isFetching}>
          <RefreshCw size={14} />
          Refresh
        </Button>
      </div>

      {health.isLoading ? (
        <div className="flex h-40 items-center justify-center text-[length:var(--text-body)] text-text-muted">
          Loading health data…
        </div>
      ) : health.error ? (
        <ErrorState error={health.error} onRetry={() => health.refetch()} />
      ) : (
        <div className="flex flex-col gap-4">
          <OverviewCard
            status={health.data?.status ?? 'fail'}
            checks={health.data?.checks ?? []}
            timestamp={health.data?.timestamp ?? ''}
          />
          <div className="grid gap-4 lg:grid-cols-2">
            {(health.data?.checks ?? []).map((check) => (
              <HealthCheckCard key={check.path} check={check} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
