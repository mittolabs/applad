import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  Loader2,
  SkipForward,
  X,
  XCircle,
} from 'lucide-react';
import { api } from '@/api/client';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { IdText } from '@/components/id-text';
import { ErrorState } from '@/components/error-state';
import { GREEN, RED, ACCENT } from './nodeDefs';

interface LogEntry {
  nodeId?: string;
  label?: string;
  nodeType?: string;
  status?: string;
  durationMs?: number;
  input?: unknown;
  output?: unknown;
}

interface Execution {
  $id?: string;
  status?: string;
  durationMs?: number;
  logs?: LogEntry[];
}

export function ExecutionsPanel({
  wfId,
  open,
  onOpenChange,
  onLoad,
}: {
  wfId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onLoad: (exec: Execution) => void;
}) {
  const query = useQuery({
    queryKey: ['workflow-executions', wfId],
    enabled: open,
    queryFn: async () => {
      const res = await api.get(`/workflows/${wfId}/executions`);
      return res.data as Record<string, unknown>;
    },
  });

  const execs = (query.data?.['executions'] as Execution[] | undefined) ?? [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={560} showClose={false}>
        <div className="flex items-center border-b border-border px-5 py-3.5">
          <span className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
            Execution History
          </span>
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            className="ml-auto text-text-secondary hover:text-text-primary"
          >
            <X size={16} />
          </button>
        </div>
        <div className="h-[440px] overflow-y-auto p-4">
          {query.isError ? (
            <ErrorState error={query.error} onRetry={query.refetch} />
          ) : query.isLoading ? (
            <div className="flex h-full items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
            </div>
          ) : execs.length === 0 ? (
            <div className="flex h-full items-center justify-center text-[length:var(--text-control)] text-text-secondary">
              No executions yet
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {execs.map((e, i) => (
                <ExecutionRow
                  key={e.$id ?? i}
                  exec={e}
                  onLoad={() => {
                    onLoad(e);
                    onOpenChange(false);
                  }}
                />
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function statusIcon(status: string | undefined, size: number) {
  if (status === 'completed') return <CheckCircle2 size={size} style={{ color: GREEN }} />;
  if (status === 'failed') return <XCircle size={size} style={{ color: RED }} />;
  return <Clock size={size} style={{ color: ACCENT }} />;
}

function ExecutionRow({ exec, onLoad }: { exec: Execution; onLoad: () => void }) {
  const [expanded, setExpanded] = useState(false);
  const status = exec.status ?? 'pending';
  const dur = exec.durationMs ?? 0;
  const logs = exec.logs ?? [];

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface">
      <div className="flex items-center gap-3 px-3 py-2.5">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="text-text-subtle hover:text-text-secondary"
        >
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </button>
        {statusIcon(status, 18)}
        <div className="min-w-0 flex-1">
          <IdText id={String(exec.$id ?? '')} />
          <div className="text-[length:var(--text-caption)] text-text-subtle">
            {status} • {dur}ms
          </div>
        </div>
        <button
          type="button"
          onClick={onLoad}
          className="text-[length:var(--text-label)]"
          style={{ color: ACCENT }}
        >
          Load
        </button>
      </div>
      {expanded && logs.length > 0 && (
        <div className="border-t border-border px-3 py-2">
          {logs.map((l, i) => (
            <div key={i} className="flex items-center gap-2 py-1">
              {l.status === 'completed' ? (
                <Check size={14} style={{ color: GREEN }} />
              ) : l.status === 'skipped' ? (
                <SkipForward size={14} className="text-text-secondary" />
              ) : (
                <X size={14} style={{ color: RED }} />
              )}
              <span className="flex-1 truncate text-[length:var(--text-label)] text-text-primary">
                {l.label} ({l.nodeType})
              </span>
              <span className="text-[length:var(--text-caption)] text-text-subtle">
                {l.durationMs}ms
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export type { Execution, LogEntry };
