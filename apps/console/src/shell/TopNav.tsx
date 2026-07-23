import { useEffect, useMemo, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Check, ChevronsUpDown, Plus, Search } from 'lucide-react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { SearchModal } from '@/components/search-modal';
import { useOrgs, useProjects, useEnvironments } from '@/api/queries';
import { useOrgStore } from '@/stores/org';
import { useEnvStore } from '@/stores/environment';
import { useIsNavCompact } from '@/hooks/use-media-query';
import { NavOverflowMenu, SupportButton, UserMenu } from './navbar-popovers';
import { extensionNavActions } from '@/extensions';

/*
 * The single top navigation bar — used by BOTH the project shell and the
 * shell-less pages (projects/account). Consolidates the two Flutter navbars
 * (shell.dart _TopNavBar + app_navbar.dart) into one component.
 *
 * - Always: logo, org switcher (unless showOrg=false), Support, module actions,
 *   ⌘K search, user menu.
 * - In a project (projectId set): also the project switcher + env badge.
 */
export function TopNav({
  projectId,
  showOrg = true,
}: {
  projectId?: string;
  showOrg?: boolean;
}) {
  const location = useLocation();
  const [searchOpen, setSearchOpen] = useState(false);
  const compact = useIsNavCompact();

  // ⌘K / Ctrl-K command palette.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setSearchOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const section = useMemo(() => {
    if (!projectId) return 'overview';
    const base = `/project/${projectId}/`;
    return location.pathname.startsWith(base)
      ? location.pathname.slice(base.length).split('/')[0] || 'overview'
      : 'overview';
  }, [location.pathname, projectId]);

  return (
    <header className="flex h-[52px] shrink-0 items-center gap-2 border-b border-border bg-surface-alt px-4">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Link to="/projects" className="flex shrink-0 items-center">
          <img src="/icon.png" alt="Applad" className="h-[42px] w-[42px] rounded-[var(--radius)]" />
        </Link>
        {showOrg && (
          <>
            <span className="shrink-0 text-text-subtle">/</span>
            <OrgSwitcher />
          </>
        )}
        {projectId && (
          <>
            <span className="shrink-0 text-text-subtle">/</span>
            <ProjectSwitcher projectId={projectId} section={section} />
            <EnvBadge />
          </>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-4">
        {compact ? (
          <NavOverflowMenu />
        ) : (
          <>
            <SupportButton />
            {/* Modules compiled into this build contribute buttons here. A
                default build contributes none. */}
            {extensionNavActions().map((Action, i) => (
              <Action key={i} />
            ))}
          </>
        )}
        <button
          onClick={() => setSearchOpen(true)}
          className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-6)] text-text-muted transition-colors hover:bg-fill hover:text-text-primary"
          title="Search (⌘K)"
          aria-label="Search"
        >
          <Search size={17} />
        </button>
        <UserMenu />
      </div>
      <SearchModal open={searchOpen} onOpenChange={setSearchOpen} />
    </header>
  );
}

function OrgSwitcher() {
  const { data: orgs = [] } = useOrgs();
  const { currentOrgId, setCurrentOrg } = useOrgStore();
  const navigate = useNavigate();
  const current = orgs.find((o) => o.$id === currentOrgId) ?? orgs[0];
  return (
    <SwitcherPopover label={current?.name ?? 'Organization'}>
      {(close) => (
        <>
          {orgs.map((o) => (
            <SwitcherItem
              key={o.$id}
              label={o.name}
              selected={o.$id === current?.$id}
              onClick={() => {
                setCurrentOrg(o.$id);
                navigate(`/org/${o.$id}/projects`);
                close();
              }}
            />
          ))}
          {/* Creating another org was only reachable via ⌘K or a hidden URL
              param, so the switcher looked like a dead end once you had one org.
              Route through /projects?create=org, which opens the create dialog. */}
          <div className="my-1 h-px bg-border" />
          <button
            onClick={() => {
              navigate('/projects?create=org');
              close();
            }}
            className="flex w-full items-center gap-2 rounded-[var(--radius-6)] px-2 py-1.5 text-[length:var(--text-body)] text-text-secondary transition-colors hover:bg-fill hover:text-text-primary"
          >
            <Plus size={14} className="text-text-subtle" />
            <span>Create organization</span>
          </button>
        </>
      )}
    </SwitcherPopover>
  );
}

function ProjectSwitcher({ projectId, section }: { projectId?: string; section: string }) {
  const { data: projects = [] } = useProjects();
  const navigate = useNavigate();
  const current = projects.find((p) => p.$id === projectId);
  return (
    <SwitcherPopover label={current?.name ?? 'Project'}>
      {(close) =>
        projects.map((p) => (
          <SwitcherItem
            key={p.$id}
            label={p.name}
            selected={p.$id === projectId}
            onClick={() => {
              navigate(`/project/${p.$id}/${section}`);
              close();
            }}
          />
        ))
      }
    </SwitcherPopover>
  );
}

const ENV_COLOR: Record<string, string> = {
  production: 'var(--status-danger)',
  staging: 'var(--status-warning)',
  development: 'var(--color-accent-3, #8b5cf6)',
};

function EnvBadge() {
  const { data: envs = [] } = useEnvironments();
  const { currentEnvId, setCurrentEnv } = useEnvStore();
  const current =
    envs.find((e) => e.$id === currentEnvId) ?? envs.find((e) => e.isDefault) ?? envs[0];
  if (envs.length === 0) return null;
  const color = ENV_COLOR[(current?.name ?? '').toLowerCase()] ?? 'var(--status-neutral)';
  return (
    <SwitcherPopover
      label={
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: color }} />
          {current?.name ?? 'Environment'}
        </span>
      }
    >
      {(close) =>
        envs.map((e) => (
          <SwitcherItem
            key={e.$id}
            label={e.name}
            selected={e.$id === current?.$id}
            onClick={() => {
              setCurrentEnv(e.$id);
              close();
            }}
          />
        ))
      }
    </SwitcherPopover>
  );
}

function SwitcherPopover({
  label,
  children,
}: {
  label: React.ReactNode;
  children: (close: () => void) => React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button className="flex min-w-0 items-center gap-1 rounded-[var(--radius-6)] px-2 py-1 text-[length:var(--text-body)] text-text-secondary transition-colors hover:bg-fill hover:text-text-primary">
          <span className="truncate">{label}</span>
          <ChevronsUpDown size={13} className="shrink-0 text-text-subtle" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-56 p-1">
        {children(() => setOpen(false))}
      </PopoverContent>
    </Popover>
  );
}

function SwitcherItem({
  label,
  selected,
  onClick,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="flex w-full items-center justify-between rounded-[var(--radius-6)] px-2 py-1.5 text-[length:var(--text-body)] text-text-secondary transition-colors hover:bg-fill"
    >
      <span className="truncate">{label}</span>
      {selected && <Check size={14} className="text-[var(--color-accent)]" />}
    </button>
  );
}
