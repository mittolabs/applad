import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, ChevronLeft, MinusCircle, Play, Plus, XCircle, AlertTriangle } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { useTabIndex } from '@/hooks/use-tab-param';
import { PageTabs } from '@/components/page-tabs';
import { StatusChip, statusVariant } from '@/components/status-chip';
import { SearchListHeader } from '@/components/search-list';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { IdText } from '@/components/id-text';
import { toast } from '@/components/toast';
import { type Row } from '@/components/data-table';
import { CreateSuiteDialog } from './CreateSuiteDialog';
import { StudioView } from './StudioView';
import { RecordDialog, type StudioSession } from './RecordDialog';
import { FlowsTab } from './FlowsTab';
import { shortDate } from '../deploy-shared/format';

/*
 * Test — run a project's own suite and read what it found.
 *
 * Applad does not know what framework a project uses: a suite says how to run
 * itself and where it leaves a JUnit report, and the report is what gets
 * displayed. Everything here is per test case rather than per run, because a
 * red run is read by looking at what broke.
 */

const LIST_TABS = ['Flows', 'Suites', 'Runs'];

export function TestPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useTabIndex(LIST_TABS);
  const [creating, setCreating] = useState(false);
  const [openRun, setOpenRun] = useState<string | null>(null);
  const [recording, setRecording] = useState(false);
  const [session, setSession] = useState<StudioSession | null>(null);
  const [flowsKey, setFlowsKey] = useState(0);

  // A live recording takes the whole page: it is the app under test.
  if (session) {
    return (
      <StudioView
        session={session}
        onClose={() => setSession(null)}
        onSaved={() => {
          setSession(null);
          setFlowsKey((k) => k + 1);
        }}
      />
    );
  }

  if (openRun) {
    return <RunDetail runId={openRun} onBack={() => setOpenRun(null)} />;
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Test</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Record a flow by using your app, or run a suite you already have
        </p>
      </div>

      <PageTabs tabs={LIST_TABS} selected={tab} onChange={setTab} />

      {tab === 0 && (
        <FlowsTab key={flowsKey} projectId={projectId} onRecord={() => setRecording(true)} />
      )}
      {tab === 1 && (
        <SuitesTab projectId={projectId} creating={creating} setCreating={setCreating} />
      )}
      {tab === 2 && <RunsTab projectId={projectId} onOpen={setOpenRun} />}

      <RecordDialog open={recording} onOpenChange={setRecording} onStarted={setSession} />
    </div>
  );
}

// ── Suites ──

