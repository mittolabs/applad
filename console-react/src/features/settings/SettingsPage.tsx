import { type ReactNode, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  ClipboardList,
  Database,
  Eye,
  EyeOff,
  FolderClosed,
  GitBranch,
  Globe,
  Info,
  Key,
  Layers,
  type LucideIcon,
  Loader2,
  MessageSquare,
  Monitor,
  Plus,
  Radio,
  Smartphone,
  ToggleRight,
  Trash2,
  Users,
  Webhook,
  Zap,
} from 'lucide-react';
import { api } from '@/api/client';
import { useProject } from '@/api/queries';
import { useResourceList } from '@/hooks/use-resource-list';
import { useTabIndex } from '@/hooks/use-tab-param';
import { PageTabs } from '@/components/page-tabs';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { SearchListHeader } from '@/components/search-list';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { FormDialog, ConfirmDialog, TextField, SelectField } from '@/components/form-dialog';
import { cn } from '@/lib/utils';
import {
  CopyButton,
  EXPIRY_OPTIONS,
  ScopeGroups,
  SCOPE_GROUPS,
  expiresAtIso,
  expiryPreview,
} from './scopes';

/* Ports console/lib/features/settings/settings_page.dart.
 * Tabs: General, API Keys, Webhooks, Audit Log (persisted via ?tab=). */

const TAB_SLUGS = ['general', 'api-keys', 'webhooks', 'audit-log'];
const TAB_LABELS = ['General', 'API Keys', 'Webhooks', 'Audit Log'];

export function SettingsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useTabIndex(TAB_SLUGS);

  if (!projectId) {
    return (
      <div className="flex flex-col gap-6 p-6 md:p-8">
        <div className="text-[length:var(--text-body)] text-text-secondary">
          No project selected
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          Settings
        </h1>
        <p className="mt-1 text-[length:var(--text-control)] text-text-muted">
          Manage your project configuration
        </p>
      </div>
      <PageTabs tabs={TAB_LABELS} selected={tab} onChange={setTab} />
      {tab === 0 && <GeneralTab projectId={projectId} />}
      {tab === 1 && <ApiKeysTab projectId={projectId} />}
      {tab === 2 && <WebhooksTab projectId={projectId} />}
      {tab === 3 && <AuditLogTab projectId={projectId} />}
    </div>
  );
}

// ===========================================================================
// General tab
// ===========================================================================

const CORE_SERVICES: { label: string; description: string; icon: LucideIcon; enabled: boolean }[] = [
  { label: 'Auth', description: 'User authentication and management', icon: Users, enabled: true },
  { label: 'Databases', description: 'Structured data storage', icon: Database, enabled: true },
  { label: 'Storage', description: 'File storage and management', icon: FolderClosed, enabled: true },
  { label: 'Functions', description: 'Serverless function execution', icon: Zap, enabled: true },
  { label: 'Messaging', description: 'Email, SMS, and push notifications', icon: MessageSquare, enabled: true },
  { label: 'Workflows', description: 'DAG workflow engine and automation', icon: GitBranch, enabled: true },
  { label: 'Realtime', description: 'WebSocket pub/sub subscriptions', icon: Radio, enabled: true },
  { label: 'Sites', description: 'Static site hosting with custom domains', icon: Globe, enabled: true },
  { label: 'Containers', description: 'Docker container deployments', icon: Box, enabled: true },
];

const EXPERIMENTAL_SERVICES: { label: string; description: string; icon: LucideIcon; enabled: boolean }[] = [
  { label: 'Mobile', description: 'Mobile app builds and distribution', icon: Smartphone, enabled: false },
  { label: 'Desktop', description: 'Desktop app builds and distribution', icon: Monitor, enabled: false },
  { label: 'Feature Flags', description: 'Feature flags and remote config', icon: ToggleRight, enabled: false },
  { label: 'Environments', description: 'Environment variables and staging', icon: Layers, enabled: false },
];

