import { Suspense, useEffect, useState } from 'react';
import { Outlet, useLocation, useNavigate, useParams } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { navGroups, routeToGroup, shortcutToRoute, type NavGroup } from '@/lib/nav';
import { api, setProject } from '@/api/client';
import { useProjectStore } from '@/stores/project';
import { ConsoleFooter } from '@/components/console-footer';
import { Notices } from '@/components/notices';
import { useIsMobile } from '@/hooks/use-media-query';
import { TopNav } from './TopNav';
import { BottomNav, GroupSheet, resolveGroupTap } from './MobileNav';
import { cn } from '@/lib/utils';

export function Shell() {
  const { projectId } = useParams<{ projectId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const syncProject = useProjectStore((s) => s.syncProject);

  // Set the X-Applad-Project header synchronously during render, before any
  // child query fires (mirrors AppShell._syncProject).
  if (projectId) setProject(projectId);
  useEffect(() => {
    syncProject(projectId ?? null);
  }, [projectId, syncProject]);

  const isMobile = useIsMobile();
  const [sheetGroup, setSheetGroup] = useState<NavGroup | null>(null);

  // Keyboard chords: press "g" then a key to jump (e.g. g d → databases).
  useEffect(() => {
    if (!projectId) return;
    let pendingG = false;
    let timer: number | undefined;
    const onKey = (e: KeyboardEvent) => {
      const el = document.activeElement as HTMLElement | null;
      const typing =
        !!el &&
        (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;
      if (pendingG) {
        pendingG = false;
        window.clearTimeout(timer);
        const route = shortcutToRoute[e.key.toLowerCase()];
        if (route) {
          e.preventDefault();
          navigate(`/project/${projectId}/${route}`);
        }
        return;
      }
      if (e.key.toLowerCase() === 'g') {
        pendingG = true;
        timer = window.setTimeout(() => (pendingG = false), 900);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('keydown', onKey);
      window.clearTimeout(timer);
    };
  }, [projectId, navigate]);

  // Active segment → group.
  const base = `/project/${projectId}/`;
  const rest = location.pathname.startsWith(base)
    ? location.pathname.slice(base.length)
    : '';
  const segment = rest.split('/')[0] || 'overview';
  const activeGroupId = routeToGroup[segment] ?? 'overview';
  const activeGroup = navGroups.find((g) => g.id === activeGroupId);
  const hasPanel = !!activeGroup?.children;

  const go = (route: string) => {
    setSheetGroup(null);
    navigate(`/project/${projectId}/${route}`);
  };

  const content = (
    <main className="min-h-0 flex-1 overflow-y-auto">
      {/* Banners are data from /v1/entitlements; nothing renders by default. */}
      <Notices region="app.top" className="px-6 pt-4" />
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-text-muted" />
          </div>
        }
      >
        <Outlet />
      </Suspense>
    </main>
  );

  // ── Mobile layout: top nav + content + bottom nav (no rail / footer) ──────
  if (isMobile) {
    return (
      <div className="flex h-screen flex-col overflow-hidden bg-background">
        <TopNav projectId={projectId} />
        {content}
        <BottomNav
          activeGroupId={activeGroupId}
          onGroupTap={(g) => {
            const sheet = resolveGroupTap(g, go);
            setSheetGroup(sheet);
          }}
        />
        {sheetGroup && (
          <GroupSheet
            group={sheetGroup}
            currentSegment={segment}
            onNavigate={go}
            onClose={() => setSheetGroup(null)}
          />
        )}
      </div>
    );
  }

  // ── Desktop layout: icon rail + optional detail panel + content ───────────
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background">
      <TopNav projectId={projectId} />
      <div className="flex min-h-0 flex-1">
        <IconRail
          activeGroupId={activeGroupId}
          onSelect={go}
          projectId={projectId}
          onGetStarted={() => go('get-started')}
        />
        {hasPanel && (
          <DetailPanel group={activeGroup!} currentSegment={segment} onSelect={go} />
        )}
        <div className="flex min-w-0 flex-1 flex-col">
          {content}
          <ConsoleFooter />
        </div>
      </div>
    </div>
  );
}

const RAIL_W = 68;
const PANEL_W = 220;

function IconRail({
  activeGroupId,
  onSelect,
  projectId,
  onGetStarted,
}: {
  activeGroupId: string;
  onSelect: (route: string) => void;
  projectId?: string;
  onGetStarted: () => void;
}) {
  const top = navGroups.filter((g) => !g.pinBottom);
  const bottom = navGroups.filter((g) => g.pinBottom);
  const item = (g: NavGroup) => (
    <RailItem
      key={g.id}
      group={g}
      active={g.id === activeGroupId}
      onClick={() => onSelect(g.route ?? g.children![0].route.split('?')[0])}
    />
  );
  return (
    <nav
      style={{ width: RAIL_W }}
      className="flex shrink-0 flex-col items-center border-r border-border bg-surface-alt py-3"
    >
      <GetStartedRing projectId={projectId} onClick={onGetStarted} />
      <div className="my-2 h-px w-8 bg-border" />
      {top.map(item)}
      <div className="flex-1" />
      {bottom.map(item)}
    </nav>
  );
}

/* Icon-only rail item — ports shell.dart _IconRail item: 68x44 tap area, a 3px
 * left accent bar when active (inset 6px top/bottom), a 38x38 fill box, icon 18. */
function RailItem({
  group,
  active,
  onClick,
}: {
  group: NavGroup;
  active: boolean;
  onClick: () => void;
}) {
  const Icon = group.icon;
  return (
    <button
      onClick={onClick}
      className="relative flex h-11 w-full items-center justify-center"
      title={group.label}
      aria-label={group.label}
    >
      {active && (
        <span className="absolute bottom-1.5 left-0 top-1.5 w-[3px] rounded-r-[3px] bg-[var(--color-accent)]" />
      )}
      <span
        className={cn(
          'flex h-[38px] w-[38px] items-center justify-center rounded-[var(--radius-10)] transition-colors',
          active
            ? 'bg-fill-active text-text-primary'
            : 'text-text-muted hover:bg-fill hover:text-text-secondary',
        )}
      >
        <Icon size={18} />
      </span>
    </button>
  );
}

/* Get-started progress ring pinned to the rail top (ports shell.dart
 * _GetStartedRailItem). Counts a few resource types; hides at 100%. */
function GetStartedRing({ projectId, onClick }: { projectId?: string; onClick: () => void }) {
  const { data: pct = 0 } = useQuery({
    queryKey: ['get-started-progress', projectId],
    enabled: !!projectId,
    staleTime: 60_000,
    queryFn: async () => {
      const paths = ['/databases', '/storage/buckets', '/functions', '/deploy/targets'];
      const results = await Promise.all(
        paths.map((p) =>
          api
            .get(p, { params: { limit: 1 } })
            .then((r) => {
              const d = r.data as Record<string, unknown>;
              const arr = Object.values(d).find(Array.isArray) as unknown[] | undefined;
              const total = typeof d.total === 'number' ? d.total : (arr?.length ?? 0);
              return total > 0 ? 1 : 0;
            })
            .catch(() => 0),
        ),
      );
      const done = results.reduce((a, b) => a + b, 0);
      return Math.round((done / paths.length) * 100);
    },
  });

  if (pct >= 100) return null;
  const r = 15;
  const c = 2 * Math.PI * r;
  return (
    <button
      onClick={onClick}
      className="relative flex h-10 w-10 items-center justify-center"
      title="Get started"
      aria-label="Get started"
    >
      <svg width="36" height="36" viewBox="0 0 36 36" className="-rotate-90">
        <circle cx="18" cy="18" r={r} fill="none" stroke="var(--fill-active)" strokeWidth="3" />
        <circle
          cx="18"
          cy="18"
          r={r}
          fill="none"
          stroke="var(--color-accent)"
          strokeWidth="3"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - pct / 100)}
        />
      </svg>
      <span className="absolute text-[length:var(--text-2xs)] font-semibold text-text-secondary">
        {pct}
      </span>
    </button>
  );
}

function DetailPanel({
  group,
  currentSegment,
  onSelect,
}: {
  group: NavGroup;
  currentSegment: string;
  onSelect: (route: string) => void;
}) {
  return (
    <div
      style={{ width: PANEL_W }}
      className="flex shrink-0 flex-col gap-0.5 border-r border-border bg-surface p-2"
    >
      <div className="px-2 py-2 text-[length:var(--text-subhead)] font-semibold text-text-primary">
        {group.label}
      </div>
      {group.children!.map((c) => {
        const seg = c.route.split('?')[0].split('/')[0];
        const active = seg === currentSegment;
        const Icon = c.icon;
        return (
          <button
            key={c.route}
            onClick={() => onSelect(c.route)}
            className={cn(
              'flex h-9 items-center gap-2.5 rounded-[var(--radius-6)] px-2 text-[length:var(--text-control)] transition-colors',
              active
                ? 'bg-fill-active font-medium text-text-primary'
                : 'font-normal text-text-muted hover:bg-fill hover:text-text-secondary',
            )}
          >
            <Icon size={15} />
            {c.label}
          </button>
        );
      })}
    </div>
  );
}
