import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Loader2, Pause, Play, Sparkles, X } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';

/*
 * The replay: a saved recording, scrubbable.
 *
 * A capture is the technical context gathered while recording — the console,
 * the network, the environment, and a frame-by-frame video — all on one clock.
 * Here they move together: scrub the video and the console and network cursors
 * follow; click a console error and the video seeks to it. This is the jam.dev
 * loop, in-product.
 *
 * Frames are served from the storage volume as images; the token and project
 * ride in the query the same way the live stream's do, so an <img> can load them
 * without headers.
 */

interface FrameMark {
  seq: number;
  ms: number;
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
interface Step {
  kind: string;
  description: string;
  ts?: number;
}
interface Capture {
  $id: string;
  target: string;
  startedAt: number;
  durationMs: number;
  frames: FrameMark[];
  console: ConsoleEvent[];
  network: NetworkEvent[];
  steps: Step[];
  aiSummary?: string;
  aiAvailable?: boolean;
}

export function CaptureReplay({ captureId, onClose }: { captureId: string; onClose: () => void }) {
  const [cap, setCap] = useState<Capture | null>(null);
  const [error, setError] = useState('');
  const [ms, setMs] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [tab, setTab] = useState<'console' | 'network'>('console');
  const [aiSummary, setAiSummary] = useState('');
  const [explaining, setExplaining] = useState(false);
  const raf = useRef<number | null>(null);
  const lastTick = useRef<number>(0);

  useEffect(() => {
    api
      .get(`/studio/captures/${captureId}`)
      .then((r) => {
        const c = r.data as Capture;
        setCap(c);
        if (c.aiSummary) setAiSummary(c.aiSummary);
      })
      .catch(() => setError('This recording has no replay. It was saved before capture existed.'));
  }, [captureId]);

  const explain = async () => {
    setExplaining(true);
    try {
      const s = (await api.post(`/studio/captures/${captureId}/explain`)).data as { summary: string };
      setAiSummary(s.summary);
    } catch (e) {
      setError(friendlyError(e));
    } finally {
      setExplaining(false);
    }
  };

  const duration = cap?.durationMs ?? 0;
  const rel = useCallback((ts: number) => (cap ? ts - cap.startedAt : 0), [cap]);

  // The frame to show at the current time: the last one whose mark is at or
  // before it. Frames are change-driven, so between changes the picture holds.
  const seq = useMemo(() => {
    if (!cap || cap.frames.length === 0) return -1;
    let s = cap.frames[0].seq;
    for (const f of cap.frames) {
      if (f.ms <= ms) s = f.seq;
      else break;
    }
    return s;
  }, [cap, ms]);

  const token = localStorage.getItem('applad_console_token') ?? '';
  const project = (api.defaults.headers.common['X-Applad-Project'] as string) ?? '';
  const frameSrc =
    seq >= 0
      ? `/v1/studio/captures/${captureId}/frames/${seq}?token=${encodeURIComponent(token)}&project=${encodeURIComponent(project)}`
      : '';

  // Play advances the clock in real time and stops at the end.
  useEffect(() => {
    if (!playing) {
      if (raf.current) cancelAnimationFrame(raf.current);
      return;
    }
    lastTick.current = performance.now();
    const step = (now: number) => {
      const dt = now - lastTick.current;
      lastTick.current = now;
      setMs((m) => {
        const next = m + dt;
        if (next >= duration) {
          setPlaying(false);
          return duration;
        }
        return next;
      });
      raf.current = requestAnimationFrame(step);
    };
    raf.current = requestAnimationFrame(step);
    return () => {
      if (raf.current) cancelAnimationFrame(raf.current);
    };
  }, [playing, duration]);

  const seek = (to: number) => {
    setPlaying(false);
    setMs(Math.max(0, Math.min(duration, to)));
  };
  const toggle = () => {
    if (ms >= duration) setMs(0);
    setPlaying((p) => !p);
  };

  const pct = (t: number) => (duration > 0 ? (t / duration) * 100 : 0);

  if (error) {
    return (
      <Overlay onClose={onClose} target="">
        <div className="flex flex-1 items-center justify-center text-[length:var(--text-body)] text-text-muted">
          {error}
        </div>
      </Overlay>
    );
  }
  if (!cap) {
    return (
      <Overlay onClose={onClose} target="">
        <div className="flex flex-1 items-center justify-center text-text-muted">Loading…</div>
      </Overlay>
    );
  }

  const consoleErrors = cap.console.filter((c) => c.level === 'error').length;
  const activeConsole = cap.console.reduce<number>((best, c, i) => (rel(c.ts) <= ms ? i : best), -1);
  const activeNet = cap.network.reduce<number>((best, n, i) => (rel(n.ts) <= ms ? i : best), -1);

  return (
    <Overlay onClose={onClose} target={cap.target}>
      <div className="flex min-h-0 flex-1">
        {/* Video */}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-black">
            {frameSrc ? (
              <img src={frameSrc} alt="Replay frame" className="h-full w-full object-contain" />
            ) : (
              <span className="text-text-subtle">No video frames were captured.</span>
            )}
          </div>

          {/* Transport + timeline */}
          <div className="flex flex-col gap-2 border-t border-border p-3">
            <div className="flex items-center gap-3">
              <button
                onClick={toggle}
                className="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-accent)] text-white"
                aria-label={playing ? 'Pause' : 'Play'}
              >
                {playing ? <Pause size={15} /> : <Play size={15} />}
              </button>
              <span className="w-24 shrink-0 font-mono text-[length:var(--text-2xs)] text-text-muted tabular-nums">
                {(ms / 1000).toFixed(1)}s / {(duration / 1000).toFixed(1)}s
              </span>
              <input
                type="range"
                min={0}
                max={duration}
                value={ms}
                onChange={(e) => seek(Number(e.target.value))}
                className="flex-1 accent-[var(--color-accent)]"
              />
            </div>

            {/* Lanes: steps, console errors, network. Click a marker to seek. */}
            <div className="flex flex-col gap-1">
              {(
                [
                  ['Steps', cap.steps.filter((s) => s.ts).map((s) => ({ t: s.ts!, danger: false, label: s.description }))],
                  ['Console', cap.console.map((c) => ({ t: rel(c.ts), danger: c.level === 'error', label: c.text }))],
                  ['Network', cap.network.map((n) => ({ t: rel(n.ts), danger: n.failed || n.status >= 400, label: `${n.status} ${n.url}` }))],
                ] as [string, { t: number; danger: boolean; label: string }[]][]
              ).map(([lane, marks]) => (
                <div key={lane} className="flex items-center gap-2">
                  <span className="w-14 shrink-0 text-[length:var(--text-2xs)] text-text-subtle">{lane}</span>
                  <div className="relative h-3 flex-1 rounded bg-fill">
                    {marks.map((m, i) => (
                      <button
                        key={i}
                        onClick={() => seek(m.t)}
                        title={m.label}
                        className="absolute top-0 h-3 w-[3px] -translate-x-1/2 rounded-full"
                        style={{
                          left: `${pct(m.t)}%`,
                          backgroundColor: m.danger ? '#F87171' : 'var(--color-accent)',
                        }}
                      />
                    ))}
                    <div
                      className="absolute top-[-2px] h-[16px] w-px bg-text-primary"
                      style={{ left: `${pct(ms)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Console / network panels */}
        <div className="flex w-[360px] shrink-0 flex-col border-l border-border">
          {(cap.aiAvailable || aiSummary) && (
            <div className="border-b border-border p-2">
              {aiSummary ? (
                <div className="rounded-[var(--radius)] bg-fill p-2.5 text-[length:var(--text-caption)] leading-relaxed text-text-secondary">
                  <div className="mb-1 flex items-center gap-1.5 text-text-primary">
                    <Sparkles size={13} className="text-[var(--color-accent)]" />
                    <span className="font-medium">Explanation</span>
                  </div>
                  <div className="whitespace-pre-wrap">{aiSummary}</div>
                </div>
              ) : (
                <Button variant="outline" className="w-full" onClick={explain} disabled={explaining}>
                  {explaining ? <Loader2 size={14} className="animate-spin" /> : <Sparkles size={14} />}
                  {explaining ? 'Diagnosing…' : 'Explain this capture'}
                </Button>
              )}
            </div>
          )}
          <div className="flex items-center gap-1 border-b border-border px-2">
            <PanelTab label="Console" badge={consoleErrors} danger active={tab === 'console'} onClick={() => setTab('console')} />
            <PanelTab label="Network" badge={cap.network.length} active={tab === 'network'} onClick={() => setTab('network')} />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto font-mono text-[length:var(--text-2xs)]">
            {tab === 'console' &&
              cap.console.map((c, i) => (
                <button
                  key={i}
                  onClick={() => seek(rel(c.ts))}
                  className={'block w-full border-b border-border/50 px-2 py-1 text-left ' + (i === activeConsole ? 'bg-fill-active' : '')}
                  style={{
                    color: c.level === 'error' ? '#F87171' : c.level === 'warn' ? '#FBBF24' : 'var(--text-secondary)',
                  }}
                >
                  {c.text}
                </button>
              ))}
            {tab === 'network' &&
              cap.network.map((n, i) => (
                <button
                  key={i}
                  onClick={() => seek(rel(n.ts))}
                  className={'flex w-full items-center gap-2 border-b border-border/50 px-2 py-1 text-left ' + (i === activeNet ? 'bg-fill-active' : '')}
                >
                  <span className="w-8 shrink-0" style={{ color: n.failed || n.status >= 400 ? '#F87171' : 'var(--text-muted)' }}>
                    {n.failed ? 'ERR' : n.status || '—'}
                  </span>
                  <span className="w-9 shrink-0 text-text-subtle">{n.method}</span>
                  <span className="min-w-0 flex-1 truncate text-text-secondary" title={n.url}>
                    {n.url.replace(/^https?:\/\//, '')}
                  </span>
                  <span className="shrink-0 text-text-subtle">{n.durMs}ms</span>
                </button>
              ))}
          </div>
        </div>
      </div>
    </Overlay>
  );
}

function Overlay({ target, onClose, children }: { target: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-2">
        <span className="truncate font-mono text-[length:var(--text-caption)] text-text-secondary">
          {target || 'Replay'}
        </span>
        <Button variant="outline" onClick={onClose}>
          <X size={14} />
          Close
        </Button>
      </div>
      {children}
    </div>
  );
}

function PanelTab({
  label,
  badge,
  danger,
  active,
  onClick,
}: {
  label: string;
  badge?: number;
  danger?: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={
        'flex items-center gap-1.5 border-b-2 px-2 pb-1.5 pt-2 text-[length:var(--text-label)] transition-colors ' +
        (active
          ? 'border-[var(--color-accent)] text-text-primary'
          : 'border-transparent text-text-muted hover:text-text-secondary')
      }
    >
      {label}
      {badge != null && badge > 0 && (
        <span
          className="rounded-full px-1.5 text-[length:var(--text-2xs)]"
          style={danger ? { backgroundColor: '#EF444422', color: '#F87171' } : { backgroundColor: 'var(--fill)', color: 'var(--text-muted)' }}
        >
          {badge}
        </span>
      )}
    </button>
  );
}
