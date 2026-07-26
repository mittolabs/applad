import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Circle, Code2, Film, Play, Trash2, Video } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { CodeBlock } from '@/components/code-block';
import { CaptureReplay } from './CaptureReplay';

/*
 * Saved recordings.
 *
 * A flow is kept as steps, and each one also has a suite that runs it, so
 * "record it once" and "run it on every change" are the same object seen from
 * two sides.
 */

interface Flow {
  $id: string;
  name: string;
  platform: string;
  target: string;
  steps: { kind: string; description: string }[];
  suiteId?: string;
}

export function FlowsTab({ projectId, onRecord }: { projectId?: string; onRecord: () => void }) {
  const qc = useQueryClient();
  const [showing, setShowing] = useState<Flow | null>(null);
  const [deleting, setDeleting] = useState<Flow | null>(null);
  const [running, setRunning] = useState<string | null>(null);
  const [replaying, setReplaying] = useState<string | null>(null);
  const [openingReplay, setOpeningReplay] = useState<string | null>(null);

  const openReplay = async (flow: Flow) => {
    setOpeningReplay(flow.$id);
    try {
      const cap = (await api.get(`/studio/flows/${flow.$id}/capture`)).data as { $id: string };
      setReplaying(cap.$id);
    } catch {
      toast.error('This recording has no replay yet.');
    } finally {
      setOpeningReplay(null);
    }
  };

  const { data: flows = [] } = useQuery({
    queryKey: ['test-flows', projectId],
    queryFn: async () => ((await api.get('/studio/flows')).data as { flows: Flow[] }).flows ?? [],
  });

  const { data: code } = useQuery({
    queryKey: ['test-flow-code', showing?.$id],
    queryFn: async () =>
      ((await api.get(`/studio/flows/${showing!.$id}`)).data as { playwright: string }).playwright,
    enabled: !!showing,
  });

  const run = async (flow: Flow) => {
    if (!flow.suiteId) return;
    setRunning(flow.$id);
    try {
      await api.post(`/tests/suites/${flow.suiteId}/run`, { triggerType: 'manual', actor: 'console' });
      toast.success(`Running ${flow.name}`);
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRunning(null);
    }
  };

  const remove = async () => {
    if (!deleting) return;
    try {
      await api.delete(`/studio/flows/${deleting.$id}`);
      toast.success('Discarded');
      qc.invalidateQueries({ queryKey: ['test-flows', projectId] });
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setDeleting(null);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-label)] text-text-secondary">
          {flows.length} recorded {flows.length === 1 ? 'flow' : 'flows'}
        </span>
        <Button onClick={onRecord}>
          <Circle size={12} fill="currentColor" />
          Record a flow
        </Button>
      </div>

      {flows.length === 0 ? (
        <EmptyState
          icon={Video}
          title="No recordings yet"
          subtitle="Open your app, click through what matters, and Applad writes the test."
          actionLabel="Record a flow"
          onAction={onRecord}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {flows.map((f) => (
            <div
              key={f.$id}
              className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
            >
              <div className="min-w-0 flex-1">
                <div className="text-[length:var(--text-body)] text-text-primary">{f.name}</div>
                <div className="mt-0.5 truncate text-[length:var(--text-caption)] text-text-muted">
                  {f.steps?.length ?? 0} steps · {f.target}
                </div>
              </div>
              <Button variant="ghost" loading={openingReplay === f.$id} onClick={() => openReplay(f)}>
                <Film size={13} />
                Replay
              </Button>
              <Button variant="ghost" onClick={() => setShowing(f)}>
                <Code2 size={13} />
                Code
              </Button>
              <Button variant="outline" loading={running === f.$id} onClick={() => run(f)}>
                <Play size={13} />
                Run
              </Button>
              <Button variant="ghost" onClick={() => setDeleting(f)} aria-label="Discard">
                <Trash2 size={13} />
              </Button>
            </div>
          ))}
        </div>
      )}

      {showing && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-[length:var(--text-label)] text-text-secondary">
              {showing.name} — generated Playwright
            </span>
            <button
              onClick={() => setShowing(null)}
              className="text-[length:var(--text-caption)] text-text-muted hover:text-text-primary"
            >
              Close
            </button>
          </div>
          <CodeBlock code={code ?? '// loading...'} language="javascript" />
        </div>
      )}

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDeleting(null)}
        title="Discard this recording?"
        message={`"${deleting?.name}" and the test generated from it will be removed.`}
        confirmLabel="Discard"
        destructive
        onConfirm={remove}
      />

      {replaying && <CaptureReplay captureId={replaying} onClose={() => setReplaying(null)} />}
    </div>
  );
}
