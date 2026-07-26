import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ChevronLeft,
  Crosshair,
  Maximize2,
  Minimize2,
  MousePointer2,
  Save,
  Trash2,
  X,
} from 'lucide-react';
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
  const [fullscreen, setFullscreen] = useState(false);
  const [railOpen, setRailOpen] = useState(true);

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

  // Wheel is bound natively and non-passively so we can preventDefault: a React
  // onWheel cannot stop the console page from scrolling underneath the view,
  // which is why scrolling "did not work" before. preventDefault keeps the page
  // still; the delta is forwarded to the browser instead.
  useEffect(() => {
    const el = imgRef.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const rect = el.getBoundingClientRect();
      send({
        type: 'scroll',
        x: ((e.clientX - rect.left) / rect.width) * size.width,
        y: ((e.clientY - rect.top) / rect.height) * size.height,
        deltaY: e.deltaY,
      });
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, [send, size.width, size.height, frame, fullscreen]);

  // Ask the browser to match the size we are showing, so the picture is 1:1 and
  // clicks land where they look like they should. Re-sent whenever the layout
  // that determines the view size changes.
  const reportViewport = useCallback(() => {
    const el = imgRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    if (r.width > 0 && r.height > 0) {
      send({ type: 'viewport', width: Math.round(r.width), height: Math.round(r.height) });
    }
  }, [send]);

  useEffect(() => {
    // Give the layout a beat to settle after a fullscreen/rail change.
    const t = setTimeout(reportViewport, 60);
    window.addEventListener('resize', reportViewport);
    return () => {
      clearTimeout(t);
      window.removeEventListener('resize', reportViewport);
    };
  }, [reportViewport, fullscreen, railOpen, connected]);

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

  const modeButton = (
    <Button variant="outline" onClick={toggleAssert}>
      {assertMode ? <Crosshair size={14} /> : <MousePointer2 size={14} />}
      {assertMode ? 'Asserting' : 'Interacting'}
    </Button>
  );

  const frameArea = frame ? (
    <img
      ref={imgRef}
      src={`data:image/jpeg;base64,${frame}`}
      alt="The page being recorded"
      tabIndex={0}
      onClick={onCanvasClick}
      onLoad={reportViewport}
      className={
        fullscreen
          ? 'max-h-full max-w-full cursor-crosshair rounded-[var(--radius)] outline-none'
          : 'w-full cursor-crosshair rounded-[var(--radius)] border border-border outline-none focus:border-[var(--color-accent)]'
      }
    />
  ) : (
    <div className="flex h-[420px] w-full items-center justify-center rounded-[var(--radius)] border border-border bg-surface text-[length:var(--text-label)] text-text-subtle">
      {connected ? 'Waiting for the first frame…' : 'Connecting to the browser…'}
    </div>
  );

  const stepsRail = (
    <div className="flex w-full flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-label)] text-text-secondary">Steps ({recorded})</span>
        {fullscreen && (
          <button
            onClick={() => setRailOpen(false)}
            className="rounded p-0.5 text-text-subtle transition-colors hover:text-text-primary"
            aria-label="Hide steps"
          >
            <X size={14} />
          </button>
        )}
      </div>
      {steps.length === 0 ? (
        <div className="rounded-[var(--radius)] border border-border bg-surface p-4 text-[length:var(--text-caption)] text-text-subtle">
          Nothing recorded yet.
        </div>
      ) : (
        <div className="flex flex-col gap-1.5 overflow-y-auto">
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
  );

  // Fullscreen: the recorded page fills the window with a slim toolbar, and the
  // steps become a collapsible right-hand overlay. This is what makes scrolling
  // and precise clicking actually work — the view is near 1:1 instead of shrunk
  // beside the sidebar.
  if (fullscreen) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-background">
        <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-2">
          <span className="truncate font-mono text-[length:var(--text-caption)] text-text-secondary">
            {session.target}
          </span>
          <div className="flex items-center gap-2">
            {modeButton}
            {!railOpen && (
              <Button variant="ghost" onClick={() => setRailOpen(true)}>
                Steps ({recorded})
              </Button>
            )}
            <Button variant="ghost" onClick={() => setNaming(true)} disabled={recorded === 0}>
              <Save size={14} />
              Save
            </Button>
            <Button variant="outline" onClick={() => setFullscreen(false)}>
              <Minimize2 size={14} />
              Exit
            </Button>
          </div>
        </div>
        <div className="flex min-h-0 flex-1">
          <div className="flex min-w-0 flex-1 items-center justify-center overflow-hidden bg-black/40 p-3">
            {frameArea}
          </div>
          {railOpen && (
            <div className="w-[320px] shrink-0 overflow-y-auto border-l border-border p-3">
              {stepsRail}
            </div>
          )}
        </div>
      </div>
    );
  }

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
          {modeButton}
          <Button variant="outline" onClick={() => setFullscreen(true)} title="Fullscreen">
            <Maximize2 size={14} />
            Expand
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
          : 'Click and type as a visitor would. Every interaction becomes a step. Expand for a bigger view and to scroll.'}
      </div>

      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="min-w-0 flex-1">{frameArea}</div>
        <div className="w-full lg:w-[380px]">{stepsRail}</div>
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
