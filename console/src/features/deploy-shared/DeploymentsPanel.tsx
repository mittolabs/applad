import { Fragment, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Play, RefreshCw, Rocket } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { StatusChip } from '@/components/status-chip';
import { IdText } from '@/components/id-text';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { toast } from '@/components/toast';
import {
  asNumber,
  formatBytes,
  formatDuration,
  formatTimestamp,
} from './format';

type Row = Record<string, unknown>;

/* Shared deployments/releases panel for deploy targets (Sites + Containers).
 * Reads /deploy/releases?targetId=…; optionally shows aggregate metrics and a
 * "Create deployment" trigger. Ports the _DeploymentsTab in sites_page.dart
 * and the _releasesTab in containers_page.dart. */

export function useReleases(targetId: string) {
  return useQuery({
    queryKey: ['deploy-releases', targetId],
    queryFn: async () => {
      const res = await api.get('/deploy/releases', { params: { targetId } });
      return res.data as Record<string, unknown>;
    },
  });
}

export function DeploymentsPanel({
  targetId,
  pipelineId,
  showMetrics = true,
  showTrigger = true,
}: {
  targetId: string;
  /** Pipeline used by "Create deployment" (defaults to the target id). */
  pipelineId?: string;
  showMetrics?: boolean;
  showTrigger?: boolean;
}) {
  const query = useReleases(targetId);
  const releases = (query.data?.['releases'] as Row[] | undefined) ?? [];
  // Which deployment is opened out to its build log.
  const [expanded, setExpanded] = useState<string | null>(null);

  const total = releases.length;
  const successful = releases.filter((r) =>
    ['success', 'ready', 'active'].includes(String(r['status'])),
  ).length;
  const failed = releases.filter((r) => r['status'] === 'failed').length;
  const totalDuration = releases.reduce(
    (s, r) => s + asNumber(r['durationMs'] ?? r['buildDuration']),
    0,
  );
  const avgDuration = total > 0 ? Math.round(totalDuration / total) : 0;
  const totalSize = releases.reduce((s, r) => s + asNumber(r['sizeBytes'] ?? r['totalSize']), 0);

  const trigger = async () => {
    try {
      await api.post(`/deploy/pipelines/${pipelineId ?? targetId}/trigger`, {});
      toast.success('Deployment triggered');
      query.refetch();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      {showMetrics && total > 0 && (
        <div className="flex flex-wrap gap-3">
          <MetricBadge label="Total builds" value={String(total)} />
          <MetricBadge label="Total size" value={formatBytes(totalSize)} />
          <MetricBadge label="Total time" value={formatDuration(totalDuration)} />
          <MetricBadge label="Avg time" value={formatDuration(avgDuration)} />
          <MetricBadge label="Successful" value={String(successful)} color="var(--status-success)" />
          <MetricBadge label="Failed" value={String(failed)} color="var(--status-danger)" />
        </div>
      )}

      <div className="flex items-center gap-2">
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">
          Deployments
        </span>
        <div className="ml-auto flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => query.refetch()}>
            <RefreshCw size={14} />
            Refresh
          </Button>
          {showTrigger && (
            <Button size="sm" onClick={trigger}>
              <Play size={14} />
              Create deployment
            </Button>
          )}
        </div>
      </div>

      {query.isLoading ? (
        <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : releases.length === 0 ? (
        <EmptyState icon={Rocket} title="No deployments yet" subtitle="Trigger a deployment to get started." />
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius)] border border-border">
          <table className="w-full min-w-[720px] text-[length:var(--text-body)]">
            <thead>
              <tr className="border-b border-border bg-surface text-left text-text-muted">
                <Th>Deployment ID</Th>
                <Th>Status</Th>
                <Th>Build duration</Th>
                <Th>Total size</Th>
                <Th>Source</Th>
                <Th>Updated</Th>
              </tr>
            </thead>
            <tbody>
              {releases.map((r) => {
                const id = String(r['$id'] ?? r['id'] ?? '');
                const open = expanded === id;
                const log = String(r['buildLog'] ?? '');
                const error = String(r['error'] ?? '');
                return (
                <Fragment key={id}>
                <tr
                  onClick={() => setExpanded(open ? null : id)}
                  className="cursor-pointer border-b border-border last:border-0 hover:bg-fill-hover"
                >
                  <Td>
                    <IdText id={String(r['$id'] ?? r['id'] ?? '')} fontSize={12} />
                  </Td>
                  <Td>
                    <StatusChip label={String(r['status'] ?? 'pending')} />
                  </Td>
                  <Td className="text-text-primary">
                    {formatDuration(r['durationMs'] ?? r['buildDuration'])}
                  </Td>
                  <Td className="text-text-primary">{formatBytes(r['sizeBytes'] ?? r['totalSize'])}</Td>
                  <Td className="text-text-muted">
                    {String(r['triggerType'] ?? r['sourceType'] ?? r['source'] ?? 'git')}
                  </Td>
                  <Td className="text-text-muted">
                    {/* A release records when it finished, not when its row was
                        touched, which is why this read N/A. */}
                    {formatTimestamp(r['completedAt'] ?? r['$createdAt'] ?? r['createdAt'])}
                  </Td>
                </tr>
                {open && (
                  <tr className="border-b border-border bg-surface-alt last:border-0">
                    <td colSpan={6} className="px-4 py-3">
                      {/* The build's own output. It used to be discarded on
                          success and folded into the error on failure, so
                          there was nowhere to see what a deploy actually did. */}
                      {error && (
                        <div
                          className="mb-2 rounded-[var(--radius)] p-2.5 text-[length:var(--text-caption)]"
                          style={{ backgroundColor: '#EF444411', color: '#F87171' }}
                        >
                          {error}
                        </div>
                      )}
                      {log ? (
                        <pre className="max-h-[360px] overflow-auto whitespace-pre-wrap rounded-[var(--radius)] bg-surface p-3 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted">
                          {log}
                        </pre>
                      ) : (
                        !error && (
                          <span className="text-[length:var(--text-caption)] text-text-subtle">
                            No build output recorded for this deployment.
                          </span>
                        )
                      )}
                    </td>
                  </tr>
                )}
                </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function MetricBadge({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface px-3 py-2">
      <div className="text-[length:var(--text-caption)] text-text-subtle">{label}</div>
      <div className="mt-0.5 text-[length:var(--text-control)] font-semibold" style={{ color: color ?? 'var(--text-primary)' }}>
        {value}
      </div>
    </div>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-4 py-2.5 font-semibold">{children}</th>;
}

function Td({ children, className }: { children: React.ReactNode; className?: string }) {
  return <td className={`px-4 py-2.5 ${className ?? ''}`}>{children}</td>;
}
