import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  GitBranch,
  LayoutTemplate,
  Upload,
  type LucideIcon,
} from 'lucide-react';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { AppBadge } from './app-badge';
import { api } from '@/api/client';
import { cn } from '@/lib/utils';

/* Ports deploy_create_entry.dart — 3-option entry dialog (template / repo /
 * upload) used by Sites, Containers, Mobile, Desktop. Returns a result via
 * onResult; caller performs the actual create. */

export type CreateEntryChoice = 'template' | 'repository' | 'upload';
export interface CreateEntryResult {
  choice: CreateEntryChoice;
  templateConfig?: Record<string, unknown>;
  repoConfig?: Record<string, unknown>;
}

type View = 'entry' | 'templates' | 'repo' | 'upload';

export function DeployCreateEntry({
  open,
  onOpenChange,
  category,
  title,
  subtitle,
  onResult,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  category: string;
  title: string;
  subtitle: string;
  onResult: (result: CreateEntryResult) => void;
}) {
  const [view, setView] = useState<View>('entry');
  useEffect(() => {
    if (open) setView('entry');
  }, [open]);

  const finish = (result: CreateEntryResult) => {
    onResult(result);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={520}>
        <DialogHeader>
          <div className="flex items-center gap-2">
            {view !== 'entry' && (
              <button
                onClick={() => setView('entry')}
                className="rounded-[var(--radius-6)] p-1 text-text-muted hover:bg-fill hover:text-text-primary"
                aria-label="Back"
              >
                <ArrowLeft size={16} />
              </button>
            )}
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription>{subtitle}</DialogDescription>
        </DialogHeader>
        <DialogBody>
          {view === 'entry' && (
            <div className="flex flex-col gap-2">
              <EntryOption
                icon={LayoutTemplate}
                title="Clone a template"
                subtitle="Start from a ready-made template."
                onClick={() => setView('templates')}
              />
              <EntryOption
                icon={GitBranch}
                title="Connect a repository"
                subtitle="Deploy from a Git repository."
                onClick={() => setView('repo')}
              />
              <EntryOption
                icon={Upload}
                title="Manual upload"
                subtitle="Upload your build directly."
                onClick={() => finish({ choice: 'upload' })}
              />
            </div>
          )}
          {view === 'templates' && (
            <TemplatesView
              category={category}
              onPick={(t) => finish({ choice: 'template', templateConfig: t })}
            />
          )}
          {view === 'repo' && (
            <RepoView onPick={(r) => finish({ choice: 'repository', repoConfig: r })} />
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function EntryOption({
  icon: Icon,
  title,
  subtitle,
  onClick,
}: {
  icon: LucideIcon;
  title: string;
  subtitle: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="flex items-center gap-3 rounded-[var(--radius-10)] border border-border bg-surface p-3 text-left transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_50%,var(--border))] hover:bg-fill"
    >
      <div className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)] bg-fill text-[var(--color-accent)]">
        <Icon size={18} />
      </div>
      <div>
        <div className="text-[length:var(--text-control)] font-medium text-text-primary">{title}</div>
        <div className="text-[length:var(--text-caption)] text-text-muted">{subtitle}</div>
      </div>
    </button>
  );
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'shrink-0 rounded-[var(--radius-6)] border px-2.5 py-1 text-[length:var(--text-label)] transition-colors',
        active
          ? 'border-[color-mix(in_srgb,var(--color-accent)_40%,transparent)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] font-medium text-[var(--color-accent)]'
          : 'border-border bg-surface text-text-secondary hover:bg-fill',
      )}
    >
      {label}
    </button>
  );
}

