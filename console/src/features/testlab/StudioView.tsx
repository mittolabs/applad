import { useCallback, useEffect, useRef, useState } from 'react';
import { ChevronLeft, Crosshair, MousePointer2, Save, Trash2, X } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { FormDialog, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';

/*
 * The recording studio.
 *
 * A real browser runs server-side; its frames arrive here and the clicks and
 * keystrokes go back. Every interaction returns as a step with a durable
 * selector, so clicking through the app is what writes the test — the point
 * being that nobody has to know Playwright to produce a Playwright test.
 */

interface Step {
  kind: string;
  description: string;
  value?: string;
  target?: { role?: string; name?: string; text?: string; css?: string; testId?: string };
}

interface Session {
  $id: string;
  target: string;
  status: string;
}

const STEP_LABEL: Record<string, string> = {
  goto: 'Open',
  tap: 'Tap',
  type: 'Type',
  press: 'Press',
  expectVisible: 'Expect',
  expectText: 'Expect',
  expectURL: 'Expect',
};

export function StudioView({
  session,
  onClose,
  onSaved,
}: {
  session: Session;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [steps, setSteps] = useState<Step[]>([]);
  const [frame, setFrame] = useState<string | null>(null);
  const [size, setSize] = useState({ width: 1280, height: 800 });
  const [connected, setConnected] = useState(false);
  const [assertMode, setAssertMode] = useState(false);
  const [saving, setSaving] = useState(false);
  const [naming, setNaming] = useState(false);
  const [name, setName] = useState('');

  const ws = useRef<WebSocket | null>(null);
  const imgRef = useRef<HTMLImageElement>(null);

  useEffect(() => {
    // A WebSocket carries no headers, so the token and project ride along in
    // the query string; the API accepts them there for exactly this reason.
    const token = localStorage.getItem('applad_console_token') ?? '';
    const project = api.defaults.headers.common['X-Applad-Project'] as string;
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${scheme}://${window.location.host}/v1/studio/sessions/${session.$id}/stream?token=${encodeURIComponent(token)}&project=${encodeURIComponent(project)}`;

    const socket = new WebSocket(url);
    ws.current = socket;
    socket.onopen = () => setConnected(true);
    socket.onclose = () => setConnected(false);
    socket.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'frame') {
        setFrame(msg.data);
        if (msg.width) setSize({ width: msg.width, height: msg.height });
      } else if (msg.type === 'step') {
        setSteps((prev) => [...prev, msg.step]);
      } else if (msg.type === 'steps') {
        setSteps(msg.steps ?? []);
      }
    };
    return () => socket.close();
  }, [session.$id]);

  const send = useCallback((msg: Record<string, unknown>) => {
    if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(JSON.stringify(msg));
  }, []);

  // The frame is displayed scaled; a click has to be mapped back to where it
  // landed in the real browser or the recording is meaningless.
  const onCanvasClick = (e: React.MouseEvent<HTMLImageElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    send({
      type: 'click',
      x: ((e.clientX - rect.left) / rect.width) * size.width,
      y: ((e.clientY - rect.top) / rect.height) * size.height,
    });
  };

  const onCanvasScroll = (e: React.WheelEvent<HTMLImageElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    send({
      type: 'scroll',
      x: ((e.clientX - rect.left) / rect.width) * size.width,
      y: ((e.clientY - rect.top) / rect.height) * size.height,
      deltaY: e.deltaY,
    });
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      // Only while the canvas has focus, so typing a flow's name does not go
      // to the page under test.
      if (document.activeElement !== imgRef.current) return;
      e.preventDefault();
      send({ type: 'key', key: e.key, text: e.key.length === 1 ? e.key : '' });
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [send]);

  const toggleAssert = () => {
    const next = !assertMode;
    setAssertMode(next);
    send({ type: 'assertMode', assertMode: next });
  };

  const save = async () => {
    setSaving(true);
    try {
      await api.post(`/studio/sessions/${session.$id}/save`, { name: name.trim() });
      toast.success(`Saved ${name.trim()}`);
      onSaved();
    } catch (e) {
      toast.error(friendlyError(e));
      setSaving(false);
    }
  };

  const discard = async () => {
    try {
      await api.delete(`/studio/sessions/${session.$id}`);
    } catch {
      // Closing is best effort: the session expires with its browser anyway.
    }
    onClose();
  };

  // The opening "goto" is not something the user did, so it does not count
  // towards having recorded anything.
  const recorded = steps.filter((s) => s.kind !== 'goto').length;

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <button
            onClick={discard}
            className="mb-1 flex items-center gap-1 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-primary"
          >
            <ChevronLeft size={14} />
            Test
          </button>
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Recording</h1>
          <p className="mt-1 font-mono text-[length:var(--text-caption)] text-text-secondary">
            {session.target}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={toggleAssert}>
            {assertMode ? <Crosshair size={14} /> : <MousePointer2 size={14} />}
            {assertMode ? 'Asserting' : 'Interacting'}
          </Button>
          <Button variant="ghost" onClick={discard}>
            <Trash2 size={14} />
            Discard
          </Button>
          <Button onClick={() => setNaming(true)} disabled={recorded === 0}>
            <Save size={14} />
            Save as test
          </Button>
        </div>
      </div>

      <div
        className="rounded-[var(--radius)] border p-3 text-[length:var(--text-label)]"
        style={
          assertMode
            ? { borderColor: '#F59E0B55', backgroundColor: '#F59E0B11', color: '#FBBF24' }
            : { borderColor: 'var(--border)', backgroundColor: 'var(--fill)', color: 'var(--text-muted)' }
        }
      >
        {assertMode
          ? 'Assert mode: click anything you want the test to check. The page will not respond.'
          : 'Click and type as a visitor would. Every interaction becomes a step.'}
      </div>

      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="min-w-0 flex-1">
          {frame ? (
            <img
              ref={imgRef}
              src={`data:image/jpeg;base64,${frame}`}
              alt="The page being recorded"
              tabIndex={0}
              onClick={onCanvasClick}
              onWheel={onCanvasScroll}
              className="w-full cursor-crosshair rounded-[var(--radius)] border border-border outline-none focus:border-[var(--color-accent)]"
            />
          ) : (
            <div className="flex h-[420px] items-center justify-center rounded-[var(--radius)] border border-border bg-surface text-[length:var(--text-label)] text-text-subtle">
              {connected ? 'Waiting for the first frame...' : 'Connecting to the browser...'}
            </div>
          )}
        </div>

        <div className="flex w-full flex-col gap-2 lg:w-[380px]">
          <div className="flex items-center justify-between">
            <span className="text-[length:var(--text-label)] text-text-secondary">
              Steps ({recorded})
            </span>
          </div>

          {steps.length === 0 ? (
            <div className="rounded-[var(--radius)] border border-border bg-surface p-4 text-[length:var(--text-caption)] text-text-subtle">
              Nothing recorded yet.
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              {steps.map((s, i) => (
                <div
                  key={i}
                  className="group flex items-start gap-2 rounded-[var(--radius)] border border-border bg-surface px-3 py-2"
                >
                  <span
                    className="mt-px shrink-0 rounded-[var(--radius-6)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                    style={
                      s.kind.startsWith('expect')
                        ? { backgroundColor: '#F59E0B22', color: '#FBBF24' }
                        : { backgroundColor: 'var(--fill)', color: 'var(--text-muted)' }
                    }
                  >
                    {STEP_LABEL[s.kind] ?? s.kind}
                  </span>
                  <span className="min-w-0 flex-1 text-[length:var(--text-caption)] text-text-primary">
                    {s.description}
                  </span>
                  <button
                    onClick={() => send({ type: 'deleteStep', index: i })}
                    className="shrink-0 rounded p-0.5 text-text-subtle opacity-0 transition-opacity hover:text-text-primary group-hover:opacity-100"
                    aria-label="Remove this step"
                  >
                    <X size={12} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <FormDialog
        open={naming}
        onOpenChange={setNaming}
        title="Save as test"
        subtitle="Name it for what it checks, not what it clicks."
        submitLabel="Save"
        loading={saving}
        submitDisabled={!name.trim()}
        onSubmit={save}
      >
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="a visitor can reach the about page"
          autoFocus
          hint="Saved as a runnable Playwright test, and kept as steps so it can also compile for a device later."
        />
      </FormDialog>
    </div>
  );
}
