import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Eye,
  EyeOff,
  GitBranch,
  Globe,
  KeyRound,
  Pencil,
  Plus,
  Tag,
  Trash2,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import { useTabIndex } from '@/hooks/use-tab-param';
import { PageTabs } from '@/components/page-tabs';
import { ErrorState } from '@/components/error-state';
import { Button } from '@/components/ui/button';
import { FormDialog, TextField } from '@/components/form-dialog';
import type { Row } from '@/components/data-table';

const TABS = ['Overview', 'Variables', 'Settings'];

function parseVars(raw: unknown): Record<string, string> {
  if (raw && typeof raw === 'object') {
    return Object.fromEntries(
      Object.entries(raw as Record<string, unknown>).map(([k, v]) => [k, String(v ?? '')]),
    );
  }
  return {};
}

export function EnvDetail({
  envId,
  onListChanged,
}: {
  envId: string;
  onListChanged: () => void;
}) {
  const [tab, setTab] = useTabIndex(TABS);

  const query = useQuery({
    queryKey: ['deploy-environment', envId],
    queryFn: async () => {
      const res = await api.get(`/deploy/environments/${envId}`);
      return res.data as Row;
    },
  });

  const env = query.data;
  const name = String(env?.['name'] ?? '…');

  return (
    <div className="flex h-full flex-col">
      <div className="px-6 pb-2 pt-4 md:px-8">
        <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">{name}</h2>
      </div>
      <PageTabs tabs={TABS} selected={tab} onChange={setTab} className="px-6 md:px-8" />

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : query.isLoading || !env ? (
        <div className="p-6 text-[length:var(--text-body)] text-text-muted md:p-8">Loading…</div>
      ) : tab === 0 ? (
        <OverviewTab env={env} />
      ) : tab === 1 ? (
        <VariablesTab
          key={`vars-${envId}`}
          envId={envId}
          env={env}
          onSaved={() => {
            query.refetch();
          }}
        />
      ) : (
        <SettingsTab
          key={`settings-${envId}`}
          envId={envId}
          env={env}
          onSaved={() => {
            query.refetch();
            onListChanged();
          }}
        />
      )}
    </div>
  );
}

// ── Overview ────────────────────────────────────────────────────────────────

function OverviewTab({ env }: { env: Row }) {
  const slug = String(env['slug'] ?? '');
  const branch = String(env['branch'] ?? '');
  const domain = String(env['domain'] ?? '');
  const vars = Object.keys(parseVars(env['envVars'])).length;

  return (
    <div className="p-6 md:p-8">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <InfoCard icon={<Tag size={16} />} label="Slug" value={slug || '—'} />
        <InfoCard icon={<GitBranch size={16} />} label="Branch" value={branch || 'all branches'} />
        <InfoCard icon={<Globe size={16} />} label="Domain" value={domain || '—'} />
        <InfoCard icon={<KeyRound size={16} />} label="Variables" value={String(vars)} />
      </div>
    </div>
  );
}

function InfoCard({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex flex-col gap-2 rounded-[var(--radius)] border border-border bg-surface p-4">
      <span className="text-text-secondary">{icon}</span>
      <span className="truncate text-[length:var(--text-subhead)] font-semibold text-text-primary">
        {value}
      </span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">{label}</span>
    </div>
  );
}

// ── Variables ───────────────────────────────────────────────────────────────

function VariablesTab({
  envId,
  env,
  onSaved,
}: {
  envId: string;
  env: Row;
  onSaved: () => void;
}) {
  const [vars, setVars] = useState<Record<string, string>>(() => parseVars(env['envVars']));
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editKey, setEditKey] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    try {
      await api.put(`/deploy/environments/${envId}`, {
        name: env['name'],
        branch: env['branch'] ?? '',
        domain: env['domain'] ?? '',
        envVars: vars,
      });
      toast.success('Variables saved');
      setDirty(false);
      onSaved();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  const upsert = (key: string, value: string, previousKey: string | null) => {
    setVars((prev) => {
      const next = { ...prev };
      if (previousKey && previousKey !== key) delete next[previousKey];
      next[key] = value;
      return next;
    });
    setDirty(true);
  };

  const remove = (key: string) => {
    setVars((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
    setDirty(true);
  };

  const entries = Object.entries(vars);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 px-6 py-3 md:px-8">
        <span className="text-[length:var(--text-body)] font-medium text-text-primary">
          Environment variables
        </span>
        <div className="ml-auto flex items-center gap-2">
          {dirty && (
            <Button size="sm" loading={saving} onClick={save}>
              Save
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setEditKey(null);
              setEditorOpen(true);
            }}
          >
            <Plus size={14} />
            Add variable
          </Button>
        </div>
      </div>
      <div className="h-px bg-border" />

      {entries.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 py-10 text-center">
          <KeyRound size={40} className="text-text-subtle" />
          <span className="text-[length:var(--text-body)] text-text-secondary">No variables yet</span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setEditKey(null);
              setEditorOpen(true);
            }}
          >
            <Plus size={14} />
            Add variable
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-2 p-6 md:p-8">
          {entries.map(([k, v]) => (
            <VarRow
              key={k}
              varKey={k}
              varValue={v}
              onEdit={() => {
                setEditKey(k);
                setEditorOpen(true);
              }}
              onDelete={() => remove(k)}
            />
          ))}
        </div>
      )}

      <VariableEditorDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        initialKey={editKey}
        initialValue={editKey !== null ? vars[editKey] : ''}
        existingKeys={Object.keys(vars)}
        onSubmit={(key, value) => upsert(key, value, editKey)}
      />
    </div>
  );
}