function TemplatesView({
  category,
  onPick,
}: {
  category: string;
  onPick: (t: Record<string, unknown>) => void;
}) {
  const [search, setSearch] = useState('');
  const [framework, setFramework] = useState('');
  const { data: templates = [], isLoading } = useQuery({
    queryKey: ['deploy-templates', category],
    queryFn: async () => {
      const res = await api.get('/deploy/templates', { params: { category } });
      return ((res.data as { templates?: Record<string, unknown>[] }).templates ?? []) as Record<
        string,
        unknown
      >[];
    },
  });

  const frameworks = Array.from(
    new Set(
      templates
        .map((t) => String(t['framework'] ?? ''))
        .filter((fw) => fw.length > 0),
    ),
  );

  const query = search.toLowerCase();
  const filtered = templates.filter((t) => {
    const matchesSearch =
      query.length === 0 ||
      String(t['name'] ?? '').toLowerCase().includes(query) ||
      String(t['description'] ?? '').toLowerCase().includes(query);
    const matchesFramework =
      framework.length === 0 || String(t['framework'] ?? '') === framework;
    return matchesSearch && matchesFramework;
  });

  return (
    <div className="flex flex-col gap-3">
      <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search templates…" />
      {frameworks.length > 0 && (
        <div className="flex gap-1.5 overflow-x-auto pb-0.5">
          <FilterChip label="All" active={framework === ''} onClick={() => setFramework('')} />
          {frameworks.map((fw) => (
            <FilterChip
              key={fw}
              label={fw}
              active={framework === fw}
              onClick={() => setFramework(fw)}
            />
          ))}
        </div>
      )}
      {isLoading ? (
        <div className="py-8 text-center text-[length:var(--text-body)] text-text-muted">Loading…</div>
      ) : filtered.length === 0 ? (
        <div className="py-8 text-center text-[length:var(--text-body)] text-text-muted">
          {templates.length === 0 ? 'No templates available' : 'No templates match your search'}
        </div>
      ) : (
        <div className="grid max-h-72 grid-cols-2 gap-2 overflow-y-auto">
          {filtered.map((t, i) => {
            const fw = String(t['framework'] ?? '');
            return (
              <button
                key={i}
                onClick={() => onPick(t)}
                className="flex flex-col items-start rounded-[var(--radius)] border border-border bg-surface p-3 text-left hover:bg-fill"
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)] bg-fill text-[var(--color-accent)]">
                  <LayoutTemplate size={18} />
                </div>
                <div className="mt-2.5 text-[length:var(--text-body)] font-medium text-text-primary">
                  {String(t['name'] ?? 'Template')}
                </div>
                <div className="mt-0.5 line-clamp-2 text-[length:var(--text-caption)] text-text-muted">
                  {String(t['description'] ?? '')}
                </div>
                {fw.length > 0 && (
                  <span className="mt-2">
                    <AppBadge label={fw} color="var(--color-accent)" />
                  </span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

function RepoView({ onPick }: { onPick: (r: Record<string, unknown>) => void }) {
  const [connectionId, setConnectionId] = useState<string | null>(null);
  const [repoSearch, setRepoSearch] = useState('');
  const { data: connections = [] } = useQuery({
    queryKey: ['git-connections'],
    queryFn: async () => {
      const res = await api.get('/deploy/git/connections');
      return ((res.data as { connections?: Record<string, unknown>[] }).connections ?? []) as Record<
        string,
        unknown
      >[];
    },
  });
  const { data: repos = [] } = useQuery({
    queryKey: ['git-repos', connectionId],
    enabled: !!connectionId,
    queryFn: async () => {
      const res = await api.get(`/deploy/git/connections/${connectionId}/repos`);
      return ((res.data as { repos?: Record<string, unknown>[] }).repos ?? []) as Record<
        string,
        unknown
      >[];
    },
  });

  if (connections.length === 0) {
    return (
      <div className="flex flex-col items-center gap-3 py-8 text-center">
        <GitBranch size={22} className="text-text-subtle" />
        <div className="text-[length:var(--text-body)] text-text-muted">
          No Git connections. Connect a provider first.
        </div>
      </div>
    );
  }

  const query = repoSearch.toLowerCase();
  const filteredRepos =
    query.length === 0
      ? repos
      : repos.filter(
          (r) =>
            String(r['name'] ?? '').toLowerCase().includes(query) ||
            String(r['fullName'] ?? '').toLowerCase().includes(query),
        );

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-1.5">
        {connections.map((c) => {
          const id = String(c['$id'] ?? c['id']);
          return (
            <button
              key={id}
              onClick={() => {
                setConnectionId(id);
                setRepoSearch('');
              }}
              className={cn(
                'rounded-[var(--radius-6)] border px-2.5 py-1 text-[length:var(--text-caption)]',
                connectionId === id
                  ? 'border-[var(--color-accent)] bg-fill-active text-text-primary'
                  : 'border-border text-text-muted',
              )}
            >
              {String(c['name'] ?? c['provider'] ?? 'Connection')}
            </button>
          );
        })}
      </div>
      <Input
        value={repoSearch}
        onChange={(e) => setRepoSearch(e.target.value)}
        placeholder="Search repositories…"
      />
      <div className="grid max-h-64 gap-1.5 overflow-y-auto">
        {filteredRepos.map((r, i) => (
          <button
            key={i}
            onClick={() => onPick(r)}
            className="flex items-center justify-between rounded-[var(--radius)] border border-border bg-surface px-3 py-2 text-left hover:bg-fill"
          >
            <span className="text-[length:var(--text-body)] text-text-primary">
              {String(r['fullName'] ?? r['name'] ?? 'repo')}
            </span>
            <Button size="sm" variant="outline">
              Deploy
            </Button>
          </button>
        ))}
        {filteredRepos.length === 0 && (
          <div className="py-5 text-center text-[length:var(--text-body)] text-text-muted">
            {repos.length === 0 ? 'No repositories found' : 'No repositories match your search'}
          </div>
        )}
      </div>
    </div>
  );
}