function GeneralTab({ projectId }: { projectId: string }) {
  const { data, isLoading, error, refetch } = useProject(projectId);
  const qc = useQueryClient();
  const navigate = useNavigate();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [dirty, setDirty] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [confirmText, setConfirmText] = useState('');

  useEffect(() => {
    if (data && !dirty) {
      setName(String(data.name ?? ''));
      setDescription(String((data as Record<string, unknown>).description ?? ''));
    }
  }, [data, dirty]);

  const save = useMutation({
    mutationFn: () =>
      api.patch(`/projects/${projectId}`, { name: name.trim(), description: description.trim() }),
    onSuccess: () => {
      setDirty(false);
      qc.invalidateQueries({ queryKey: ['project', projectId] });
      qc.invalidateQueries({ queryKey: ['projects'] });
    },
  });

  const del = useMutation({
    mutationFn: () => api.delete(`/projects/${projectId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] });
      navigate('/projects');
    },
  });

  if (isLoading) return <LoadingBlock />;
  if (error) return <ErrorState error={error} onRetry={refetch} />;

  const markDirty = () => {
    if (!dirty) setDirty(true);
  };

  return (
    <div className="flex max-w-4xl flex-col gap-5">
      <SettingsCard
        title="Project details"
        subtitle="Update your project name and description"
        trailing={
          dirty ? (
            <Button size="sm" loading={save.isPending} onClick={() => save.mutate()}>
              Save
            </Button>
          ) : undefined
        }
      >
        <div className="flex flex-col gap-4">
          <SettingsField label="Project name">
            <Input
              value={name}
              placeholder="My project"
              onChange={(e) => {
                setName(e.target.value);
                markDirty();
              }}
            />
          </SettingsField>
          <SettingsField label="Description">
            <Input
              value={description}
              placeholder="Optional description"
              onChange={(e) => {
                setDescription(e.target.value);
                markDirty();
              }}
            />
          </SettingsField>
          <SettingsField label="Project ID">
            <ReadOnlyValue value={projectId} />
          </SettingsField>
        </div>
      </SettingsCard>

      <SettingsCard
        title="Services"
        subtitle="Enable or disable individual services for this project"
      >
        <div className="flex flex-col">
          {CORE_SERVICES.map((s) => (
            <ServiceToggle key={s.label} {...s} />
          ))}
          <div className="pb-1 pt-3 text-[length:var(--text-caption)] font-semibold uppercase tracking-wider text-text-subtle">
            Experimental
          </div>
          {EXPERIMENTAL_SERVICES.map((s) => (
            <ServiceToggle key={s.label} {...s} />
          ))}
        </div>
      </SettingsCard>

      <SettingsCard
        title="Danger zone"
        subtitle="Irreversible actions. Please proceed with caution."
        danger
      >
        <div className="flex items-center gap-4">
          <div className="flex-1">
            <div className="text-[length:var(--text-control)] font-medium text-text-primary">
              Delete project
            </div>
            <div className="mt-1 text-[length:var(--text-body)] text-text-muted">
              Permanently delete this project and all its data. This action cannot be undone.
            </div>
          </div>
          <Button
            variant="outline"
            className="border-[var(--color-danger)] text-[var(--status-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)]"
            onClick={() => {
              setConfirmText('');
              setConfirmDelete(true);
            }}
          >
            Delete project
          </Button>
        </div>
      </SettingsCard>

      <FormDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete project"
        subtitle="This will permanently delete the project and all its data including databases, storage, functions, and deployments."
        submitLabel="Delete"
        destructive
        loading={del.isPending}
        submitDisabled={confirmText !== projectId}
        onSubmit={() => del.mutate()}
      >
        <TextField
          label="Type the project ID to confirm"
          value={confirmText}
          placeholder={projectId}
          onChange={(e) => setConfirmText(e.target.value)}
          autoFocus
        />
      </FormDialog>
    </div>
  );
}

function ServiceToggle({
  label,
  description,
  icon: Icon,
  enabled,
}: {
  label: string;
  description: string;
  icon: LucideIcon;
  enabled: boolean;
}) {
  const [on, setOn] = useState(enabled);
  return (
    <div className="flex items-center gap-3 py-1.5">
      <Icon size={16} className="text-text-secondary" />
      <div className="flex-1">
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">{label}</div>
        <div className="text-[length:var(--text-label)] text-text-muted">{description}</div>
      </div>
      <Switch checked={on} onCheckedChange={setOn} />
    </div>
  );
}

// ===========================================================================
// API Keys tab
// ===========================================================================

function scopeSummary(row: Row): string {
  const scopes = (row.scopes as string[] | undefined) ?? [];
  return scopes.length === 0
    ? 'All scopes'
    : `${scopes.length} scope${scopes.length === 1 ? '' : 's'}`;
}

function shortDate(iso: string | undefined | null): string {
  if (!iso) return '';
  const t = new Date(iso);
  if (Number.isNaN(t.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${t.getFullYear()}-${p(t.getMonth() + 1)}-${p(t.getDate())}`;
}

function isExpired(iso: string): boolean {
  const t = new Date(iso);
  return !Number.isNaN(t.getTime()) && t.getTime() < Date.now();
}

const API_KEY_COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'secretPrefix', label: 'Secret', flex: 3 },
  { key: 'scopes', label: 'Scopes', flex: 2 },
  { key: 'expire', label: 'Expires', flex: 2 },
  { key: '$createdAt', label: 'Created', flex: 2 },
];

function ApiKeysTab({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const list = useResourceList({
    endpoint: `/projects/${projectId}/keys`,
    itemsKey: 'keys',
    scope: [projectId],
    defaultPerPage: 25,
  });
  const [creating, setCreating] = useState(false);
  const [secret, setSecret] = useState<string | null>(null);

  return (
    <>
      <DataTable
        columns={API_KEY_COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row.name ?? '');
            case 'secretPrefix':
              return String(row.secretPrefix ?? '');
            case 'scopes':
              return scopeSummary(row);
            case 'expire':
              return row.expire ? shortDate(String(row.expire)) : 'Never';
            case '$createdAt':
              return shortDate(String(row.$createdAt ?? row.createdAt ?? ''));
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'secretPrefix') {
            const prefix = String(row.secretPrefix ?? '');
            return prefix ? (
              <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-subtle">
                {prefix}···
              </span>
            ) : (
              <span className="text-text-subtle">—</span>
            );
          }
          if (key === 'expire') {
            const iso = row.expire ? String(row.expire) : '';
            if (!iso) {
              return <span className="text-[length:var(--text-label)] text-text-subtle">Never</span>;
            }
            const expired = isExpired(iso);
            const color = expired ? 'var(--status-danger)' : 'var(--status-success)';
            return (
              <span
                className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium"
                style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
              >
                {shortDate(iso)}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => Key}
        onRowClick={(row) =>
          navigate(`/project/${projectId}/settings/keys/${String(row.$id)}`)
        }
        onDeleteRow={async (row) => {
          await api.delete(`/projects/${projectId}/keys/${String(row.$id)}`);
          list.refetch();
        }}
        deleteTitle="Delete API key"
        deleteMessage="Any applications using this key will lose access immediately."
        createLabel="Create API key"
        onCreate={() => setCreating(true)}
        searchHint="Search by name"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="API keys"
        emptyIcon={Key}
        emptyTitle="No API keys"
        emptySubtitle="Create an API key to authenticate server-side requests"
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <CreateKeyDialog
        projectId={projectId}
        open={creating}
        onOpenChange={setCreating}
        onCreated={(s) => {
          list.refetch();
          setSecret(s);
        }}
      />

      <SecretDialog secret={secret} onClose={() => setSecret(null)} />
    </>
  );
}

function CreateKeyDialog({
  projectId,
  open,
  onOpenChange,
  onCreated,
}: {
  projectId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (secret: string) => void;
}) {
  const [name, setName] = useState('');
  const [expiry, setExpiry] = useState('never');
  const [customDate, setCustomDate] = useState('');
  const [scopes, setScopes] = useState<Set<string>>(new Set());

  const reset = () => {
    setName('');
    setExpiry('never');
    setCustomDate('');
    setScopes(new Set());
  };

  const create = useMutation({
    mutationFn: async () => {
      const body: Record<string, unknown> = { name: name.trim(), scopes: [...scopes] };
      const iso = expiresAtIso(expiry, customDate);
      if (iso) body.expiresAt = iso;
      const res = await api.post(`/projects/${projectId}/keys`, body);
      return (res.data as { secret?: string }).secret ?? '';
    },
    onSuccess: (secret) => {
      onOpenChange(false);
      reset();
      onCreated(secret);
    },
  });

  const preview = expiryPreview(expiry, customDate);

  return (
    <FormDialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="Create API key"
      subtitle="API keys authenticate server-side SDK requests"
      width={520}
      submitLabel="Create"
      loading={create.isPending}
      submitDisabled={!name.trim()}
      onSubmit={() => create.mutate()}
    >
      <TextField
        label="Name"
        value={name}
        placeholder="e.g. Production key"
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />

      <div className="flex flex-col gap-1.5">
        <SelectField
          label="Expiration"
          value={expiry}
          onChange={setExpiry}
          options={EXPIRY_OPTIONS}
        />
        {expiry === 'custom' && (
          <Input type="date" value={customDate} onChange={(e) => setCustomDate(e.target.value)} />
        )}
        {preview && (
          <div className="flex items-center gap-1.5 text-[length:var(--text-label)] text-text-subtle">
            <Info size={12} />
            {preview}
          </div>
        )}
      </div>

      <ScopeSelectorField scopes={scopes} setScopes={setScopes} />
    </FormDialog>
  );
}

/** Scopes header (select/deselect all) + grouped selector, shared shape. */
function ScopeSelectorField({
  scopes,
  setScopes,
}: {
  scopes: Set<string>;
  setScopes: (updater: (prev: Set<string>) => Set<string>) => void;
}) {
  const toggleScope = (scope: string) =>
    setScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) next.delete(scope);
      else next.add(scope);
      return next;
    });
  const toggleGroup = (group: string) =>
    setScopes((prev) => {
      const next = new Set(prev);
      const groupScopes = SCOPE_GROUPS[group];
      const all = groupScopes.every((s) => next.has(s));
      groupScopes.forEach((s) => (all ? next.delete(s) : next.add(s)));
      return next;
    });
  const selectAll = () => setScopes(() => new Set(Object.values(SCOPE_GROUPS).flat()));
  const deselectAll = () => setScopes(() => new Set());

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center">
        <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
          Scopes
        </span>
        <button
          type="button"
          onClick={selectAll}
          className="ml-auto text-[length:var(--text-label)] text-[var(--color-accent)]"
        >
          Select all
        </button>
        <button
          type="button"
          onClick={deselectAll}
          className="ml-3 text-[length:var(--text-label)] text-text-subtle hover:text-text-secondary"
        >
          Deselect all
        </button>
      </div>
      <p className="text-[length:var(--text-label)] text-text-subtle">
        Grant only the permissions your application needs.
      </p>
      <ScopeGroups selected={scopes} onToggleScope={toggleScope} onToggleGroup={toggleGroup} />
    </div>
  );
}

