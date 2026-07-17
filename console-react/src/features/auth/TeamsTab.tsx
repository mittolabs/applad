import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, MoreHorizontal, UserPlus, Users, UserX } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FormDialog, ConfirmDialog, TextField } from '@/components/form-dialog';
import { relativeTime } from './format';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Team ID', flex: 3 },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'total', label: 'Members', flex: 2 },
  { key: '$createdAt', label: 'Created', flex: 2, sortable: true },
];

function parseRoles(input: string): string[] {
  return input
    .split(',')
    .map((r) => r.trim())
    .filter(Boolean);
}

export function TeamsTab({ projectId }: { projectId: string }) {
  const list = useResourceList({ endpoint: '/teams', itemsKey: 'teams', scope: [projectId] });
  const [selected, setSelected] = useState<{ id: string; name: string } | null>(null);

  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [roles, setRoles] = useState('');

  const createTeam = useMutation({
    mutationFn: () => {
      const parsed = parseRoles(roles);
      return api
        .post('/teams', { name: name.trim(), ...(parsed.length ? { roles: parsed } : {}) })
        .then((r) => r.data);
    },
    onSuccess: () => {
      setCreating(false);
      setName('');
      setRoles('');
      list.refetch();
    },
  });

  if (selected) {
    return (
      <TeamDetail
        teamId={selected.id}
        teamName={selected.name}
        onBack={() => setSelected(null)}
        onMembershipChange={() => list.refetch()}
      />
    );
  }

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows as Row[]}
        getCellValue={(row, key) => {
          switch (key) {
            case '$id':
              return String(row['$id'] ?? '');
            case 'name':
              return String(row['name'] ?? '') || 'Unnamed';
            case 'total':
              return String(row['total'] ?? 0);
            case '$createdAt':
              return relativeTime(String(row['$createdAt'] ?? ''));
            default:
              return '';
          }
        }}
        rowIcon={() => Users}
        onRowClick={(row) =>
          setSelected({ id: String(row['$id'] ?? ''), name: String(row['name'] ?? 'Team') })
        }
        onDeleteRow={async (row) => {
          await api.delete(`/teams/${row['$id']}`);
          list.refetch();
        }}
        createLabel="Create team"
        onCreate={() => setCreating(true)}
        searchHint="Search by name or ID"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="teams"
        emptyIcon={Users}
        emptyTitle="No teams yet"
        emptySubtitle="Create a team to group users together."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create team"
        subtitle="Group users with shared roles"
        submitLabel="Create"
        loading={createTeam.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => createTeam.mutate()}
      >
        <TextField label="Team name" placeholder="e.g. Admins, Beta Testers" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <TextField
          label="Default roles (optional)"
          placeholder="Comma-separated, e.g. admin, viewer"
          value={roles}
          onChange={(e) => setRoles(e.target.value)}
        />
      </FormDialog>
    </>
  );
}

// ── Team detail ────────────────────────────────────────────────────────────

