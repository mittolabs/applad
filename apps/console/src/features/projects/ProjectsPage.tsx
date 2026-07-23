import { useEffect, useState, type ReactNode } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Building2, FolderGit2, MoreVertical, Plus, Settings, Trash2, UserPlus } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useOrgs, useProjects, type Project } from '@/api/queries';
import { useOrgStore } from '@/stores/org';
import { StandaloneLayout } from '@/shell/StandaloneLayout';
import { PageTabs } from '@/components/page-tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { SearchListHeader, SearchListFooter } from '@/components/search-list';
import { EmptyState } from '@/components/empty-state';
import { IdText } from '@/components/id-text';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormDialog, ConfirmDialog, TextField, FormField } from '@/components/form-dialog';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import { useTabIndex } from '@/hooks/use-tab-param';

const TABS = ['Projects', 'Members', 'Roles', 'Usage', 'Activity', 'Settings'];

export function ProjectsPage() {
  const { orgId } = useParams<{ orgId: string }>();
  const { currentOrgId, setCurrentOrg } = useOrgStore();
  const { data: orgs = [], isLoading: orgsLoading, isSuccess: orgsFetched } = useOrgs();
  const [tab, setTab] = useTabIndex(TABS);
  const [params, setParams] = useSearchParams();

  // ⌘K commands land here as ?create=project|org — open the matching dialog.
  const [projectCreateNonce, setProjectCreateNonce] = useState(0);
  const [orgCreateOpen, setOrgCreateOpen] = useState(false);
  useEffect(() => {
    const c = params.get('create');
    if (!c) return;
    if (c === 'project') {
      setTab(0);
      setProjectCreateNonce((n) => n + 1);
    } else if (c === 'org') {
      setOrgCreateOpen(true);
    }
    const p = new URLSearchParams(params);
    p.delete('create');
    setParams(p, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params]);

  // Sync :orgId from the URL into the store.
  useEffect(() => {
    if (orgId && orgId !== currentOrgId) setCurrentOrg(orgId);
  }, [orgId, currentOrgId, setCurrentOrg]);

  // A stored org id only counts once it resolves to a real org. After an org is
  // deleted, localStorage still holds its id; trusting it blindly renders an
  // empty ghost workspace, so fall back to the first org the user actually has.
  const orgsLoaded = orgs.length > 0;
  const storedIsValid = currentOrgId != null && orgs.some((o) => o.$id === currentOrgId);
  const activeOrgId = orgId ?? (storedIsValid ? currentOrgId : orgs[0]?.$id) ?? null;
  const org = orgs.find((o) => o.$id === activeOrgId);

  // Heal the store when it points at an org that no longer exists.
  useEffect(() => {
    if (orgsLoaded && currentOrgId != null && !storedIsValid) {
      setCurrentOrg(orgs[0]?.$id ?? null);
    }
  }, [orgsLoaded, currentOrgId, storedIsValid, orgs, setCurrentOrg]);

  // No organizations at all — the account has none, or the last one was just
  // deleted. The project workspace makes no sense here (a project must belong to
  // an org), so offer the one action that does: create an organization. Gated on
  // a completed fetch so an in-flight load does not flash this state.
  if (orgsFetched && !orgsLoading && orgs.length === 0) {
    return (
      <StandaloneLayout showOrg={false}>
        <div className="mx-auto flex w-full max-w-[1200px] flex-1 flex-col px-4 py-8 sm:px-8 lg:px-12">
          <EmptyState
            icon={Building2}
            title="No organizations"
            subtitle="An organization holds your projects and team. Create one to get started."
            actionLabel="Create organization"
            onAction={() => setOrgCreateOpen(true)}
          />
        </div>
        <CreateOrgDialog open={orgCreateOpen} onOpenChange={setOrgCreateOpen} />
      </StandaloneLayout>
    );
  }

  return (
    <StandaloneLayout>
      {/* Ports projects_page.dart: Center → ConstrainedBox(maxWidth 1200) → padding pageHPad. */}
      <div className="mx-auto w-full max-w-[1200px] flex-1 px-4 py-8 sm:px-8 lg:px-12">
        <div className="flex items-start justify-between">
          <h1 className="text-[length:var(--text-h2)] font-bold text-text-primary">
            {org?.name ?? 'Workspace'}
          </h1>
          <div className="flex items-center gap-3">
            <MemberAvatarStack orgId={activeOrgId} fallback={(org?.name ?? 'W')[0]?.toUpperCase()} />
            <Button variant="secondary" size="sm" onClick={() => setTab(1)}>
              <UserPlus size={14} />
              Invite
            </Button>
          </div>
        </div>

        <div className="mt-6">
          <PageTabs tabs={TABS} selected={tab} onChange={setTab} />
        </div>

        <div className="mt-6">
          {tab === 0 && <ProjectsTab orgId={activeOrgId} openNonce={projectCreateNonce} />}
          {tab === 1 && <MembersTab orgId={activeOrgId} />}
          {tab === 2 && <RolesTab />}
          {tab === 3 && <UsageTab orgId={activeOrgId} />}
          {tab === 4 && <ActivityTab orgId={activeOrgId} />}
          {tab === 5 && <SettingsTab orgId={activeOrgId} orgName={org?.name} />}
        </div>
      </div>
      <CreateOrgDialog open={orgCreateOpen} onOpenChange={setOrgCreateOpen} />
    </StandaloneLayout>
  );
}

/* Overlapping member avatars in the workspace heading (ports the Flutter stack). */
function MemberAvatarStack({ orgId, fallback }: { orgId: string | null; fallback?: string }) {
  const { data: members = [] } = useQuery({
    queryKey: ['org-members', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get(`/organizations/${orgId}/members`);
      return ((res.data as { members?: Member[] }).members ?? []) as Member[];
    },
  });
  if (members.length === 0) {
    return fallback ? (
      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-fill text-[length:var(--text-caption)] font-semibold text-text-secondary">
        {fallback}
      </div>
    ) : null;
  }
  const shown = members.slice(0, 4);
  const extra = members.length - shown.length;
  return (
    <div className="flex items-center -space-x-2">
      {shown.map((m) => (
        <div
          key={m.$id}
          title={m.name || m.email}
          className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-background bg-[var(--color-accent)] text-[length:var(--text-caption)] font-semibold text-white"
        >
          {(m.name || m.email)[0]?.toUpperCase()}
        </div>
      ))}
      {extra > 0 && (
        <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-background bg-fill text-[length:var(--text-caption)] font-medium text-text-secondary">
          +{extra}
        </div>
      )}
    </div>
  );
}

/* Create organization — wired from the ⌘K "Create organization" command. */
function CreateOrgDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { setCurrentOrg } = useOrgStore();
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: async () => {
      const res = await api.post('/organizations', { name });
      return res.data as { $id?: string; id?: string };
    },
    onSuccess: (o) => {
      qc.invalidateQueries({ queryKey: ['organizations'] });
      const id = String(o.$id ?? o.id ?? '');
      onOpenChange(false);
      setName('');
      if (id) {
        setCurrentOrg(id);
        navigate(`/org/${id}/projects`);
      }
    },
    onError: (e) => setError(friendlyError(e)),
  });

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create organization"
      subtitle="Organizations group your projects and team."
      submitLabel="Create"
      loading={create.isPending}
      submitDisabled={!name.trim()}
      onSubmit={() => {
        setError(null);
        create.mutate();
      }}
    >
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="My organization"
        autoFocus
        error={error ?? undefined}
      />
    </FormDialog>
  );
}

