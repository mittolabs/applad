import { CheckCircle2, Loader2, Terminal, X, XCircle } from 'lucide-react';
import { ACCENT, GREEN, RED } from './nodeDefs';
import type { LogEntry } from './ExecutionsPanel';

/**
 * Dockable bottom strip showing the latest execution's per-step logs.
 * Ports the Flutter `_logsPanel`. Data comes from the most recent Execute (or a
 * loaded past execution); statuses also render live on the nodes themselves.
 */
export function LiveLogsPanel({
  logs,
  onClose,
}: {
  logs: LogEntry[];
  onClose: () => void;
}) {
  return (
    <div className="flex h-[180px] shrink-0 flex-col border-t border-border bg-surface">
      <div className="flex items-center gap-2 px-4 py-2">
        <Terminal size={14} className="text-text-secondary" />
        <span className="text-[length:var(--text-body)] font-semibold text-text-primary">Logs</span>
        <button
          type="button"
          onClick={onClose}
          className="ml-auto text-text-secondary hover:text-text-primary"
        >
          <X size={14} />
        </button>
      </div>
      <div className="h-px bg-border" />
      <div className="min-h-0 flex-1 overflow-y-auto p-3">
        {logs.length === 0 ? (
          <div className="flex h-full items-center justify-center text-[length:var(--text-label)] text-text-subtle">
            Execute workflow to see logs
          </div>
        ) : (
          logs.map((l, i) => (
            <div key={i} className="flex items-center gap-2 py-1">
              {l.status === 'completed' ? (
                <CheckCircle2 size={13} style={{ color: GREEN }} />
              ) : l.status === 'failed' ? (
                <XCircle size={13} style={{ color: RED }} />
              ) : (
                <Loader2 size={13} style={{ color: ACCENT }} />
              )}
              <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
                {(l.label ?? l.nodeId ?? 'node')}: {l.status ?? 'pending'}
                {l.nodeType ? ` (${l.nodeType})` : ''}
              </span>
              {typeof l.durationMs === 'number' && (
                <span className="text-[length:var(--text-caption)] text-text-subtle">
                  {l.durationMs}ms
                </span>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
