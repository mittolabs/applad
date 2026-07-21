import { useEffect, useState } from 'react';
import { CheckCircle2, AlertTriangle, XCircle } from 'lucide-react';

// Local dev mirrors production with `.localhost` appended, so derive the host
// rather than hardcoding one and breaking the other.
const SITE_URL =
  typeof window !== 'undefined' && window.location.hostname.endsWith('.localhost')
    ? 'http://applad.io.localhost'
    : 'https://applad.io';

type Level = 'operational' | 'degraded' | 'down';

interface ComponentStatus {
  key: string;
  name: string;
  status: Level;
  latencyMs: number;
  uptime24h: number;
  uptime7d: number;
  uptime90d: number;
  history: Level[];
}

interface Incident {
  id: string;
  component: string;
  title: string;
  status: 'investigating' | 'resolved';
  severity: 'minor' | 'major' | 'critical';
  startedAt: string;
  resolvedAt?: string;
}

interface Snapshot {
  overall: Level;
  updatedAt: string;
  components: ComponentStatus[];
  incidents: Incident[];
}

const LEVEL: Record<Level, { label: string; color: string; Icon: typeof CheckCircle2 }> = {
  operational: { label: 'Operational', color: 'var(--color-ok)', Icon: CheckCircle2 },
  degraded: { label: 'Degraded', color: 'var(--color-degraded)', Icon: AlertTriangle },
  down: { label: 'Down', color: 'var(--color-down)', Icon: XCircle },
};

const OVERALL: Record<Level, string> = {
  operational: 'All systems operational',
  degraded: 'Some systems degraded',
  down: 'Major outage in progress',
};

function fmt(ts: string): string {
  try {
    return new Date(ts).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return ts;
  }
}

export function App() {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [error, setError] = useState(false);

  const load = () => {
    fetch('/v1/status', { headers: { Accept: 'application/json' } })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d: Snapshot) => {
        setSnap(d);
        setError(false);
      })
      .catch(() => setError(true));
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 30_000);
    return () => clearInterval(t);
  }, []);

  const overall = snap?.overall ?? 'operational';
  const O = LEVEL[overall];

  return (
    <div className="mx-auto min-h-screen max-w-3xl px-6 py-14">
      <header className="mb-10 flex items-center justify-between">
        <a href={SITE_URL} className="flex items-center gap-2.5">
          <img src="/favicon.png" alt="" className="h-7 w-7 rounded-[7px] object-cover" />
          <span className="text-[17px] font-bold tracking-tight">applad</span>
          <span className="ml-1 text-sm text-muted">Status</span>
        </a>
        <a href={SITE_URL} className="text-sm text-muted transition-colors hover:text-text">
          applad.io →
        </a>
      </header>

      {/* Overall banner — solid, confident status bar */}
      {error ? (
        <div className="flex items-center gap-3 rounded-xl border border-border bg-surface px-5 py-4">
          <AlertTriangle size={19} className="text-muted" />
          <span className="text-[15px] font-medium">Status temporarily unavailable</span>
          <span className="ml-auto text-[13px] text-muted">retrying…</span>
        </div>
      ) : (
        <div
          className="flex items-center gap-3 rounded-xl px-5 py-4"
          style={{ background: O.color, color: '#0b0b0f', boxShadow: 'inset 0 1px 0 rgba(255,255,255,0.22)' }}
        >
          <O.Icon size={20} strokeWidth={2.5} />
          <span className="text-[16px] font-semibold tracking-tight">{OVERALL[overall]}</span>
          <span className="ml-auto text-[13px] font-medium" style={{ color: 'rgba(11,11,15,0.6)' }}>
            {snap ? `Updated ${fmt(snap.updatedAt)}` : 'Checking…'}
          </span>
        </div>
      )}

      {/* Components */}
      <div className="panel mt-8 overflow-hidden">
        {(snap?.components ?? []).map((c, i) => (
          <div key={c.key} className={`px-5 py-4 ${i > 0 ? 'border-t border-white/[0.06]' : ''}`}>
            <div className="flex items-center justify-between">
              <span className="text-[15px] font-medium">{c.name}</span>
              <span className="flex items-center gap-1.5 text-[13px]" style={{ color: LEVEL[c.status].color }}>
                <span className="h-2 w-2 rounded-full" style={{ background: LEVEL[c.status].color }} />
                {LEVEL[c.status].label}
              </span>
            </div>
            <UptimeBar history={c.history} />
            <div className="mt-1.5 flex justify-between text-[11px] text-muted">
              <span>90 days ago</span>
              <span>{c.uptime90d.toFixed(2)}% uptime</span>
              <span>today</span>
            </div>
          </div>
        ))}
        {!snap && !error && <div className="px-5 py-10 text-center text-sm text-muted">Loading components…</div>}
      </div>

      {/* Incidents */}
      <h2 className="mb-4 mt-12 text-sm font-semibold uppercase tracking-[0.14em] text-muted/70">
        Recent incidents
      </h2>
      {snap && snap.incidents.length === 0 ? (
        <div className="rounded-xl border border-border px-5 py-6 text-sm text-muted">
          No incidents reported in the last 90 days.
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {(snap?.incidents ?? []).map((inc) => {
            const resolved = inc.status === 'resolved';
            const color = resolved ? 'var(--color-ok)' : LEVEL.down.color;
            return (
              <div key={inc.id} className="rounded-xl border border-border p-5">
                <div className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full" style={{ background: color }} />
                  <span className="font-medium">{inc.title}</span>
                  <span className="ml-auto text-[12px] uppercase tracking-wide" style={{ color }}>
                    {resolved ? 'Resolved' : 'Investigating'}
                  </span>
                </div>
                <div className="mt-2 text-[13px] text-muted">
                  Started {fmt(inc.startedAt)}
                  {inc.resolvedAt ? ` · Resolved ${fmt(inc.resolvedAt)}` : ''}
                </div>
              </div>
            );
          })}
        </div>
      )}

      <footer className="mt-16 flex justify-center border-t border-border pt-6">
        <a
          href={SITE_URL}
          className="inline-flex items-center gap-2 text-[13px] text-muted transition-colors hover:text-text"
        >
          <img src="/favicon.png" alt="" className="h-4 w-4 rounded-[4px] object-cover opacity-90" />
          Powered by <span className="font-medium text-text">Applad</span>
        </a>
      </footer>
    </div>
  );
}

/** 90 daily cells, right-aligned to today. Missing days render as "no data". */
function UptimeBar({ history }: { history: Level[] }) {
  const DAYS = 90;
  const recent = history.slice(-DAYS);
  const pad = DAYS - recent.length;
  const cells: (Level | null)[] = [...Array(pad).fill(null), ...recent];
  return (
    <div className="mt-3 flex gap-[2px]">
      {cells.map((lvl, i) => (
        <div
          key={i}
          className="h-8 flex-1 rounded-[2px]"
          title={lvl ?? 'no data'}
          style={{ background: lvl ? LEVEL[lvl].color : 'rgba(255,255,255,0.06)' }}
        />
      ))}
    </div>
  );
}