function SuitesTab({
  projectId,
  creating,
  setCreating,
}: {
  projectId?: string;
  creating: boolean;
  setCreating: (v: boolean) => void;
}) {
  const qc = useQueryClient();
  const list = useResourceList<Row>({ endpoint: '/tests/suites', itemsKey: 'suites', scope: [projectId] });
  const [running, setRunning] = useState<string | null>(null);

  const run = async (suiteId: string) => {
    setRunning(suiteId);
    try {
      await api.post(`/tests/suites/${suiteId}/run`, { triggerType: 'manual', actor: 'console' });
      toast.success('Test run queued');
      qc.invalidateQueries({ queryKey: ['resource-list', '/tests/runs'] });
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRunning(null);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        searchHint="Search suites..."
        value={list.search}
        onChange={list.setSearch}
        onSearch={list.runSearch}
        trailing={
          <Button onClick={() => setCreating(true)}>
            <Plus size={14} />
            New suite
          </Button>
        }
      />

      {list.rows.length === 0 ? (
        <EmptyState
          title="No test suites yet"
          subtitle="A suite is a command that runs your tests and leaves a JUnit report behind."
        />
      ) : (
        <div className="flex flex-col gap-2">
          {list.rows.map((s) => (
            <div
              key={String(s.$id)}
              className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
            >
              <div className="min-w-0 flex-1">
                <div className="text-[length:var(--text-body)] text-text-primary">{String(s.name)}</div>
                <div className="mt-0.5 truncate font-mono text-[length:var(--text-caption)] text-text-muted">
                  {String(s.command)}
                </div>
              </div>
              <div className="text-[length:var(--text-caption)] text-text-subtle">{String(s.sourceType)}</div>
              <Button
                variant="outline"
                loading={running === String(s.$id)}
                onClick={() => run(String(s.$id))}
              >
                <Play size={13} />
                Run
              </Button>
            </div>
          ))}
        </div>
      )}

      <CreateSuiteDialog open={creating} onOpenChange={setCreating} onCreated={() => list.refetch()} />
    </div>
  );
}

// ── Runs ──

function RunsTab({ projectId, onOpen }: { projectId?: string; onOpen: (id: string) => void }) {
  const list = useResourceList<Row>({ endpoint: '/tests/runs', itemsKey: 'runs', scope: [projectId] });

  if (list.rows.length === 0) {
    return <EmptyState title="No runs yet" subtitle="Run a suite to see its results here." />;
  }

  return (
    <div className="flex flex-col gap-2">
      {list.rows.map((r) => (
        <button
          key={String(r.$id)}
          onClick={() => onOpen(String(r.$id))}
          className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface px-4 py-3 text-left transition-colors hover:bg-fill-hover"
        >
          <StatusChip label={String(r.status)} variant={statusVariant(String(r.status))} />
          <Counts run={r} />
          <div className="ml-auto flex items-center gap-4 text-[length:var(--text-caption)] text-text-muted">
            <span>{formatDuration(Number(r.durationMs ?? 0))}</span>
            <span>{shortDate(r.$createdAt)}</span>
          </div>
        </button>
      ))}
    </div>
  );
}

function Counts({ run }: { run: Row }) {
  const failed = Number(run.failed ?? 0);
  const passed = Number(run.passed ?? 0);
  const skipped = Number(run.skipped ?? 0);
  return (
    <div className="flex items-center gap-3 text-[length:var(--text-caption)]">
      <span className="text-text-primary">{Number(run.total ?? 0)} tests</span>
      {failed > 0 && <span style={{ color: '#EF4444' }}>{failed} failed</span>}
      <span className="text-text-muted">{passed} passed</span>
      {skipped > 0 && <span className="text-text-subtle">{skipped} skipped</span>}
    </div>
  );
}

// ── One run ──

function RunDetail({ runId, onBack }: { runId: string; onBack: () => void }) {
  const { data: run } = useQuery({
    queryKey: ['test-run', runId],
    queryFn: async () => (await api.get(`/tests/runs/${runId}`)).data as Row,
    // A queued or running suite settles on its own; poll until it does.
    refetchInterval: (q) => {
      const s = String((q.state.data as Row | undefined)?.status ?? '');
      return s === 'queued' || s === 'running' ? 3000 : false;
    },
  });

  const { data: artifacts } = useQuery({
    queryKey: ['test-artifacts', runId],
    queryFn: async () =>
      ((await api.get(`/tests/runs/${runId}/artifacts`)).data as { artifacts: Row[] }).artifacts ?? [],
    enabled: !!run && !['queued', 'running'].includes(String(run.status)),
  });

  const { data: cases } = useQuery({
    queryKey: ['test-cases', runId],
    queryFn: async () =>
      ((await api.get(`/tests/runs/${runId}/cases`)).data as { cases: Row[] }).cases ?? [],
    enabled: !!run && !['queued', 'running'].includes(String(run.status)),
  });

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <button
        onClick={onBack}
        className="flex w-fit items-center gap-1 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-primary"
      >
        <ChevronLeft size={14} />
        All runs
      </button>

      <div className="flex items-center gap-4">
        <StatusChip label={String(run?.status ?? 'queued')} variant={statusVariant(String(run?.status ?? 'queued'))} />
        {run && <Counts run={run} />}
        <IdText id={runId} />
      </div>

      {run?.error ? (
        <div
          className="rounded-[var(--radius)] border p-3 text-[length:var(--text-label)]"
          style={{ borderColor: '#EF444455', backgroundColor: '#EF444411', color: '#F87171' }}
        >
          {String(run.error)}
        </div>
      ) : null}

      {cases && cases.length > 0 && (
        <div className="flex flex-col gap-1.5">
          {cases.map((c) => (
            <CaseRow
              key={String(c.$id)}
              c={c}
              artifacts={(artifacts ?? []).filter((a) => a.caseId === c.$id)}
            />
          ))}
        </div>
      )}

      {/* Evidence that could not be traced to one test — a combined report, a
          trace covering the whole suite. */}
      {(artifacts ?? []).some((a) => !a.caseId) && (
        <div>
          <div className="mb-2 text-[length:var(--text-label)] text-text-secondary">Recordings</div>
          <div className="grid gap-3 md:grid-cols-2">
            {(artifacts ?? [])
              .filter((a) => !a.caseId)
              .map((a) => (
                <ArtifactView key={String(a.$id)} a={a} />
              ))}
          </div>
        </div>
      )}

      {run?.log ? (
        <div>
          <div className="mb-2 text-[length:var(--text-label)] text-text-secondary">Output</div>
          <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded-[var(--radius)] border border-border bg-surface-alt p-3 font-mono text-[length:var(--text-caption)] text-text-muted">
            {String(run.log)}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

const CASE_ICON = {
  passed: { Icon: CheckCircle2, color: '#22C55E' },
  failed: { Icon: XCircle, color: '#EF4444' },
  errored: { Icon: AlertTriangle, color: '#F59E0B' },
  skipped: { Icon: MinusCircle, color: '#6B7280' },
} as const;

function ArtifactView({ a }: { a: Row }) {
  const kind = String(a.kind);
  const [src, setSrc] = useState<string | null>(null);

  /*
   * Fetched through the API client rather than pointed at by the media
   * element: a <video src> is a plain browser request, carrying neither the
   * bearer token nor the project header, so the file would come back 401.
   * Recordings are small enough that loading them whole costs nothing.
   */
  useEffect(() => {
    if (kind !== 'video' && kind !== 'screenshot') return;
    let url: string | null = null;
    let cancelled = false;
    api
      .get(`/tests/artifacts/${String(a.$id)}`, { responseType: 'blob' })
      .then((res) => {
        if (cancelled) return;
        url = URL.createObjectURL(res.data as Blob);
        setSrc(url);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [a.$id, kind]);

  return (
    <div className="overflow-hidden rounded-[var(--radius)] border border-border bg-surface-alt">
      {kind === 'video' && src && (
        <video src={src} controls preload="metadata" className="w-full bg-black" />
      )}
      {kind === 'screenshot' && src && <img src={src} alt={String(a.name)} className="w-full" />}
      {!src && (kind === 'video' || kind === 'screenshot') && (
        <div className="flex h-[160px] items-center justify-center text-[length:var(--text-caption)] text-text-subtle">
          Loading recording...
        </div>
      )}
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <span className="truncate font-mono text-[length:var(--text-caption)] text-text-muted">
          {String(a.name)}
        </span>
        <span className="shrink-0 text-[length:var(--text-caption)] text-text-subtle">
          {formatBytes(Number(a.sizeBytes ?? 0))}
        </span>
      </div>
    </div>
  );
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function CaseRow({ c, artifacts = [] }: { c: Row; artifacts?: Row[] }) {
  const status = String(c.status) as keyof typeof CASE_ICON;
  const { Icon, color } = CASE_ICON[status] ?? CASE_ICON.skipped;
  const failure = String(c.failureMessage ?? '');
  const details = String(c.failureDetails ?? '');
  const [open, setOpen] = useState(false);

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface">
      <button
        onClick={() => (details || artifacts.length > 0) && setOpen(!open)}
        className="flex w-full items-start gap-2.5 px-3 py-2 text-left"
        style={{ cursor: details || artifacts.length > 0 ? 'pointer' : 'default' }}
      >
        <Icon size={14} style={{ color, marginTop: 2, flexShrink: 0 }} />
        <div className="min-w-0 flex-1">
          <div className="text-[length:var(--text-label)] text-text-primary">{String(c.name)}</div>
          <div className="mt-0.5 truncate text-[length:var(--text-caption)] text-text-subtle">
            {String(c.suiteName)}
          </div>
          {failure && (
            <div className="mt-1 text-[length:var(--text-caption)]" style={{ color: '#F87171' }}>
              {failure}
            </div>
          )}
        </div>
        <span className="shrink-0 text-[length:var(--text-caption)] text-text-subtle">
          {formatDuration(Number(c.durationMs ?? 0))}
        </span>
      </button>
      {open && details && (
        <pre className="mx-3 mb-3 overflow-auto whitespace-pre-wrap rounded-[var(--radius-6)] bg-surface-alt p-2.5 font-mono text-[length:var(--text-caption)] text-text-muted">
          {details}
        </pre>
      )}
      {artifacts.length > 0 && (
        <div className="mx-3 mb-3 grid gap-2 md:grid-cols-2">
          {artifacts.map((a) => (
            <ArtifactView key={String(a.$id)} a={a} />
          ))}
        </div>
      )}
    </div>
  );
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
}