const CARD_COLORS = ['#3472A4', '#0E7490', '#7C3AED', '#059669', '#D97706', '#DC2626'];
function cardColor(seed: string): string {
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0;
  return CARD_COLORS[h % CARD_COLORS.length];
}
function relTime(v: unknown): string {
  if (!v) return '';
  const d = new Date(String(v));
  if (Number.isNaN(d.getTime())) return '';
  const days = Math.floor((Date.parse('2026-07-17') - d.getTime()) / 86_400_000);
  if (days <= 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 30) return `${days}d ago`;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function ProjectsTab({ orgId, openNonce = 0 }: { orgId: string | null; openNonce?: number }) {
  const { data: projects = [], isLoading } = useProjects();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);

  // Open the create dialog when the ⌘K "Create project" command fires.
  useEffect(() => {
    if (openNonce > 0) setCreating(true);
  }, [openNonce]);

  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [perPage, setPerPage] = useState(6);
  const [page, setPage] = useState(1);
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null);

  const create = useMutation({
    mutationFn: async () => {
      const path = orgId ? `/organizations/${orgId}/projects` : '/projects';
      const res = await api.post(path, { name });
      return res.data as Project;
    },
    onSuccess: (proj) => {
      qc.invalidateQueries({ queryKey: ['projects'] });
      setCreating(false);
      setName('');
      const id = String((proj as { $id?: string; id?: string }).$id ?? (proj as { id?: string }).id);
      if (id) navigate(`/project/${id}/overview`);
    },
    onError: (e) => setError(friendlyError(e)),
  });

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`/projects/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] });
      setDeleteTarget(null);
    },
  });

  const filtered = projects.filter((p) => {
    const q = search.trim().toLowerCase();
    return !q || p.name.toLowerCase().includes(q) || p.$id.toLowerCase().includes(q);
  });
  const start = (page - 1) * perPage;
  const paged = filtered.slice(start, start + perPage);

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        value={search}
        onChange={(v) => {
          setSearch(v);
          setPage(1);
        }}
        trailing={
          <Button size="sm" onClick={() => setCreating(true)}>
            <Plus size={14} />
            Create project
          </Button>
        }
      />

      {!isLoading && filtered.length === 0 && search === '' ? (
        <EmptyState
          icon={FolderGit2}
          title="No projects yet"
          subtitle="Create your first project to get started."
          actionLabel="Create project"
          onAction={() => setCreating(true)}
        />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {paged.map((p) => {
              const color = cardColor(p.$id);
              return (
                <div
                  key={p.$id}
                  role="button"
                  tabIndex={0}
                  onClick={() => navigate(`/project/${p.$id}/overview`)}
                  onKeyDown={(e) => e.key === 'Enter' && navigate(`/project/${p.$id}/overview`)}
                  className="group relative flex min-h-[176px] cursor-pointer flex-col gap-4 rounded-[var(--radius-12)] border border-border bg-surface p-5 text-left transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_50%,var(--border))]"
                >
                  {/* kebab menu — ports the project card PopupMenuButton (Settings / Delete) */}
                  <div className="absolute right-3 top-3">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <button
                          onClick={(e) => e.stopPropagation()}
                          className="rounded-[var(--radius-6)] p-1 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-text-primary focus:opacity-100 group-hover:opacity-100 data-[state=open]:opacity-100"
                          aria-label="Project options"
                        >
                          <MoreVertical size={16} />
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent
                        align="end"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <DropdownMenuItem onSelect={() => navigate(`/project/${p.$id}/settings`)}>
                          <Settings size={14} />
                          Settings
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem destructive onSelect={() => setDeleteTarget(p)}>
                          <Trash2 size={14} />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>

                  <div
                    className="flex h-12 w-12 items-center justify-center rounded-[var(--radius-12)] text-[length:var(--text-h1)] font-semibold text-white"
                    style={{ backgroundColor: `color-mix(in srgb, ${color} 85%, black)` }}
                  >
                    {p.name[0]?.toUpperCase()}
                  </div>
                  <div className="flex-1">
                    <div className="truncate text-[length:var(--text-control)] font-medium text-text-primary">
                      {p.name}
                    </div>
                    {typeof p['description'] === 'string' && p['description'] && (
                      <div className="mt-0.5 truncate text-[length:var(--text-body)] text-text-muted">
                        {String(p['description'])}
                      </div>
                    )}
                  </div>
                  <div className="text-[length:var(--text-caption)] text-text-subtle">
                    {relTime(p['$createdAt'] ?? p['createdAt']) || 'Recently'}
                  </div>
                </div>
              );
            })}

            {/* Dashed "create a new project" card */}
            <button
              onClick={() => setCreating(true)}
              className="flex min-h-[176px] flex-col items-center justify-center gap-2 rounded-[var(--radius-12)] border border-dashed border-border text-text-muted transition-colors hover:border-[var(--color-accent)] hover:text-text-secondary"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-full border border-border">
                <Plus size={16} />
              </div>
              <span className="text-[length:var(--text-body)]">Create a new project</span>
            </button>
          </div>

          <SearchListFooter
            total={filtered.length}
            perPage={perPage}
            currentPage={page}
            itemLabel="projects"
            onPerPageChange={(n) => {
              setPerPage(n);
              setPage(1);
            }}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => p + 1)}
          />
        </>
      )}

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create project"
        subtitle="Projects group your databases, functions, and storage."
        submitLabel="Create"
        loading={create.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => {
          setError(null);
          create.mutate();
        }}
      >
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My project"
          autoFocus
          error={error ?? undefined}
        />
      </FormDialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete project"
        message={
          <>
            This permanently deletes <b>{deleteTarget?.name}</b> and all its data. This
            cannot be undone.
          </>
        }
        confirmLabel="Delete project"
        loading={del.isPending}
        onConfirm={() => deleteTarget && del.mutate(deleteTarget.$id)}
      />
    </div>
  );
}

interface Member {
  $id: string;
  email: string;
  name?: string;
  role?: string;
}

function MembersTab({ orgId }: { orgId: string | null }) {
  const qc = useQueryClient();
  const { data: members = [] } = useQuery({
    queryKey: ['org-members', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get(`/organizations/${orgId}/members`);
      return ((res.data as { members?: Member[] }).members ?? []) as Member[];
    },
  });

  const [inviting, setInviting] = useState(false);
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [role, setRole] = useState('member');
  const [removeTarget, setRemoveTarget] = useState<Member | null>(null);
  const [inviteLink, setInviteLink] = useState<string | null>(null);

  const invalidate = () => qc.invalidateQueries({ queryKey: ['org-members', orgId] });

  const invite = useMutation({
    mutationFn: () =>
      api.post(`/organizations/${orgId}/members`, {
        email,
        role,
        ...(name.trim() ? { name: name.trim() } : {}),
      }),
    onSuccess: (res) => {
      invalidate();
      setInviting(false);
      // Surface the link. On a self-hosted instance with no SMTP configured
      // nothing is emailed, so without this the invite is unreachable and the
      // person can never create their account.
      const token = (res.data as { inviteToken?: string }).inviteToken;
      if (token) setInviteLink(`${window.location.origin}/invite/${token}`);
      setEmail('');
      setName('');
      setRole('member');
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/organizations/${orgId}/members/${id}`),
    onSuccess: () => {
      invalidate();
      setRemoveTarget(null);
    },
  });
  const updateRole = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) =>
      api.patch(`/organizations/${orgId}/members/${id}`, { role }),
    onSuccess: invalidate,
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
          Members
        </div>
        <Button size="sm" onClick={() => setInviting(true)}>
          <Plus size={14} />
          Invite member
        </Button>
      </div>

      <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
        {members.length === 0 && (
          <div className="px-4 py-10 text-center text-[length:var(--text-body)] text-text-muted">
            No members yet.
          </div>
        )}
        {members.map((m) => (
          <div
            key={m.$id}
            className="group flex items-center gap-3 border-b border-[var(--fill)] px-4 py-3 last:border-0"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-accent)] text-[length:var(--text-caption)] font-semibold text-white">
              {(m.name || m.email)[0]?.toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-[length:var(--text-body)] text-text-primary">
                {m.name || m.email}
              </div>
              <div className="truncate text-[length:var(--text-caption)] text-text-muted">
                {m.email}
              </div>
            </div>
            <Select
              value={m.role ?? 'member'}
              onValueChange={(role) => updateRole.mutate({ id: m.$id, role })}
            >
              <SelectTrigger className="h-7 w-[112px] text-[length:var(--text-caption)] capitalize">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {['owner', 'admin', 'member'].map((r) => (
                  <SelectItem key={r} value={r} className="capitalize">
                    {r}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <button
              onClick={() => setRemoveTarget(m)}
              className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
              aria-label="Remove member"
            >
              <Trash2 size={14} />
            </button>
          </div>
        ))}
      </div>

      {/* The invite link, shown once. Self-hosted instances usually have no
          SMTP configured, so this is how the invite actually reaches someone. */}
      <FormDialog
        open={inviteLink !== null}
        onOpenChange={(o) => !o && setInviteLink(null)}
        title="Invite created"
        subtitle="Send this link to the person you invited. It creates their account and adds them to this organization."
        submitLabel="Copy link"
        onSubmit={() => {
          if (inviteLink) navigator.clipboard.writeText(inviteLink);
          setInviteLink(null);
        }}
      >
        <div className="break-all rounded-[var(--radius)] border border-field-border bg-fill px-3 py-2.5 font-mono text-[length:var(--text-caption)] text-text-primary">
          {inviteLink}
        </div>
      </FormDialog>

      <FormDialog
        open={inviting}
        onOpenChange={setInviting}
        title="Invite member"
        subtitle="Invite someone to collaborate on this organization."
        submitLabel="Send invite"
        loading={invite.isPending}
        submitDisabled={!email.trim()}
        onSubmit={() => invite.mutate()}
      >
        <TextField
          label="Email address"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="name@example.com"
          autoFocus
        />
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Optional"
        />
        <FormField label="Role">
          <div className="flex gap-1.5">
            {['owner', 'admin', 'member'].map((r) => (
              <button
                key={r}
                type="button"
                onClick={() => setRole(r)}
                className={`rounded-[var(--radius-6)] border px-2.5 py-1 text-[length:var(--text-caption)] capitalize ${
                  role === r
                    ? 'border-[var(--color-accent)] bg-fill-active text-text-primary'
                    : 'border-border text-text-muted'
                }`}
              >
                {r}
              </button>
            ))}
          </div>
          <p className="mt-1.5 text-[length:var(--text-caption)] text-text-subtle">
            {role === 'owner'
              ? 'Full access to all resources and settings'
              : role === 'admin'
                ? 'Manage projects, members, and settings'
                : 'Access assigned projects only'}
          </p>
        </FormField>
      </FormDialog>

      <ConfirmDialog
        open={removeTarget !== null}
        onOpenChange={(o) => !o && setRemoveTarget(null)}
        title="Remove member"
        message={`Remove ${removeTarget?.email} from this organization?`}
        confirmLabel="Remove"
        loading={remove.isPending}
        onConfirm={() => removeTarget && remove.mutate(removeTarget.$id)}
      />
    </div>
  );
}

