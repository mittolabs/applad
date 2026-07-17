import { useEffect } from 'react';
import { navGroups, type NavGroup } from '@/lib/nav';
import { cn } from '@/lib/utils';

/*
 * Mobile navigation — ports shell.dart `_BottomNav` + `_GroupBottomSheet`.
 * Below 650px the icon rail and detail panel are replaced by a fixed bottom
 * nav bar (max 5 groups). Direct-nav groups (overview/platforms/settings) go
 * straight to their route; groups with children open a bottom sheet.
 */

const DIRECT_ROUTES: Record<string, string> = {
  overview: 'overview',
  platforms: 'platforms',
  settings: 'settings',
};

export function BottomNav({
  activeGroupId,
  onGroupTap,
}: {
  activeGroupId: string;
  onGroupTap: (group: NavGroup) => void;
}) {
  // Regular groups first, pinned (settings) last, capped at 5 (keeping the last
  // pinned item if we overflow) — matches _BottomNav's slicing.
  const regular = navGroups.filter((g) => !g.pinBottom);
  const pinned = navGroups.filter((g) => g.pinBottom);
  const all = [...regular, ...pinned];
  const visible = all.length > 5 ? [...all.slice(0, 4), all[all.length - 1]] : all;

  return (
    <nav className="flex h-14 shrink-0 items-stretch border-t border-border bg-surface-alt">
      {visible.map((g) => {
        const Icon = g.icon;
        const active = g.id === activeGroupId;
        return (
          <button
            key={g.id}
            onClick={() => onGroupTap(g)}
            className={cn(
              'flex flex-1 flex-col items-center justify-center gap-[3px]',
              active ? 'text-[var(--color-accent)]' : 'text-text-subtle',
            )}
          >
            <Icon size={20} />
            <span
              className={cn(
                'max-w-full truncate px-1 text-[10px]',
                active ? 'font-semibold' : 'font-normal',
              )}
            >
              {g.label}
            </span>
          </button>
        );
      })}
    </nav>
  );
}

/** Resolve a bottom-nav tap: navigate directly, or return the group whose
 * children should open in a sheet (null = handled by navigation). */
export function resolveGroupTap(
  group: NavGroup,
  go: (route: string) => void,
): NavGroup | null {
  if (DIRECT_ROUTES[group.id]) {
    go(DIRECT_ROUTES[group.id]);
    return null;
  }
  if (!group.children || group.children.length === 0) {
    go(group.route ?? group.id);
    return null;
  }
  return group;
}

export function GroupSheet({
  group,
  currentSegment,
  onNavigate,
  onClose,
}: {
  group: NavGroup;
  currentSegment: string;
  onNavigate: (route: string) => void;
  onClose: () => void;
}) {
  // Lock body scroll while the sheet is open.
  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  return (
    <div className="fixed inset-0 z-50 flex flex-col justify-end" role="dialog" aria-modal="true">
      <button
        className="absolute inset-0 bg-black/50"
        aria-label="Close menu"
        onClick={onClose}
      />
      <div className="relative max-h-[70vh] overflow-y-auto rounded-t-[16px] border-t border-border bg-surface pb-3 shadow-[0_-8px_32px_var(--shadow)]">
        <div className="sticky top-0 bg-surface pt-3">
          <div className="mx-auto mb-2 h-1 w-9 rounded-full bg-border" />
          <div className="px-5 pb-3 text-[length:var(--text-subhead)] font-semibold text-text-primary">
            {group.label}
          </div>
        </div>
        <div className="flex flex-col gap-0.5 px-2">
          {group.children!.map((c) => {
            const seg = c.route.split('?')[0].split('/')[0];
            const active = seg === currentSegment;
            const Icon = c.icon;
            return (
              <button
                key={c.route}
                onClick={() => onNavigate(c.route)}
                className={cn(
                  'flex h-11 items-center gap-3 rounded-[var(--radius-8)] px-3 text-[length:var(--text-control)] transition-colors',
                  active
                    ? 'bg-fill-active font-medium text-text-primary'
                    : 'font-normal text-text-secondary hover:bg-fill',
                )}
              >
                <Icon size={18} />
                {c.label}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
