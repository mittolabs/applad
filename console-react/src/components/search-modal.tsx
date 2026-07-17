import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Building2,
  CornerDownLeft,
  FolderPlus,
  Search,
  User,
  type LucideIcon,
} from 'lucide-react';
import { Dialog, DialogContent } from './ui/dialog';
import { api } from '@/api/client';
import { navGroups, navShortcuts } from '@/lib/nav';
import { cn } from '@/lib/utils';

/* Ports search_modal.dart — ⌘K command palette with icons, keyboard chord
 * shortcuts (shown here, handled by the shell), navigation + working commands
 * (create project/org, account), and debounced server search. */

interface Item {
  id: string;
  label: string;
  category: string;
  icon: LucideIcon;
  shortcut?: string;
  run: () => void;
}

export function SearchModal({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const navigate = useNavigate();
  const { projectId } = useParams<{ projectId: string }>();
  const [query, setQuery] = useState('');
  const [debounced, setDebounced] = useState('');
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const close = () => onOpenChange(false);

  useEffect(() => {
    if (open) {
      setQuery('');
      setDebounced('');
      setActive(0);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  // Navigation items (icons + chord shortcuts) from the nav model.
  const navItems: Item[] = useMemo(() => {
    if (!projectId) return [];
    const items: Item[] = [];
    const push = (label: string, route: string, icon: LucideIcon) => {
      const seg = route.split('?')[0];
      items.push({
        id: `nav-${seg}`,
        label,
        category: 'Navigate',
        icon,
        shortcut: navShortcuts[seg] ? `g ${navShortcuts[seg]}` : undefined,
        run: () => {
          navigate(`/project/${projectId}/${seg}`);
          close();
        },
      });
    };
    for (const g of navGroups) {
      if (g.route) push(g.label, g.route, g.icon);
      for (const c of g.children ?? []) push(c.label, c.route, c.icon);
    }
    return items;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  // Working commands.
  const commandItems: Item[] = useMemo(
    () => [
      {
        id: 'cmd-create-project',
        label: 'Create project',
        category: 'Commands',
        icon: FolderPlus,
        run: () => {
          navigate('/projects?create=project');
          close();
        },
      },
      {
        id: 'cmd-create-org',
        label: 'Create organization',
        category: 'Commands',
        icon: Building2,
        run: () => {
          navigate('/projects?create=org');
          close();
        },
      },
      {
        id: 'cmd-account',
        label: 'Account settings',
        category: 'Commands',
        icon: User,
        run: () => {
          navigate('/account');
          close();
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const { data: serverResults = [] } = useQuery({
    queryKey: ['cmdk-search', projectId, debounced],
    enabled: !!projectId && debounced.length >= 2,
    queryFn: async () => {
      const res = await api.get(`/projects/${projectId}/search`, {
        params: { q: debounced, limit: 20 },
      });
      const raw = (res.data as { results?: Record<string, unknown>[] }).results ?? [];
      return raw.map((r, i) => ({
        id: `srv-${i}`,
        label: String(r['title'] ?? r['name'] ?? ''),
        category: String(r['category'] ?? 'Results'),
        icon: Search,
        run: () => {
          navigate(String(r['path'] ?? '#'));
          close();
        },
      })) as Item[];
    },
  });

  const q = query.trim().toLowerCase();
  const localMatches = [...navItems, ...commandItems].filter((i) =>
    i.label.toLowerCase().includes(q),
  );
  const results = [...localMatches, ...serverResults];

  const groups = useMemo(() => {
    const order: string[] = [];
    const map: Record<string, Item[]> = {};
    for (const r of results) {
      if (!map[r.category]) {
        map[r.category] = [];
        order.push(r.category);
      }
      map[r.category].push(r);
    }
    return order.map((c) => ({ category: c, items: map[c] }));
  }, [results]);

  const flat = groups.flatMap((g) => g.items);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActive((a) => Math.min(a + 1, flat.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      flat[active]?.run();
    }
  };

  let runningIndex = -1;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={640} showClose={false} className="max-h-[72vh]">
        <div className="flex items-center gap-2 border-b border-border px-4">
          <Search size={16} className="text-text-subtle" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setActive(0);
            }}
            onKeyDown={onKeyDown}
            placeholder="Search or jump to…"
            className="h-12 flex-1 bg-transparent text-[length:var(--text-control)] text-text-primary placeholder:text-text-subtle focus:outline-none"
          />
          <kbd className="rounded border border-border px-1.5 py-0.5 text-[length:var(--text-2xs)] text-text-subtle">
            Esc
          </kbd>
        </div>

        <div className="max-h-[52vh] overflow-y-auto p-2">
          {flat.length === 0 ? (
            <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
              No results
            </div>
          ) : (
            groups.map((g) => (
              <div key={g.category} className="mb-2">
                <div className="px-2 py-1 text-[length:var(--text-2xs)] uppercase tracking-wide text-text-subtle">
                  {g.category}
                </div>
                {g.items.map((item) => {
                  runningIndex += 1;
                  const idx = runningIndex;
                  const Icon = item.icon;
                  return (
                    <button
                      key={item.id}
                      onMouseEnter={() => setActive(idx)}
                      onClick={() => item.run()}
                      className={cn(
                        'flex w-full items-center gap-2.5 rounded-[var(--radius-6)] px-2 py-2 text-left text-[length:var(--text-body)] transition-colors',
                        idx === active
                          ? 'bg-fill-active text-text-primary'
                          : 'text-text-secondary',
                      )}
                    >
                      <Icon
                        size={15}
                        className={idx === active ? 'text-text-primary' : 'text-text-muted'}
                      />
                      <span className="flex-1">{item.label}</span>
                      {item.shortcut && (
                        <span className="flex gap-1">
                          {item.shortcut.split(' ').map((k, i) => (
                            <kbd
                              key={i}
                              className="rounded border border-border px-1.5 py-0.5 text-[length:var(--text-2xs)] text-text-subtle"
                            >
                              {k}
                            </kbd>
                          ))}
                        </span>
                      )}
                      {idx === active && !item.shortcut && (
                        <CornerDownLeft size={13} className="text-text-subtle" />
                      )}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>

        {/* Footer legend */}
        <div className="flex items-center gap-4 border-t border-border px-4 py-2.5 text-[length:var(--text-caption)] text-text-subtle">
          <span className="flex items-center gap-1.5">
            <Legend>↑</Legend>
            <Legend>↓</Legend>
            navigate
          </span>
          <span className="flex items-center gap-1.5">
            <Legend>↵</Legend>
            select
          </span>
          <span className="flex items-center gap-1.5">
            <Legend>esc</Legend>
            close
          </span>
          <span className="ml-auto flex items-center gap-1.5">
            <Legend>g</Legend>
            then a key to jump
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function Legend({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="rounded border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-2xs)] text-text-muted">
      {children}
    </kbd>
  );
}
