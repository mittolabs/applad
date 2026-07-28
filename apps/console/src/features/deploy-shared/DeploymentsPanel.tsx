import { Fragment, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertCircle, Play, RefreshCw, Rocket } from 'lucide-react';
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

/** Statuses that mean a build is still running, so the panel keeps refreshing. */
const IN_PROGRESS = ['pending', 'queued', 'building', 'deploying', 'running'];

function isInProgress(status: unknown): boolean {
  return IN_PROGRESS.includes(String(status));
}

export function useReleases(targetId: string) {
  return useQuery({
    queryKey: ['deploy-releases', targetId],
    queryFn: async () => {
      const res = await api.get('/deploy/releases', { params: { targetId } });
      return res.data as Record<string, unknown>;
    },
    // While anything is building, refresh so status transitions and the final
    // canonical log land without the operator hitting Refresh. The live stream
    // fills the gap between polls.
    refetchInterval: (query) => {
      const releases = (query.state.data?.['releases'] as Row[] | undefined) ?? [];
      return releases.some((r) => isInProgress(r['status'])) ? 2000 : false;
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
  /** Pipeline used by "Create deployment". When absent, the target's pipeline is resolved from /deploy/pipelines. */
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
      // POST /deploy/pipelines/{id}/trigger wants a PIPELINE id, not a target
      // id. When no pipeline was passed in, resolve the target's pipeline from
      // the project-wide list rather than substituting the target id (which 404s).
      let id = pipelineId;
      if (!id) {
        const res = await api.get('/deploy/pipelines');
        const pipelines = ((res.data as Record<string, unknown>)['pipelines'] as Row[] | undefined) ?? [];
        const match = pipelines.find((p) => String(p['targetId'] ?? '') === targetId);
        id = match ? String(match['$id'] ?? match['id'] ?? '') : '';
        if (!id) {
          toast.error('No deployment pipeline is configured for this target.');
          return;
        }
      }
      await api.post(`/deploy/pipelines/${id}/trigger`, {});
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
                      {error && <FailureNotice error={error} />}
                      <LiveBuildLog
                        releaseId={id}
                        initialLog={log}
                        live={isInProgress(r['status'])}
                        hasError={!!error}
                      />
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

/*
 * A deployment's build output, streamed while it runs.
 *
 * The persisted build_log (initialLog) is whatever the worker has flushed so
 * far; while the build is live we also subscribe to the realtime channel the
 * worker pushes each line to, and append lines as they arrive — so the operator
 * watches progress the way they would on fly.io, instead of a blank until the
 * build ends. When the poll returns the final canonical log, live lines are
 * dropped in favour of it.
 */
function LiveBuildLog({
  releaseId,
  initialLog,
  live,
  hasError,
}: {
  releaseId: string;
  initialLog: string;
  live: boolean;
  hasError: boolean;
}) {
  const [liveLines, setLiveLines] = useState<string[]>([]);
  const preRef = useRef<HTMLPreElement>(null);

  // Reset the transient live buffer whenever the persisted log advances (a poll
  // landed) or the build finishes, so we never show a line twice.
  useEffect(() => {
    setLiveLines([]);
  }, [initialLog, live]);

  useEffect(() => {
    if (!live) return;
    const token = localStorage.getItem('applad_console_token') ?? '';
    const project = (api.defaults.headers.common['X-Applad-Project'] as string) ?? '';
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${scheme}://${window.location.host}/v1/realtime?project=${encodeURIComponent(
      project,
    )}&token=${encodeURIComponent(token)}`;
    const channel = `deploy.${releaseId}`;

    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(url);
    } catch {
      return; // no live stream; the poll still fills in the log
    }
    socket.onopen = () => socket?.send(JSON.stringify({ type: 'subscribe', channels: [channel] }));
    socket.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data as string);
        if (msg.channel !== channel) return;
        const line = msg.payload?.line;
        if (typeof line === 'string') setLiveLines((prev) => [...prev, line]);
      } catch {
        // ignore malformed frames
      }
    };
    return () => {
      try {
        socket?.close();
      } catch {
        // already closed
      }
    };
  }, [releaseId, live]);

  // Auto-scroll to the newest output.
  useEffect(() => {
    const el = preRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [initialLog, liveLines]);

  const text = [initialLog, liveLines.join('\n')].filter(Boolean).join('\n');

  if (!text) {
    if (hasError) return null;
    if (live) {
      return (
        <span className="text-[length:var(--text-caption)] text-text-subtle">
          Waiting for build output…
        </span>
      );
    }
    return (
      <span className="text-[length:var(--text-caption)] text-text-subtle">
        No build output recorded for this deployment.
      </span>
    );
  }

  return (
    <pre
      ref={preRef}
      className="max-h-[360px] overflow-auto whitespace-pre-wrap rounded-[var(--radius)] bg-surface p-3 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted"
    >
      {text}
      {live && <span className="ml-1 animate-pulse text-[var(--color-accent)]">▍</span>}
    </pre>
  );
}

/*
 * Why a deployment failed.
 *
 * The backend sends a headline — which command failed and how — followed by
 * that command's own last words. Both used to arrive as one string rendered as
 * prose, so the newlines collapsed and the whole build stream ran together as a
 * red paragraph. The headline reads as a sentence; the output below it is the
 * tool's, so it keeps the tool's formatting.
 */
function FailureNotice({ error }: { error: string }) {
  const newline = error.indexOf('\n');
  const headline = newline === -1 ? error : error.slice(0, newline);
  const detail = newline === -1 ? '' : error.slice(newline + 1);

  return (
    <div
      className="mb-2 rounded-[var(--radius)] border p-3"
      style={{ backgroundColor: '#EF44440E', borderColor: '#EF444433' }}
    >
      <div className="flex items-start gap-2">
        <AlertCircle size={14} className="mt-px shrink-0" style={{ color: '#F87171' }} />
        <span
          className="text-[length:var(--text-label)] font-medium"
          style={{ color: '#F87171' }}
        >
          {headline}
        </span>
      </div>
      {detail && (
        <pre
          className="mt-2 max-h-[180px] overflow-auto whitespace-pre-wrap pl-[22px] font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] leading-relaxed"
          style={{ color: '#FCA5A5' }}
        >
          {detail}
        </pre>
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