function VarRow({
  varKey,
  varValue,
  onEdit,
  onDelete,
}: {
  varKey: string;
  varValue: string;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [obscure, setObscure] = useState(true);
  const masked = '•'.repeat(Math.min(24, Math.max(8, varValue.length)));
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-3 py-2">
      <span className="w-52 shrink-0 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-primary">
        {varKey}
      </span>
      <span className="h-5 w-px bg-border" />
      <span
        className={`flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] ${
          obscure ? 'text-text-subtle' : 'text-text-secondary'
        }`}
      >
        {obscure ? masked : varValue}
      </span>
      <button
        type="button"
        onClick={() => setObscure((v) => !v)}
        className="text-text-secondary transition-colors hover:text-text-primary"
        aria-label={obscure ? 'Show value' : 'Hide value'}
      >
        {obscure ? <Eye size={14} /> : <EyeOff size={14} />}
      </button>
      <button
        type="button"
        onClick={onEdit}
        className="text-text-secondary transition-colors hover:text-text-primary"
        aria-label="Edit variable"
      >
        <Pencil size={14} />
      </button>
      <button
        type="button"
        onClick={onDelete}
        className="text-text-secondary transition-colors hover:text-[var(--color-danger)]"
        aria-label="Delete variable"
      >
        <Trash2 size={14} />
      </button>
    </div>
  );
}

function VariableEditorDialog({
  open,
  onOpenChange,
  initialKey,
  initialValue,
  existingKeys,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialKey: string | null;
  initialValue: string;
  existingKeys: string[];
  onSubmit: (key: string, value: string) => void;
}) {
  const [key, setKey] = useState('');
  const [value, setValue] = useState('');

  useEffect(() => {
    if (open) {
      setKey(initialKey ?? '');
      setValue(initialValue);
    }
  }, [open, initialKey, initialValue]);

  const isEdit = initialKey !== null;
  const trimmedKey = key.trim();
  const duplicate =
    !!trimmedKey && trimmedKey !== initialKey && existingKeys.includes(trimmedKey);

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? 'Edit variable' : 'Add variable'}
      submitLabel={isEdit ? 'Save' : 'Add'}
      submitDisabled={!trimmedKey || duplicate}
      onSubmit={() => {
        onOpenChange(false);
        onSubmit(trimmedKey, value);
      }}
    >
      <TextField
        label="Key"
        placeholder="VARIABLE_NAME"
        value={key}
        error={duplicate ? 'A variable with this key already exists' : undefined}
        onChange={(e) => setKey(e.target.value)}
        autoFocus
      />
      <TextField
        label="Value"
        placeholder="value"
        value={value}
        onChange={(e) => setValue(e.target.value)}
      />
    </FormDialog>
  );
}

// ── Settings ────────────────────────────────────────────────────────────────

function SettingsTab({
  envId,
  env,
  onSaved,
}: {
  envId: string;
  env: Row;
  onSaved: () => void;
}) {
  const [name, setName] = useState(String(env['name'] ?? ''));
  const [branch, setBranch] = useState(String(env['branch'] ?? ''));
  const [domain, setDomain] = useState(String(env['domain'] ?? ''));
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      await api.put(`/deploy/environments/${envId}`, {
        name: name.trim(),
        branch: branch.trim(),
        domain: domain.trim(),
        envVars: env['envVars'] ?? {},
      });
      toast.success('Settings saved');
      onSaved();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex max-w-lg flex-col gap-4 rounded-[var(--radius-10)] border border-border bg-surface p-5">
        <span className="text-[length:var(--text-body)] font-semibold text-text-primary">General</span>
        <TextField
          label="Name"
          placeholder="Environment name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <TextField
          label="Branch"
          placeholder="e.g. main (optional)"
          value={branch}
          onChange={(e) => setBranch(e.target.value)}
        />
        <TextField
          label="Domain"
          placeholder="e.g. staging.example.com"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
        />
      </div>
      <div>
        <Button loading={saving} disabled={!name.trim()} onClick={save}>
          Save changes
        </Button>
      </div>
    </div>
  );
}
