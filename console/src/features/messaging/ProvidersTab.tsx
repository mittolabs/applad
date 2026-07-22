import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Bell, Mail, MessageSquare, Pencil, Plus, Trash2, type LucideIcon } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  ConfirmDialog,
  FormDialog,
  SelectField,
  TextField,
} from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';

/* Real messaging-provider CRUD backed by the messaging service:
 *   GET/POST         /messaging/providers
 *   GET/PUT/DELETE   /messaging/providers/{id}
 * A provider is { id, name, type (email|sms|push), provider (smtp|…),
 * config (arbitrary JSON), enabled }. The catalog below drives the config
 * fields for each known provider; config is stored verbatim by the backend. */

type Category = 'email' | 'sms' | 'push';

interface CatalogField {
  key: string;
  label: string;
  secret?: boolean;
  placeholder?: string;
}

interface CatalogEntry {
  provider: string;
  type: Category;
  label: string;
  fields: CatalogField[];
}

const CATALOG: CatalogEntry[] = [
  {
    provider: 'smtp',
    type: 'email',
    label: 'SMTP',
    fields: [
      { key: 'host', label: 'Host', placeholder: 'smtp.example.com' },
      { key: 'port', label: 'Port', placeholder: '587' },
      { key: 'username', label: 'Username' },
      { key: 'password', label: 'Password', secret: true },
      { key: 'from', label: 'From address', placeholder: 'no-reply@example.com' },
    ],
  },
  {
    provider: 'mailgun',
    type: 'email',
    label: 'Mailgun',
    fields: [
      { key: 'apiKey', label: 'API key', secret: true },
      { key: 'domain', label: 'Domain', placeholder: 'mg.example.com' },
    ],
  },
  {
    provider: 'resend',
    type: 'email',
    label: 'Resend',
    fields: [{ key: 'apiKey', label: 'API key', secret: true }],
  },
  {
    provider: 'sendgrid',
    type: 'email',
    label: 'SendGrid',
    fields: [{ key: 'apiKey', label: 'API key', secret: true }],
  },
  {
    provider: 'twilio',
    type: 'sms',
    label: 'Twilio',
    fields: [
      { key: 'accountSid', label: 'Account SID' },
      { key: 'authToken', label: 'Auth token', secret: true },
      { key: 'from', label: 'From number', placeholder: '+15551234567' },
    ],
  },
  {
    provider: 'vonage',
    type: 'sms',
    label: 'Vonage (Nexmo)',
    fields: [
      { key: 'apiKey', label: 'API key' },
      { key: 'apiSecret', label: 'API secret', secret: true },
      { key: 'from', label: 'From' },
    ],
  },
  {
    provider: 'msg91',
    type: 'sms',
    label: 'MSG91',
    fields: [
      { key: 'authKey', label: 'Auth key', secret: true },
      { key: 'senderId', label: 'Sender ID' },
    ],
  },
  {
    provider: 'fcm',
    type: 'push',
    label: 'Firebase Cloud Messaging (FCM)',
    fields: [{ key: 'serverKey', label: 'Server key', secret: true }],
  },
  {
    provider: 'apns',
    type: 'push',
    label: 'Apple Push (APNS)',
    fields: [
      { key: 'keyId', label: 'Key ID' },
      { key: 'teamId', label: 'Team ID' },
      { key: 'keyPath', label: 'Key path' },
      { key: 'bundleId', label: 'Bundle ID' },
    ],
  },
];

const CATEGORY_LABEL: Record<Category, string> = {
  email: 'Email',
  sms: 'SMS',
  push: 'Push',
};

const CATEGORY_ICON: Record<Category, LucideIcon> = {
  email: Mail,
  sms: MessageSquare,
  push: Bell,
};

interface Provider {
  id: string;
  name: string;
  type: Category;
  provider: string;
  config?: Record<string, unknown> | null;
  enabled: boolean;
}

function catalogFor(provider: string): CatalogEntry | undefined {
  return CATALOG.find((c) => c.provider === provider);
}