const ROLE_MATRIX: [string, [boolean, boolean, boolean]][] = [
  ['View projects', [true, true, true]],
  ['Create & edit projects', [true, true, false]],
  ['Invite & remove members', [true, true, false]],
  ['Manage billing', [true, false, false]],
  ['Delete organization', [true, false, false]],
];

function RolesTab() {
  return (
    <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
      <table className="w-full text-left">
        <thead>
          <tr className="border-b border-border text-[length:var(--text-label)] text-text-muted">
            <th className="px-4 py-2.5 font-medium">Permission</th>
            {['Owner', 'Admin', 'Member'].map((r) => (
              <th key={r} className="px-4 py-2.5 text-center font-medium">
                {r}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {ROLE_MATRIX.map(([perm, allowed]) => (
            <tr key={perm} className="border-b border-[var(--fill)] last:border-0">
              <td className="px-4 py-3 text-[length:var(--text-body)] text-text-secondary">
                {perm}
              </td>
              {allowed.map((ok, i) => (
                <td key={i} className="px-4 py-3 text-center">
                  <span className={ok ? 'text-status-success' : 'text-text-subtle'}>
                    {ok ? '✓' : '—'}
                  </span>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function UsageTab({ orgId }: { orgId: string | null }) {
  const { data } = useQuery({
    queryKey: ['org-stats', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get(`/organizations/${orgId}/stats`);
      return res.data as Record<string, unknown>;
    },
  });
  // Keys match the backend org-stats response (organizations/service.go).
  const stats: [string, string][] = [
    ['Projects', String(data?.['totalProjects'] ?? 0)],
    ['Users', String(data?.['totalUsers'] ?? 0)],
    ['Storage', fmtBytes(data?.['totalStorage'])],
    ['Executions', String(data?.['totalExecutions'] ?? 0)],
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {stats.map(([label, value]) => (
        <div key={label} className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
          <div className="text-[length:var(--text-caption)] text-text-muted">{label}</div>
          <div className="mt-1 text-[length:var(--text-h1)] font-semibold text-text-primary">
            {value}
          </div>
        </div>
      ))}
    </div>
  );
}

function fmtBytes(v: unknown): string {
  const n = typeof v === 'number' ? v : Number(v);
  if (!n || Number.isNaN(n)) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  let size = n;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i++;
  }
  return `${size % 1 === 0 ? size : size.toFixed(1)} ${units[i]}`;
}

function ActivityTab({ orgId }: { orgId: string | null }) {
  const { data: events = [] } = useQuery({
    queryKey: ['org-activity', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get(`/organizations/${orgId}/activity`, { params: { limit: 50 } });
      return ((res.data as { activity?: Record<string, unknown>[] }).activity ?? []) as Record<
        string,
        unknown
      >[];
    },
  });
  if (events.length === 0) {
    return (
      <div className="rounded-[var(--radius-10)] border border-dashed border-border py-16 text-center text-[length:var(--text-body)] text-text-muted">
        No recent activity.
      </div>
    );
  }
  return (
    <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
      {events.map((e, i) => (
        <div
          key={i}
          className="flex items-center justify-between border-b border-[var(--fill)] px-4 py-3 text-[length:var(--text-body)] last:border-0"
        >
          <span className="text-text-secondary">{String(e['action'] ?? e['event'] ?? 'Event')}</span>
          <span className="text-[length:var(--text-caption)] text-text-subtle">
            {String(e['createdAt'] ?? e['$createdAt'] ?? '')}
          </span>
        </div>
      ))}
    </div>
  );
}

function SettingsTab({ orgId, orgName }: { orgId: string | null; orgName?: string }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { currentOrgId, setCurrentOrg } = useOrgStore();
  const [name, setName] = useState(orgName ?? '');
  const [confirmDelete, setConfirmDelete] = useState(false);

  useEffect(() => setName(orgName ?? ''), [orgName]);

  const rename = useMutation({
    mutationFn: () => api.patch(`/organizations/${orgId}`, { name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['organizations'] }),
  });
  const del = useMutation({
    mutationFn: () => api.delete(`/organizations/${orgId}`),
    onSuccess: () => {
      // Drop the stored pointer if it was this org, or the next load lands on a
      // ghost workspace (an id that no longer resolves to any org).
      if (currentOrgId === orgId) setCurrentOrg(null);
      qc.invalidateQueries({ queryKey: ['organizations'] });
      navigate('/projects');
    },
  });

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <SettingsCard title="Organization name" subtitle="This is how your organization appears across the console.">
        <div className="flex flex-col gap-4">
          <Input value={name} onChange={(e) => setName(e.target.value)} />
          <div className="flex justify-end">
            <Button loading={rename.isPending} disabled={!name.trim()} onClick={() => rename.mutate()}>
              Save changes
            </Button>
          </div>
        </div>
      </SettingsCard>

      <SettingsCard title="Organization ID" subtitle="Use this ID when calling the API or configuring the SDKs.">
        {orgId ? <IdText id={orgId} /> : <span className="text-text-subtle">—</span>}
      </SettingsCard>

      <SettingsCard
        danger
        icon={Trash2}
        title="Delete organization"
        subtitle="Permanently removes this organization and all of its projects and data. This cannot be undone."
      >
        <div className="flex md:justify-end">
          <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
            Delete organization
          </Button>
        </div>
      </SettingsCard>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete organization"
        message="This permanently deletes the organization and all its projects. This cannot be undone."
        confirmLabel="Delete organization"
        loading={del.isPending}
        onConfirm={() => del.mutate()}
      />
    </div>
  );
}

/* Two-column settings section: title + subtitle on the left, controls on the
 * right — matches the account page. */
function SettingsCard({
  title,
  subtitle,
  icon: Icon,
  danger,
  children,
}: {
  title: string;
  subtitle?: string;
  icon?: typeof Trash2;
  danger?: boolean;
  children: ReactNode;
}) {
  return (
    <div
      className={`rounded-[var(--radius-10)] border bg-surface p-5 md:p-6 ${
        danger ? 'border-[color-mix(in_srgb,var(--color-danger)_40%,var(--border))]' : 'border-border'
      }`}
    >
      <div className="flex flex-col gap-5 md:flex-row md:gap-8">
        <div className="md:w-2/5">
          <div
            className={`flex items-center gap-2 text-[length:var(--text-control)] font-medium ${
              danger ? 'text-[var(--status-danger)]' : 'text-text-primary'
            }`}
          >
            {Icon && <Icon size={16} />}
            {title}
          </div>
          {subtitle && (
            <div className="mt-2 text-[length:var(--text-body)] text-text-muted">{subtitle}</div>
          )}
        </div>
        <div className="flex-1">{children}</div>
      </div>
    </div>
  );
}
