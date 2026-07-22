import { useEffect, useState } from 'react';
import { useTabIndex } from '@/hooks/use-tab-param';
import type { ReactNode } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, ShieldOff, Table2, X } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { PageTabs } from '@/components/page-tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { IdText } from '@/components/id-text';
import { ConfirmDialog, FormDialog, FormField, TextField } from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';
import { toast } from '@/components/toast';
import { BackHeader, ChipGroup, str, type Json } from './shared';
import { RowsPanel } from './RowsPanel';
import { EntriesPanel } from './EntriesPanel';
import { ColumnsPanel } from './ColumnsPanel';
import { IndexesPanel } from './IndexesPanel';
import { RelationshipsPanel } from './RelationshipsPanel';

const BASE_TABS = ['Rows', 'Columns', 'Indexes', 'Relationships', 'Settings'];

export function TableDetail({
  dbId,
  tableId,
  tableName,
  onBack,
}: {
  dbId: string;
  tableId: string;
  tableName: string;
  onBack: () => void;
}) {
  // In the URL so a refresh stays on the tab somebody was reading.
  const [tab, setTab] = useTabIndex(BASE_TABS, undefined, 'view');
  const [showCreateColumn, setShowCreateColumn] = useState(false);

  // Content-enabled tables get an editorial Entries view in front of the raw grid.
  const { data: table } = useQuery({
    queryKey: ['db-table', dbId, tableId],
    queryFn: async () => (await api.get(`/databases/${dbId}/tables/${tableId}`)).data as Json,
  });
  const contentEnabled = Boolean(table?.['contentEnabled']);
  const tabs = contentEnabled ? ['Entries', ...BASE_TABS] : BASE_TABS;
  const active = tabs[tab] ?? 'Rows';

  // Toggling content mode changes the tab list; keep the selection in range.
  useEffect(() => {
    if (tab >= tabs.length) setTab(0);
  }, [tab, tabs.length]);

  const goToColumns = () => {
    setTab(tabs.indexOf('Columns'));
    setShowCreateColumn(true);
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <BackHeader title={tableName} subtitle={tableId} icon={Table2} onBack={onBack} />
      <PageTabs tabs={tabs} selected={tab} onChange={setTab} />

      {active === 'Entries' && <EntriesPanel dbId={dbId} tableId={tableId} />}
      {active === 'Rows' && (
        <RowsPanel dbId={dbId} tableId={tableId} onCreateColumn={goToColumns} />
      )}
      {active === 'Columns' && (
        <ColumnsPanel
          dbId={dbId}
          tableId={tableId}
          showCreate={showCreateColumn}
          setShowCreate={setShowCreateColumn}
        />
      )}
      {active === 'Indexes' && <IndexesPanel dbId={dbId} tableId={tableId} />}
      {active === 'Relationships' && <RelationshipsPanel dbId={dbId} tableId={tableId} />}
      {active === 'Settings' && (
        <TableSettings dbId={dbId} tableId={tableId} tableName={tableName} onDeleted={onBack} />
      )}
    </div>
  );
}

/* --- Settings tab --- */

type Permission = { role: string; action: string };

const PERM_ACTIONS: { value: string; label: string }[] = [
  { value: 'read', label: 'read' },
  { value: 'create', label: 'create' },
  { value: 'update', label: 'update' },
  { value: 'delete', label: 'delete' },
];

function SectionCard({
  title,
  subtitle,
  children,
}: {
  title?: string;
  subtitle?: string;
  children: ReactNode;
}) {
  return (
    <div className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
      {title && (
        <div className="mb-4">
          <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
            {title}
          </div>
          {subtitle && (
            <div className="mt-1 text-[length:var(--text-body)] text-text-muted">{subtitle}</div>
          )}
        </div>
      )}
      {children}
    </div>
  );
}