export function ProvidersTab() {
  const qc = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<Provider | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Provider | null>(null);

  const query = useQuery({
    queryKey: ['/messaging/providers'],
    queryFn: async () => {
      const res = await api.get('/messaging/providers');
      return (res.data as { providers?: Provider[] }).providers ?? [];
    },
  });

  const del = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/messaging/providers/${id}`);
    },
    onSuccess: () => {
      setPendingDelete(null);
      void qc.invalidateQueries({ queryKey: ['/messaging/providers'] });
    },
  });

  const toggle = useMutation({
    mutationFn: async (p: Provider) => {
      await api.put(`/messaging/providers/${p.id}`, {
        name: p.name,
        config: p.config ?? {},
        enabled: !p.enabled,
      });
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['/messaging/providers'] }),
  });

  const providers = query.data ?? [];
  const categories: Category[] = ['email', 'sms', 'push'];

  return (
    <div className="flex flex-col gap-6 pb-8">
      <div className="flex items-center">
        <span className="text-[length:var(--text-body)] text-text-secondary">
          {providers.length} provider{providers.length === 1 ? '' : 's'} configured
        </span>
        <span className="flex-1" />
        <Button size="sm" onClick={() => setAdding(true)}>
          <Plus size={14} />
          Add provider
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : providers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-[var(--radius)] border border-border bg-surface px-6 py-16 text-center">
          <div className="text-[length:var(--text-subhead)] font-medium text-text-primary">
            No providers configured
          </div>
          <div className="mt-2 max-w-md text-[length:var(--text-body)] text-text-secondary">
            Add an email, SMS, or push provider so this project can send messages
            through its own account instead of the global defaults.
          </div>
          <Button variant="outline" className="mt-5" onClick={() => setAdding(true)}>
            Add provider
          </Button>
        </div>
      ) : (
        categories
          .filter((cat) => providers.some((p) => p.type === cat))
          .map((cat) => {
            const Icon = CATEGORY_ICON[cat];
            return (
              <div key={cat} className="flex flex-col gap-2">
                <div className="text-[length:var(--text-caption)] font-semibold uppercase tracking-[0.6px] text-text-subtle">
                  {CATEGORY_LABEL[cat]}
                </div>
                {providers
                  .filter((p) => p.type === cat)
                  .map((p) => {
                    const entry = catalogFor(p.provider);
                    return (
                      <div
                        key={p.id}
                        className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface p-4"
                      >
                        <div className="flex h-10 w-10 items-center justify-center rounded-[var(--radius)] bg-fill">
                          <Icon size={18} className="text-text-muted" />
                        </div>
                        <div className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate text-[length:var(--text-control)] font-medium text-text-primary">
                            {p.name}
                          </span>
                          <span className="text-[length:var(--text-label)] text-text-subtle">
                            {entry?.label ?? p.provider}
                          </span>
                        </div>
                        <span
                          className="rounded-full px-2 py-[3px] text-[length:var(--text-caption)] font-medium"
                          style={{
                            color: p.enabled
                              ? 'var(--color-status-success)'
                              : 'var(--text-subtle)',
                            backgroundColor: p.enabled
                              ? 'color-mix(in srgb, var(--color-status-success) 12%, transparent)'
                              : 'var(--fill)',
                          }}
                        >
                          {p.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <Switch
                          checked={p.enabled}
                          onCheckedChange={() => toggle.mutate(p)}
                          aria-label="Toggle provider"
                        />
                        <button
                          type="button"
                          onClick={() => setEditing(p)}
                          className="rounded-[var(--radius-6)] p-1.5 text-text-subtle transition-all hover:bg-fill hover:text-text-primary"
                          aria-label="Edit provider"
                        >
                          <Pencil size={14} />
                        </button>
                        <button
                          type="button"
                          onClick={() => setPendingDelete(p)}
                          className="rounded-[var(--radius-6)] p-1.5 text-text-subtle transition-all hover:bg-fill hover:text-[var(--color-danger)]"
                          aria-label="Delete provider"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    );
                  })}
              </div>
            );
          })
      )}

      {adding && (
        <ProviderDialog
          onClose={() => setAdding(false)}
          onSaved={() => {
            setAdding(false);
            void qc.invalidateQueries({ queryKey: ['/messaging/providers'] });
          }}
        />
      )}
      {editing && (
        <ProviderDialog
          provider={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void qc.invalidateQueries({ queryKey: ['/messaging/providers'] });
          }}
        />
      )}

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title="Delete provider"
        message={`Remove "${pendingDelete?.name ?? ''}"? Messages of this type will fall back to the project defaults.`}
        loading={del.isPending}
        onConfirm={() => pendingDelete && del.mutate(pendingDelete.id)}
      />
    </div>
  );
}

function ProviderDialog({
  provider,
  onClose,
  onSaved,
}: {
  provider?: Provider;
  onClose: () => void;
  onSaved: () => void;
}) {
  const editMode = !!provider;
  const [selected, setSelected] = useState<string>(
    provider?.provider ?? CATALOG[0].provider,
  );
  const entry = catalogFor(selected) ?? CATALOG[0];

  const [name, setName] = useState(provider?.name ?? entry.label);
  const [config, setConfig] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    const cfg = provider?.config ?? {};
    for (const f of (catalogFor(provider?.provider ?? selected) ?? entry).fields) {
      initial[f.key] = cfg[f.key] != null ? String(cfg[f.key]) : '';
    }
    return initial;
  });

  const save = useMutation({
    mutationFn: async () => {
      const cfgPayload: Record<string, string> = {};
      for (const f of entry.fields) {
        const v = config[f.key]?.trim() ?? '';
        if (v) cfgPayload[f.key] = v;
      }
      if (editMode && provider) {
        await api.put(`/messaging/providers/${provider.id}`, {
          name: name.trim(),
          config: cfgPayload,
          enabled: provider.enabled,
        });
      } else {
        await api.post('/messaging/providers', {
          name: name.trim(),
          type: entry.type,
          provider: entry.provider,
          config: cfgPayload,
        });
      }
    },
    onSuccess: onSaved,
  });

  const onSelectProvider = (value: string) => {
    setSelected(value);
    const next = catalogFor(value) ?? CATALOG[0];
    setName(next.label);
    const fresh: Record<string, string> = {};
    for (const f of next.fields) fresh[f.key] = '';
    setConfig(fresh);
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={editMode ? 'Edit provider' : 'Add provider'}
      subtitle={editMode ? entry.label : 'Configure a sender for this project'}
      width={500}
      submitLabel={editMode ? 'Save' : 'Add'}
      loading={save.isPending}
      submitDisabled={!name.trim()}
      onSubmit={() => save.mutate()}
    >
      {!editMode && (
        <SelectField
          label="Provider"
          value={selected}
          onChange={onSelectProvider}
          options={CATALOG.map((c) => ({
            value: c.provider,
            label: `${c.label} · ${CATEGORY_LABEL[c.type]}`,
          }))}
        />
      )}
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={entry.label}
      />
      {entry.fields.map((f) => (
        <TextField
          key={f.key}
          label={f.label}
          type={f.secret ? 'password' : 'text'}
          autoComplete={f.secret ? 'new-password' : 'off'}
          value={config[f.key] ?? ''}
          onChange={(e) => setConfig((prev) => ({ ...prev, [f.key]: e.target.value }))}
          placeholder={f.placeholder}
        />
      ))}
    </FormDialog>
  );
}