function TeamDetail({
  teamId,
  teamName,
  onBack,
  onMembershipChange,
}: {
  teamId: string;
  teamName: string;
  onBack: () => void;
  onMembershipChange: () => void;
}) {
  const qc = useQueryClient();
  const membershipsKey = ['/teams', teamId, 'memberships'];
  const query = useQuery({
    queryKey: membershipsKey,
    queryFn: () => api.get(`/teams/${teamId}/memberships`).then((r) => r.data as Record<string, unknown>),
  });

  const refresh = () => {
    qc.invalidateQueries({ queryKey: membershipsKey });
    onMembershipChange();
  };

  const [adding, setAdding] = useState(false);
  const [addEmail, setAddEmail] = useState('');
  const [addRoles, setAddRoles] = useState('');
  const [editing, setEditing] = useState<Record<string, unknown> | null>(null);
  const [editRoles, setEditRoles] = useState('');
  const [removing, setRemoving] = useState<Record<string, unknown> | null>(null);

  const addMember = useMutation({
    mutationFn: () =>
      api
        .post(`/teams/${teamId}/memberships`, { email: addEmail.trim(), roles: parseRoles(addRoles) })
        .then((r) => r.data),
    onSuccess: () => {
      setAdding(false);
      setAddEmail('');
      setAddRoles('');
      refresh();
    },
  });

  const memberEmail = (m: Record<string, unknown>) =>
    String(m['userEmail'] ?? m['invitedEmail'] ?? '—');

  const saveRoles = useMutation({
    mutationFn: async () => {
      if (!editing) return;
      const membershipId = String(editing['$id'] ?? '');
      // No PATCH membership endpoint exists — delete + re-invite with new roles.
      await api.delete(`/teams/${teamId}/memberships/${membershipId}`);
      await api.post(`/teams/${teamId}/memberships`, {
        email: memberEmail(editing),
        roles: parseRoles(editRoles),
      });
    },
    onSuccess: () => {
      setEditing(null);
      refresh();
    },
  });

  const removeMember = useMutation({
    mutationFn: () => {
      const membershipId = String(removing?.['$id'] ?? '');
      return api.delete(`/teams/${teamId}/memberships/${membershipId}`);
    },
    onSuccess: () => {
      setRemoving(null);
      refresh();
    },
  });

  const memberships = (query.data?.['memberships'] as Record<string, unknown>[] | undefined) ?? [];

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="flex items-center gap-2.5">
        <button
          type="button"
          onClick={onBack}
          className="rounded-[var(--radius-6)] p-1 text-text-muted transition-colors hover:bg-fill"
          aria-label="Back"
        >
          <ArrowLeft size={18} />
        </button>
        <Users size={18} className="text-text-muted" />
        <span className="text-[length:var(--text-title)] font-semibold text-text-primary">{teamName}</span>
        <span className="rounded-[var(--radius-sm)] bg-fill px-2 py-0.5 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-subtle">
          ID: {teamId}
        </span>
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => setAdding(true)}>
            <UserPlus size={15} />
            Add member
          </Button>
        </div>
      </div>

      <div className="mt-5">
        {query.error ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : query.isLoading ? (
          <div className="h-40 animate-pulse rounded-[var(--radius-10)] border border-border bg-surface" />
        ) : memberships.length === 0 ? (
          <EmptyState icon={UserX} title="No members yet" subtitle="Add members to this team." />
        ) : (
          <div className="overflow-hidden rounded-[var(--radius-10)] border border-border bg-surface">
            <div className="grid grid-cols-[3fr_3fr_2fr_2fr_40px] border-b border-border px-5 py-3 text-[length:var(--text-label)] font-medium text-text-muted">
              <span>Email</span>
              <span>Roles</span>
              <span>Status</span>
              <span>Joined</span>
              <span />
            </div>
            {memberships.map((m, i) => {
              const email = memberEmail(m);
              const roleList = Array.isArray(m['roles']) ? (m['roles'] as unknown[]).map(String) : [];
              const joined = m['joined'] === true || m['joined'] === 1;
              const createdAt = String(m['$createdAt'] ?? '');
              return (
                <div
                  key={String(m['$id'] ?? i)}
                  className="group grid grid-cols-[3fr_3fr_2fr_2fr_40px] items-center border-b border-border px-5 py-3.5 last:border-0 transition-colors hover:bg-fill-hover"
                >
                  <span className="truncate text-[length:var(--text-body)] text-text-primary">{email}</span>
                  <div className="flex flex-wrap gap-1.5">
                    {roleList.length === 0 ? (
                      <span className="text-[length:var(--text-label)] text-text-subtle">No roles</span>
                    ) : (
                      roleList.map((r) => <RoleChip key={r} label={r} />)
                    )}
                  </div>
                  <span>
                    <StatusChip label={joined ? 'active' : 'pending'} />
                  </span>
                  <span className="text-[length:var(--text-label)] text-text-muted">
                    {createdAt ? relativeTime(createdAt) : '—'}
                  </span>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button
                        type="button"
                        className="rounded-[var(--radius-6)] p-1 text-transparent transition-colors hover:bg-fill group-hover:text-text-muted"
                        aria-label="Member actions"
                      >
                        <MoreHorizontal size={14} />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        onClick={() => {
                          setEditRoles(roleList.join(', '));
                          setEditing(m);
                        }}
                      >
                        Edit roles
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-[var(--color-danger)]"
                        onClick={() => setRemoving(m)}
                      >
                        Remove
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <FormDialog
        open={adding}
        onOpenChange={setAdding}
        title="Add member"
        subtitle={`Invite a user to ${teamName}`}
        submitLabel="Add"
        loading={addMember.isPending}
        submitDisabled={!addEmail.trim()}
        onSubmit={() => addMember.mutate()}
      >
        <TextField label="Email address" placeholder="user@example.com" value={addEmail} onChange={(e) => setAddEmail(e.target.value)} autoFocus />
        <TextField label="Roles (optional)" placeholder="Comma-separated, e.g. admin, editor" value={addRoles} onChange={(e) => setAddRoles(e.target.value)} />
      </FormDialog>

      <FormDialog
        open={editing !== null}
        onOpenChange={(o) => !o && setEditing(null)}
        title="Edit roles"
        subtitle={editing ? memberEmail(editing) : undefined}
        submitLabel="Save"
        loading={saveRoles.isPending}
        onSubmit={() => saveRoles.mutate()}
      >
        <TextField label="Roles" placeholder="Comma-separated, e.g. admin, editor" value={editRoles} onChange={(e) => setEditRoles(e.target.value)} autoFocus />
      </FormDialog>

      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(o) => !o && setRemoving(null)}
        title="Remove member"
        message={removing ? `Remove ${memberEmail(removing)} from this team?` : ''}
        confirmLabel="Remove"
        loading={removeMember.isPending}
        onConfirm={() => removeMember.mutate()}
      />
    </div>
  );
}

function RoleChip({ label }: { label: string }) {
  return (
    <span
      className="rounded-[var(--radius-sm)] px-2 py-0.5 text-[length:var(--text-caption)] font-medium"
      style={{
        color: 'var(--color-accent)',
        backgroundColor: 'color-mix(in srgb, var(--color-accent) 15%, transparent)',
        border: '1px solid color-mix(in srgb, var(--color-accent) 30%, transparent)',
      }}
    >
      {label}
    </span>
  );
}
