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
  ZoomIn,
  ZoomOut,
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

interface ConsoleEvent {
  ts: number;
  level: 'info' | 'warn' | 'error';
  text: string;
  url?: string;
  line?: number;
}

interface NetworkEvent {
  ts: number;
  method: string;
  url: string;
  status: number;
  durMs: number;
  failed?: boolean;
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
  const [consoleLog, setConsoleLog] = useState<ConsoleEvent[]>([]);
  const [network, setNetwork] = useState<NetworkEvent[]>([]);
  const [tab, setTab] = useState<'steps' | 'console' | 'network'>('steps');
  // Zoom < 1 renders the page at a larger logical viewport so more of it fits —
  // exactly like a browser's zoom-out. Click mapping is unaffected because the
  // frame still reports its true pixel size.
  const [zoom, setZoom] = useState(1);

  const ws = useRef<WebSocket | null>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const frameBoxRef = useRef<HTMLDivElement>(null);

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
      } else if (msg.type === 'console') {
        setConsoleLog((prev) => [...prev.slice(-500), msg as ConsoleEvent]);
      } else if (msg.type === 'network') {
        setNetwork((prev) => [...prev.slice(-500), msg as NetworkEvent]);
      } else if (msg.type === 'capture') {
        // The backlog a mid-session connection missed.
        if (msg.console) setConsoleLog(msg.console);
        if (msg.network) setNetwork(msg.network);
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
      // Trackpads report deltaX directly; a mouse with Shift held scrolls
      // horizontally, which the browser signals as deltaX on most platforms but
      // we also fold in shift+deltaY so a plain wheel can reach wide content.
      const dx = e.deltaX || (e.shiftKey ? e.deltaY : 0);
      const dy = e.shiftKey && !e.deltaX ? 0 : e.deltaY;
      send({
        type: 'scroll',
        x: ((e.clientX - rect.left) / rect.width) * size.width,
        y: ((e.clientY - rect.top) / rect.height) * size.height,
        deltaX: dx,
        deltaY: dy,
      });
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, [send, size.width, size.height, frame, fullscreen]);

  // Ask the browser to render at the size of the area we have for it, so the
  // page fills the view crisply instead of being a small letterboxed picture,
  // and clicks land where they look. We measure the CONTAINER, not the image:
  // the image's size depends on the frame we are trying to size, which is
  // circular. A ResizeObserver re-sends on any layout change (fullscreen, rail,
  // window) with no fragile timeouts.
  const reportViewport = useCallback(() => {
    const el = frameBoxRef.current;
    if (!el) return;
    const w = Math.round(el.clientWidth / zoom);
    const h = Math.round(el.clientHeight / zoom);
    if (w > 0 && h > 0) send({ type: 'viewport', width: w, height: h });
  }, [send, zoom]);

  useEffect(() => {
    const el = frameBoxRef.current;
    if (!el || !connected) return;
    reportViewport();
    const ro = new ResizeObserver(() => reportViewport());
    ro.observe(el);
    return () => ro.disconnect();
  }, [reportViewport, connected, fullscreen, railOpen]);

  const ZOOM_MIN = 0.4;
  const ZOOM_MAX = 1;
  const clampZoom = (z: number) => Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, Math.round(z * 100) / 100));
  const zoomControls = (
    <div className="flex items-center gap-0.5 rounded-[var(--radius-6)] border border-border">
      <button
        onClick={() => setZoom((z) => clampZoom(z - 0.1))}
        disabled={zoom <= ZOOM_MIN}
        className="px-2 py-1 text-text-secondary transition-colors hover:text-text-primary disabled:opacity-40"
        title="Zoom out (show more)"
      >
        <ZoomOut size={14} />
      </button>
      <button
        onClick={() => setZoom(1)}
        className="min-w-[42px] px-1 text-center text-[length:var(--text-2xs)] tabular-nums text-text-muted transition-colors hover:text-text-primary"
        title="Reset zoom"
      >
        {Math.round(zoom * 100)}%
      </button>
      <button
        onClick={() => setZoom((z) => clampZoom(z + 0.1))}
        disabled={zoom >= ZOOM_MAX}
        className="px-2 py-1 text-text-secondary transition-colors hover:text-text-primary disabled:opacity-40"
        title="Zoom in"
      >
        <ZoomIn size={14} />
      </button>
    </div>
  );

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
          ? 'h-full w-full cursor-crosshair object-contain outline-none'
          : 'w-full cursor-crosshair rounded-[var(--radius)] border border-border outline-none focus:border-[var(--color-accent)]'
      }
    />
  ) : (
    <div className="flex h-[420px] w-full items-center justify-center rounded-[var(--radius)] border border-border bg-surface text-[length:var(--text-label)] text-text-subtle">
      {connected ? 'Waiting for the first frame…' : 'Connecting to the browser…'}
    </div>
  );

  const consoleErrors = consoleLog.filter((c) => c.level === 'error').length;

  const tabBtn = (key: typeof tab, label: string, badge?: number, danger?: boolean) => (
    <button
      onClick={() => setTab(key)}
      className={
        'flex items-center gap-1.5 border-b-2 px-2 pb-1.5 text-[length:var(--text-label)] transition-colors ' +
        (tab === key
          ? 'border-[var(--color-accent)] text-text-primary'
          : 'border-transparent text-text-muted hover:text-text-secondary')
      }
    >
      {label}
      {badge != null && badge > 0 && (
        <span
          className="rounded-full px-1.5 text-[length:var(--text-2xs)]"
          style={
            danger
              ? { backgroundColor: '#EF444422', color: '#F87171' }
              : { backgroundColor: 'var(--fill)', color: 'var(--text-muted)' }
          }
        >
          {badge}
        </span>
      )}
    </button>
  );

  const stepsRail = (
    <div className="flex min-h-0 w-full flex-1 flex-col gap-2">
      <div className="flex items-center justify-between border-b border-border">
        <div className="flex items-center gap-1">
          {tabBtn('steps', 'Steps', recorded)}
          {tabBtn('console', 'Console', consoleErrors, true)}
          {tabBtn('network', 'Network', network.length)}
        </div>
        {fullscreen && (
          <button
            onClick={() => setRailOpen(false)}
            className="mb-1 rounded p-0.5 text-text-subtle transition-colors hover:text-text-primary"
            aria-label="Hide panel"
          >
            <X size={14} />
          </button>
        )}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {tab === 'steps' &&
          (steps.length === 0 ? (
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
          ))}

        {tab === 'console' &&
          (consoleLog.length === 0 ? (
            <div className="p-3 text-[length:var(--text-caption)] text-text-subtle">
              No console output yet.
            </div>
          ) : (
            <div className="flex flex-col font-mono text-[length:var(--text-2xs)]">
              {consoleLog.map((c, i) => (
                <div
                  key={i}
                  className="border-b border-border/50 px-1 py-1"
                  style={{
                    color:
                      c.level === 'error'
                        ? '#F87171'
                        : c.level === 'warn'
                          ? '#FBBF24'
                          : 'var(--text-secondary)',
                  }}
                >
                  {c.text}
                  {c.url && (
                    <span className="ml-1 text-text-subtle">
                      ({c.url.split('/').pop()}
                      {c.line ? `:${c.line}` : ''})
                    </span>
                  )}
                </div>
              ))}
            </div>
          ))}

        {tab === 'network' &&
          (network.length === 0 ? (
            <div className="p-3 text-[length:var(--text-caption)] text-text-subtle">
              No requests yet.
            </div>
          ) : (
            <div className="flex flex-col font-mono text-[length:var(--text-2xs)]">
              {network.map((n, i) => (
                <div key={i} className="flex items-center gap-2 border-b border-border/50 px-1 py-1">
                  <span
                    className="w-9 shrink-0 text-right"
                    style={{
                      color: n.failed || n.status >= 400 ? '#F87171' : 'var(--text-muted)',
                    }}
                  >
                    {n.failed ? 'ERR' : n.status || '—'}
                  </span>
                  <span className="w-10 shrink-0 text-text-subtle">{n.method}</span>
                  <span className="min-w-0 flex-1 truncate text-text-secondary" title={n.url}>
                    {n.url.replace(/^https?:\/\//, '')}
                  </span>
                  <span className="shrink-0 text-text-subtle">{n.durMs}ms</span>
                </div>
              ))}
            </div>
          ))}
      </div>
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
            {zoomControls}
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
          <div
            ref={frameBoxRef}
            className="flex min-w-0 flex-1 items-center justify-center overflow-hidden bg-black"
          >
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
          {zoomControls}
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
        <div ref={frameBoxRef} className="min-w-0 flex-1">
          {frameArea}
        </div>
        <div className="flex max-h-[70vh] w-full flex-col lg:w-[380px]">{stepsRail}</div>
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