function TableSettings({
  dbId,
  tableId,
  tableName,
  onDeleted,
}: {
  dbId: string;
  tableId: string;
  tableName: string;
  onDeleted: () => void;
}) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;

  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [addingPerm, setAddingPerm] = useState(false);

  // Optimistic local state for the toggles (null = use API value).
  const [enabledOverride, setEnabledOverride] = useState<boolean | null>(null);
  const [rowSecurityOverride, setRowSecurityOverride] = useState<boolean | null>(null);
  const [contentOverride, setContentOverride] = useState<boolean | null>(null);
  const [name, setName] = useState('');
  const [savingName, setSavingName] = useState(false);

  const detail = useQuery({
    queryKey: ['db-table', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(base);
      return res.data as Json;
    },
  });

  const perms = useQuery({
    queryKey: ['db-table-permissions', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/permissions`);
      return ((res.data as { permissions?: Permission[] }).permissions ?? []) as Permission[];
    },
  });

  const table = detail.data;
  const apiName = table ? str(table['name']) || tableName : tableName;

  // Seed the name field once the table object arrives.
  useEffect(() => {
    if (table) setName(str(table['name']) || tableName);
  }, [table, tableName]);

  const enabled = enabledOverride ?? (table?.['enabled'] as boolean | undefined) ?? true;
  const rowSecurity =
    rowSecurityOverride ?? (table?.['rowSecurity'] as boolean | undefined) ?? false;
  const contentEnabled =
    contentOverride ?? (table?.['contentEnabled'] as boolean | undefined) ?? false;

  const invalidateDetail = () => qc.invalidateQueries({ queryKey: ['db-table', dbId, tableId] });
  const invalidatePerms = () =>
    qc.invalidateQueries({ queryKey: ['db-table-permissions', dbId, tableId] });

  const toggleEnabled = async (v: boolean) => {
    setEnabledOverride(v);
    try {
      await api.put(base, { enabled: v });
      invalidateDetail();
    } catch (e) {
      setEnabledOverride(!v);
      toast.error(friendlyError(e));
    }
  };

  const toggleRowSecurity = async (v: boolean) => {
    setRowSecurityOverride(v);
    try {
      await api.put(base, { rowSecurity: v });
      invalidateDetail();
    } catch (e) {
      setRowSecurityOverride(!v);
      toast.error(friendlyError(e));
    }
  };

  const toggleContentMode = async (v: boolean) => {
    setContentOverride(v);
    try {
      if (v) await api.post(`${base}/content`);
      else await api.delete(`${base}/content`);
      invalidateDetail();
    } catch (e) {
      setContentOverride(!v);
      toast.error(friendlyError(e));
    }
  };

  const saveName = async () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    setSavingName(true);
    try {
      await api.put(base, { name: trimmed });
      qc.invalidateQueries({ queryKey: ['db-tables', dbId] });
      invalidateDetail();
      toast.success('Table name updated');
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSavingName(false);
    }
  };

  const savePermissions = async (updated: Permission[]) => {
    await api.post(`${base}/permissions`, {
      permissions: updated.map((p) => ({ role: p.role, action: p.action })),
    });
    invalidatePerms();
  };

  const removePermission = async (p: Permission) => {
    const current = perms.data ?? [];
    const updated = current.filter((x) => x.role !== p.role || x.action !== p.action);
    try {
      await savePermissions(updated);
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(base);
      qc.invalidateQueries({ queryKey: ['db-tables', dbId] });
      setConfirming(false);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  if (detail.error) {
    return <ErrorState error={detail.error} onRetry={detail.refetch} />;
  }

  return (
    <div className="flex max-w-2xl flex-col gap-4">
      {/* 1. Status + enabled toggle */}
      <SectionCard>
        <div className="flex items-center gap-4">
          <div className="min-w-0 flex-1">
            <div className="truncate text-[length:var(--text-subhead)] font-semibold text-text-primary">
              {apiName}
            </div>
            <div className="mt-1 flex items-center gap-2 text-[length:var(--text-label)] text-text-subtle">
              <span>Table ID</span>
              <IdText id={tableId} />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={toggleEnabled} disabled={!table} />
            <span className="text-[length:var(--text-body)] text-text-primary">Enabled</span>
          </div>
        </div>
      </SectionCard>

      {/* 2. Name */}
      <SectionCard title="Name">
        <div className="flex items-end gap-3">
          <div className="w-20 pb-2 text-[length:var(--text-label)] font-medium text-text-muted">
            Name
          </div>
          <Input
            className="flex-1"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={!table}
          />
          <Button onClick={saveName} loading={savingName} disabled={!name.trim() || !table}>
            Update
          </Button>
        </div>
      </SectionCard>

      {/* 3. Permissions */}
      <SectionCard title="Permissions" subtitle="Choose who can access your tables and rows.">
        {perms.error ? (
          <ErrorState error={perms.error} onRetry={perms.refetch} />
        ) : perms.isLoading ? (
          <div className="py-6 text-center text-[length:var(--text-body)] text-text-subtle">
            Loading…
          </div>
        ) : (
          <PermissionsPanel
            perms={perms.data ?? []}
            onRemove={removePermission}
            onAdd={() => setAddingPerm(true)}
          />
        )}
      </SectionCard>

      {/* 4. Row security */}
      <SectionCard title="Row security">
        <div className="flex items-center gap-2">
          <Switch checked={rowSecurity} onCheckedChange={toggleRowSecurity} disabled={!table} />
          <span className="text-[length:var(--text-body)] text-text-primary">Row security</span>
        </div>
        <p className="mt-3 whitespace-pre-line text-[length:var(--text-label)] leading-relaxed text-text-subtle">
          {
            'When row security is enabled, users will be able to access rows for which they have been granted either row or table permissions.\n\nIf row security is disabled, users can access rows only if they have table permissions. Row permissions will be ignored.'
          }
        </p>
      </SectionCard>

      {/* 5. Content mode */}
      <SectionCard title="Content mode">
        <div className="flex items-center gap-2">
          <Switch checked={contentEnabled} onCheckedChange={toggleContentMode} disabled={!table} />
          <span className="text-[length:var(--text-body)] text-text-primary">Content mode</span>
        </div>
        <p className="mt-3 whitespace-pre-line text-[length:var(--text-label)] leading-relaxed text-text-subtle">
          {
            'Turn this table into an editorial collection. Rows gain a draft/published workflow, a slug, a locale and version history, and an Entries tab for editing them.\n\nIt is the same table and the same API. Turning it off only hides the editorial tools, nothing is deleted.'
          }
        </p>
      </SectionCard>

      {/* 6. Delete table */}
      <div className="rounded-[var(--radius-10)] border border-[color-mix(in_srgb,var(--color-danger)_40%,var(--border))] bg-surface p-5">
        <div className="text-[length:var(--text-control)] font-medium text-[var(--status-danger)]">
          Delete table
        </div>
        <div className="mt-1 text-[length:var(--text-body)] text-text-muted">
          The table will be permanently deleted, including all the rows within it. This action is
          irreversible.
        </div>
        <Button variant="destructive" className="mt-3" onClick={() => setConfirming(true)}>
          Delete table
        </Button>
      </div>

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Delete table"
        message="This permanently deletes the table and all its rows. This cannot be undone."
        confirmLabel="Delete"
        loading={deleting}
        onConfirm={del}
      />

      {addingPerm && (
        <AddPermissionDialog
          existing={perms.data ?? []}
          onClose={() => setAddingPerm(false)}
          onSave={savePermissions}
        />
      )}
    </div>
  );
}

function PermissionsPanel({
  perms,
  onRemove,
  onAdd,
}: {
  perms: Permission[];
  onRemove: (p: Permission) => void;
  onAdd: () => void;
}) {
  return (
    <div className="flex flex-col items-start gap-3">
      {perms.length === 0 ? (
        <div className="flex w-full flex-col items-center gap-2 rounded-[var(--radius)] border border-border bg-fill p-4">
          <ShieldOff size={20} className="text-text-subtle" />
          <span className="text-[length:var(--text-body)] text-text-muted">
            No permissions set. All access is denied.
          </span>
        </div>
      ) : (
        <div className="w-full overflow-hidden rounded-[var(--radius)] border border-border">
          {perms.map((p, i) => (
            <div
              key={`${p.role}:${p.action}`}
              className={`flex items-center gap-2 px-3.5 py-2.5 ${
                i < perms.length - 1 ? 'border-b border-border' : ''
              }`}
            >
              <span className="min-w-0 flex-1 truncate text-[length:var(--text-body)] text-text-primary">
                {p.role}
              </span>
              <span className="rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--color-accent)_12%,transparent)] px-2 py-0.5 text-[length:var(--text-caption)] font-medium text-[var(--color-accent)]">
                {p.action}
              </span>
              <button
                type="button"
                onClick={() => onRemove(p)}
                className="text-text-subtle transition-colors hover:text-text-primary"
                aria-label="Remove permission"
              >
                <X size={14} />
              </button>
            </div>
          ))}
        </div>
      )}
      <Button variant="ghost" size="sm" onClick={onAdd} className="px-0 text-[var(--color-accent)]">
        <Plus size={14} />
        Add permission
      </Button>
    </div>
  );
}

function AddPermissionDialog({
  existing,
  onClose,
  onSave,
}: {
  existing: Permission[];
  onClose: () => void;
  onSave: (updated: Permission[]) => Promise<void>;
}) {
  const [role, setRole] = useState('');
  const [action, setAction] = useState('read');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    const trimmed = role.trim();
    if (!trimmed) return;
    setSaving(true);
    try {
      await onSave([...existing, { role: trimmed, action }]);
      onClose();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Add permission"
      submitLabel="Add"
      loading={saving}
      submitDisabled={!role.trim()}
      onSubmit={submit}
    >
      <TextField
        label="Role"
        placeholder="e.g. users, any, user:123"
        value={role}
        onChange={(e) => setRole(e.target.value)}
        autoFocus
      />
      <FormField label="Action">
        <ChipGroup options={PERM_ACTIONS} value={action} onChange={setAction} />
      </FormField>
    </FormDialog>
  );
}