function SecretDialog({ secret, onClose }: { secret: string | null; onClose: () => void }) {
  const [visible, setVisible] = useState(true);
  return (
    <Dialog open={secret !== null} onOpenChange={(o) => !o && onClose()}>
      <DialogContent width={480}>
        <DialogHeader>
          <DialogTitle>Copy your API key</DialogTitle>
          <DialogDescription>
            This is the only time your key will be shown. Copy it now.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-fill px-3 py-2.5">
            <span className="flex-1 break-all font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
              {visible ? secret : '•'.repeat(48)}
            </span>
            <button
              type="button"
              onClick={() => setVisible((v) => !v)}
              className="text-text-subtle transition-colors hover:text-text-primary"
              aria-label={visible ? 'Hide secret' : 'Show secret'}
            >
              {visible ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
            <CopyButton text={secret ?? ''} />
          </div>
        </DialogBody>
        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===========================================================================
// Webhooks tab
// ===========================================================================

const WEBHOOK_EVENTS = [
  'databases.*',
  'storage.*',
  'users.*',
  'teams.*',
  'functions.*',
  'messaging.*',
  'deploy.*',
  'workflows.*',
  'credentials.*',
];

function WebhooksTab({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['webhooks', projectId],
    queryFn: async () => {
      const res = await api.get('/webhooks', { params: { projectId } });
      return ((res.data as { webhooks?: Row[] }).webhooks ?? []) as Row[];
    },
  });

  const [search, setSearch] = useState('');
  const [creating, setCreating] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<Row | null>(null);

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`/webhooks/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['webhooks', projectId] });
      setPendingDelete(null);
    },
  });

  if (isLoading) return <LoadingBlock />;
  if (error) return <ErrorState error={error} onRetry={refetch} />;

  const rows = data ?? [];
  const query = search.trim().toLowerCase();
  const filtered = query
    ? rows.filter(
        (w) =>
          String(w.name ?? '').toLowerCase().includes(query) ||
          String(w.url ?? '').toLowerCase().includes(query),
      )
    : rows;

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        searchHint="Search webhooks…"
        value={search}
        onChange={setSearch}
        trailing={
          <div className="flex items-center gap-3">
            <span className="text-[length:var(--text-body)] text-text-secondary">
              {rows.length} webhook{rows.length === 1 ? '' : 's'}
            </span>
            <Button size="sm" onClick={() => setCreating(true)}>
              <Plus size={14} />
              Create webhook
            </Button>
          </div>
        }
      />

      {filtered.length === 0 ? (
        <EmptyState
          icon={Webhook}
          title="No webhooks configured"
          subtitle="Webhooks send real-time notifications to your server when events occur"
          actionLabel="Create webhook"
          onAction={() => setCreating(true)}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {filtered.map((w) => (
            <WebhookCard key={String(w.$id)} webhook={w} onDelete={() => setPendingDelete(w)} />
          ))}
        </div>
      )}

      <CreateWebhookDialog open={creating} onOpenChange={setCreating} onCreated={refetch} />

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title="Delete webhook"
        message="This webhook will stop receiving notifications."
        loading={del.isPending}
        onConfirm={() => pendingDelete && del.mutate(String(pendingDelete.$id))}
      />
    </div>
  );
}

function WebhookCard({ webhook, onDelete }: { webhook: Row; onDelete: () => void }) {
  const name = String(webhook.name ?? 'Unnamed');
  const url = String(webhook.url ?? '');
  const enabled = webhook.enabled !== false;
  const events = (webhook.events as string[] | undefined) ?? [];
  const activeColor = enabled ? 'var(--status-success)' : 'var(--status-neutral)';

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-4">
      <div className="flex items-start gap-3.5">
        <div className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)] bg-[color-mix(in_srgb,var(--color-accent)_10%,transparent)]">
          <Webhook size={18} className="text-[var(--color-accent)]" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-[length:var(--text-control)] font-medium text-text-primary">
              {name}
            </span>
            <span
              className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium"
              style={{
                color: activeColor,
                backgroundColor: `color-mix(in srgb, ${activeColor} 15%, transparent)`,
              }}
            >
              {enabled ? 'Active' : 'Disabled'}
            </span>
          </div>
          <div className="mt-0.5 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-secondary">
            {url}
          </div>
          {events.length > 0 && (
            <div className="mt-2.5 flex flex-wrap gap-1.5">
              {events.map((e) => (
                <span
                  key={e}
                  className="rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary"
                >
                  {e}
                </span>
              ))}
            </div>
          )}
        </div>
        <button
          type="button"
          onClick={onDelete}
          className="text-text-subtle transition-colors hover:text-[var(--status-danger)]"
          aria-label="Delete webhook"
        >
          <Trash2 size={14} />
        </button>
      </div>
    </div>
  );
}

function CreateWebhookDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [events, setEvents] = useState<Set<string>>(new Set());

  const reset = () => {
    setName('');
    setUrl('');
    setEvents(new Set());
  };

  const create = useMutation({
    mutationFn: () =>
      api.post('/webhooks', {
        name: name.trim(),
        url: url.trim(),
        events: [...events],
        enabled: true,
      }),
    onSuccess: () => {
      onOpenChange(false);
      reset();
      onCreated();
    },
  });

  const toggleEvent = (e: string) =>
    setEvents((prev) => {
      const next = new Set(prev);
      if (next.has(e)) next.delete(e);
      else next.add(e);
      return next;
    });

  return (
    <FormDialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="Create webhook"
      subtitle="Receive notifications when events occur in your project"
      width={520}
      submitLabel="Create"
      loading={create.isPending}
      submitDisabled={!name.trim() || !url.trim()}
      onSubmit={() => create.mutate()}
    >
      <TextField
        label="Name"
        value={name}
        placeholder="My webhook"
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <TextField
        label="POST URL"
        value={url}
        placeholder="https://example.com/webhook"
        onChange={(e) => setUrl(e.target.value)}
      />
      <div className="flex flex-col gap-2">
        <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
          Events
        </span>
        <div className="flex flex-wrap gap-2">
          {WEBHOOK_EVENTS.map((e) => {
            const active = events.has(e);
            return (
              <button
                key={e}
                type="button"
                onClick={() => toggleEvent(e)}
                className={cn(
                  'rounded-[var(--radius-6)] border px-2.5 py-1 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] transition-colors',
                  active
                    ? 'border-[var(--color-accent)] bg-[color-mix(in_srgb,var(--color-accent)_25%,transparent)] text-text-primary'
                    : 'border-field-border text-text-secondary hover:text-text-primary',
                )}
              >
                {e}
              </button>
            );
          })}
        </div>
      </div>
    </FormDialog>
  );
}

// ===========================================================================
// Audit Log tab
// ===========================================================================

function fmtTs(iso: string): string {
  if (!iso) return '';
  const t = new Date(iso);
  if (Number.isNaN(t.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${t.getFullYear()}-${p(t.getMonth() + 1)}-${p(t.getDate())} ${p(t.getHours())}:${p(t.getMinutes())}`;
}

const AUDIT_COLUMNS: DataTableColumn[] = [
  { key: 'method', label: 'Method', flex: 1 },
  { key: 'path', label: 'Path', flex: 3 },
  { key: 'statusCode', label: 'Status', flex: 1 },
  { key: 'resourceType', label: 'Resource', flex: 2 },
  { key: 'action', label: 'Action', flex: 2, defaultVisible: false },
  { key: 'userId', label: 'User', flex: 2, defaultVisible: false },
  { key: 'ipAddress', label: 'IP', flex: 2, defaultVisible: false },
  { key: '$createdAt', label: 'Time', flex: 2 },
];

const AUDIT_FILTERS = [
  {
    key: 'method',
    label: 'Method',
    options: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((v) => ({ value: v, label: v })),
  },
  {
    key: 'resourceType',
    label: 'Resource type',
    options: [
      'user', 'session', 'team', 'database', 'table', 'row', 'bucket', 'file',
      'function', 'workflow', 'deployment', 'project', 'api_key',
    ].map((v) => ({ value: v, label: v })),
  },
];

function AuditLogTab({ projectId }: { projectId: string }) {
  const list = useResourceList({
    endpoint: '/audit',
    itemsKey: 'logs',
    scope: [projectId],
    defaultPerPage: 25,
  });

  return (
    <DataTable
      columns={AUDIT_COLUMNS}
      rows={list.rows}
      getCellValue={(row, key) => {
        switch (key) {
          case 'statusCode':
            return String(Number(row.statusCode ?? 0));
          case '$createdAt':
            return fmtTs(String(row.$createdAt ?? ''));
          default:
            return String(row[key] ?? '');
        }
      }}
      cellRender={(row, key) => {
        if (key === 'method') return <MethodChip method={String(row.method ?? '')} />;
        if (key === 'statusCode') return <StatusCodeChip status={Number(row.statusCode ?? 0)} />;
        return undefined;
      }}
      filters={AUDIT_FILTERS}
      filterValues={list.filters}
      onFiltersChange={list.setFilters}
      total={list.total}
      perPage={list.perPage}
      page={list.page}
      onPerPageChange={list.setPerPage}
      onPrev={list.prevPage}
      onNext={list.nextPage}
      itemLabel="entries"
      searchHint="Search not available server-side"
      searchValue=""
      onSearchChange={() => {}}
      emptyIcon={ClipboardList}
      emptyTitle="No audit log entries yet"
      emptySubtitle="API activity on this project will appear here"
      loading={list.isLoading}
      error={list.error}
      onRetry={list.refetch}
    />
  );
}

function MethodChip({ method }: { method: string }) {
  const m = method.toUpperCase();
  const color =
    m === 'GET'
      ? 'var(--color-accent)'
      : m === 'POST'
        ? 'var(--status-success)'
        : m === 'PUT' || m === 'PATCH'
          ? 'var(--status-warning)'
          : m === 'DELETE'
            ? 'var(--status-danger)'
            : 'var(--status-neutral)';
  return (
    <span
      className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-semibold"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)` }}
    >
      {m}
    </span>
  );
}

function StatusCodeChip({ status }: { status: number }) {
  const color =
    status >= 500
      ? 'var(--status-danger)'
      : status >= 400
        ? 'var(--status-warning)'
        : 'var(--status-success)';
  return (
    <span
      className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-semibold"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)` }}
    >
      {status}
    </span>
  );
}

// ===========================================================================
// Shared layout bits
// ===========================================================================

function SettingsCard({
  title,
  subtitle,
  trailing,
  danger,
  children,
}: {
  title: string;
  subtitle?: string;
  trailing?: ReactNode;
  danger?: boolean;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        'rounded-[var(--radius)] border bg-surface p-5',
        danger
          ? 'border-[color-mix(in_srgb,var(--color-danger)_30%,var(--border))]'
          : 'border-border',
      )}
    >
      <div className="flex items-start gap-4">
        <div className="flex-1">
          <div
            className={cn(
              'text-[length:var(--text-subhead)] font-semibold',
              danger ? 'text-[var(--status-danger)]' : 'text-text-primary',
            )}
          >
            {title}
          </div>
          {subtitle && (
            <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
              {subtitle}
            </div>
          )}
        </div>
        {trailing}
      </div>
      <div className="my-4 h-px bg-border" />
      {children}
    </div>
  );
}

function SettingsField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start gap-4">
      <div className="w-40 shrink-0 pt-2 text-[length:var(--text-body)] font-medium text-text-secondary">
        {label}
      </div>
      <div className="flex-1">{children}</div>
    </div>
  );
}

function ReadOnlyValue({ value }: { value: string }) {
  return (
    <div className="flex items-center gap-2 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2.5">
      <span className="flex-1 break-all font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-secondary">
        {value}
      </span>
      <CopyButton text={value} size={14} />
    </div>
  );
}

function LoadingBlock() {
  return (
    <div className="flex items-center justify-center py-16">
      <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
    </div>
  );
}
