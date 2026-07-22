import { useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, Rocket, Clock, Hand } from 'lucide-react';
import { api } from '@/api/client';
import { ArtifactPlayer } from './ArtifactPlayer';

/*
 * One test, across every run it has appeared in.
 *
 * The catalogue shows a shape; this shows what actually happened, with the
 * recording of each attempt. A failure you can watch is a failure you can
 * usually explain without reproducing it.
 */

interface HistoryEntry {
  runId: string;
  status: string;
  durationMs: number;
  flaky: boolean;
  failureMessage?: string;
  failureDetails?: string;
  targetUrl?: string;
  triggerType?: string;
  at: string;
  videoId?: string;
}

const DOT: Record<string, string> = {
  passed: '#22C55E',
  failed: '#EF4444',
  errored: '#F59E0B',
  skipped: '#4B5563',
};

const TRIGGER_ICON: Record<string, typeof Rocket> = {
  deploy: Rocket,
  schedule: Clock,
  manual: Hand,
};

export function TestDetail({ testId, onBack }: { testId: string; onBack: () => void }) {
  const [open, setOpen] = useState<string | null>(null);

  // The name comes with the catalogue rather than the caller, so the page can
  // be opened directly from a link.
  const { data: name = '' } = useQuery({
    queryKey: ['test-name', testId],
    queryFn: async () => {
      const tests = ((await api.get('/tests/tests')).data as { tests: { $id: string; name: string }[] })
        .tests ?? [];
      return tests.find((t) => t.$id === testId)?.name ?? 'Test';
    },
  });

  const { data: history = [] } = useQuery({
    queryKey: ['test-history', testId],
    queryFn: async () =>
      ((await api.get(`/tests/tests/${testId}/history`)).data as { history: HistoryEntry[] })
        .history ?? [],
    // A run in progress will add to this shortly.
    refetchInterval: 5000,
  });

  const running = history.length === 0;

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <button
          onClick={onBack}
          className="mb-1 flex items-center gap-1 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-primary"
        >
          <ChevronLeft size={14} />
          Tests
        </button>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">{name}</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          {running ? 'Not run yet' : `${history.length} runs`}
        </p>
      </div>

      <div className="flex flex-col gap-2">
        {history.map((h) => {
          const Icon = TRIGGER_ICON[h.triggerType ?? 'manual'] ?? Hand;
          const isOpen = open === h.runId;
          return (
            <div key={h.runId} className="rounded-[var(--radius)] border border-border bg-surface">
              <button
                onClick={() => setOpen(isOpen ? null : h.runId)}
                className="flex w-full items-center gap-3 px-4 py-3 text-left"
              >
                <span
                  className="h-2 w-2 shrink-0 rounded-full"
                  style={{ backgroundColor: DOT[h.status] ?? 'var(--text-subtle)' }}
                />
                <span className="text-[length:var(--text-label)] text-text-primary">{h.status}</span>
                {h.flaky && (
                  <span
                    className="rounded-[var(--radius-6)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                    style={{ backgroundColor: '#F59E0B22', color: '#FBBF24' }}
                  >
                    flaky
                  </span>
                )}
                <span className="flex items-center gap-1 text-[length:var(--text-caption)] text-text-muted">
                  <Icon size={11} />
                  {h.triggerType}
                </span>
                <span className="ml-auto truncate text-[length:var(--text-caption)] text-text-subtle">
                  {h.targetUrl}
                </span>
                <span className="shrink-0 text-[length:var(--text-caption)] text-text-muted">
                  {new Date(h.at).toLocaleString()}
                </span>
              </button>

              {isOpen && (
                <div className="flex flex-col gap-3 px-4 pb-4">
                  {h.failureMessage && (
                    <div
                      className="rounded-[var(--radius)] p-2.5 text-[length:var(--text-caption)]"
                      style={{ backgroundColor: '#EF444411', color: '#F87171' }}
                    >
                      {h.failureMessage}
                    </div>
                  )}
                  {h.failureDetails && (
                    <pre className="max-h-[240px] overflow-auto whitespace-pre-wrap rounded-[var(--radius)] bg-surface-alt p-2.5 font-mono text-[length:var(--text-caption)] text-text-muted">
                      {h.failureDetails}
                    </pre>
                  )}
                  {h.videoId ? (
                    <div className="max-w-[560px]">
                      <ArtifactPlayer artifactId={h.videoId} kind="video" />
                    </div>
                  ) : (
                    <span className="text-[length:var(--text-caption)] text-text-subtle">
                      No recording for this run.
                    </span>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/*
 * Live output from a run in progress.
 *
 * The suite's own output is streamed as it is produced, so a run that spends
 * four minutes installing dependencies shows what it is doing rather than a
 * spinner.
 */
export function LiveRunLog({ runId }: { runId: string }) {
  const [lines, setLines] = useState<string[]>([]);
  const boxRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const token = localStorage.getItem('applad_console_token') ?? '';
    const project = api.defaults.headers.common['X-Applad-Project'] as string;
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const socket = new WebSocket(
      `${scheme}://${window.location.host}/v1/tests/runs/${runId}/stream?token=${encodeURIComponent(token)}&project=${encodeURIComponent(project)}`,
    );
    socket.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'line') setLines((prev) => [...prev.slice(-500), msg.line]);
    };
    return () => socket.close();
  }, [runId]);

  useEffect(() => {
    boxRef.current?.scrollTo({ top: boxRef.current.scrollHeight });
  }, [lines]);

  return (
    <div>
      <div className="mb-2 flex items-center gap-2 text-[length:var(--text-label)] text-text-secondary">
        <span className="h-1.5 w-1.5 animate-pulse rounded-full" style={{ backgroundColor: '#22C55E' }} />
        Running
      </div>
      <pre
        ref={boxRef}
        className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded-[var(--radius)] border border-border bg-surface-alt p-3 font-mono text-[length:var(--text-caption)] text-text-muted"
      >
        {lines.length > 0 ? lines.join('\n') : 'Waiting for output...'}
      </pre>
    </div>
  );
}
