import { useState } from 'react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { TextAreaField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import { MethodBadge } from './EndpointList';

interface TestResult {
  statusCode: number;
  body: unknown;
  text: string;
  isText: boolean;
  error: string;
  logs: { nodeType: string; label: string; status: string; durationMs: number; error?: string }[];
}

/** Runs the endpoint against a sample request and shows the response plus the
 * per-block trace. Saves pending edits first, so it tests what is on screen. */
export function TestPanel({
  endpointId,
  method,
  path,
  dirty,
  onSaveFirst,
}: {
  endpointId: string;
  method: string;
  path: string;
  dirty: boolean;
  onSaveFirst: () => Promise<boolean>;
}) {
  const [body, setBody] = useState('{\n  \n}');
  const [testPath, setTestPath] = useState(path);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<TestResult | null>(null);

  const run = async () => {
    setRunning(true);
    setResult(null);
    try {
      if (dirty) {
        const ok = await onSaveFirst();
        if (!ok) {
          setRunning(false);
          return;
        }
      }
      let parsedBody: unknown = undefined;
      if (body.trim()) {
        try {
          parsedBody = JSON.parse(body);
        } catch {
          toast.error('Request body is not valid JSON');
          setRunning(false);
          return;
        }
      }
      const res = await api.post(`/endpoints/${endpointId}/test`, {
        method,
        path: testPath,
        body: parsedBody,
      });
      setResult(res.data as TestResult);
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="flex flex-col gap-4 p-5">
      <div className="flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <MethodBadge method={method} />
          <TextField value={testPath} onChange={(e) => setTestPath(e.target.value)} placeholder={path} />
        </div>
        <TextAreaField
          label="Request body (JSON)"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={6}
          className="font-mono"
        />
        <div className="flex items-center gap-3">
          <Button onClick={() => void run()} loading={running}>
            Run test
          </Button>
          {dirty && (
            <span className="text-[length:var(--text-caption)] text-text-muted">Saves your edits first.</span>
          )}
        </div>
      </div>

      {result && (
        <div className="flex flex-col gap-4 border-t border-border pt-4">
          <div className="flex items-center gap-2">
            <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
              Response
            </span>
            <span
              className={cn(
                'rounded-[var(--radius-sm)] px-2 py-0.5 font-mono text-[length:var(--text-caption)] font-semibold',
                result.statusCode >= 400
                  ? 'bg-[color-mix(in_srgb,var(--color-danger)_14%,transparent)] text-[var(--color-danger)]'
                  : 'bg-[color-mix(in_srgb,var(--status-success)_14%,transparent)] text-[var(--status-success)]',
              )}
            >
              {result.statusCode}
            </span>
          </div>
          <pre className="max-h-56 overflow-auto rounded-[var(--radius)] border border-border bg-background p-3 font-mono text-[length:var(--text-caption)] text-text-primary">
            {result.isText ? result.text : JSON.stringify(result.body, null, 2)}
          </pre>

          <div>
            <div className="mb-2 text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
              Trace
            </div>
            <div className="flex flex-col gap-1">
              {result.logs.map((l, i) => (
                <div
                  key={i}
                  className="flex items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5 text-[length:var(--text-caption)]"
                >
                  <span
                    className={cn(
                      'h-1.5 w-1.5 shrink-0 rounded-full',
                      l.status === 'failed'
                        ? 'bg-[var(--color-danger)]'
                        : l.status === 'skipped'
                          ? 'bg-fill-active'
                          : 'bg-[var(--status-success)]',
                    )}
                  />
                  <span className="text-text-primary">{l.label || l.nodeType}</span>
                  <span className="text-text-muted">{l.nodeType}</span>
                  <span className="ml-auto text-text-muted">{l.durationMs}ms</span>
                  {l.error && <span className="text-[var(--color-danger)]">{l.error}</span>}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
