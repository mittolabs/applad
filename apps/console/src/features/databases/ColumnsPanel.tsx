import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckSquare, Columns3, Lock, Plus, Square, X } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormDialog, ConfirmDialog } from '@/components/form-dialog';
import { columnTypeIcon, str, type Json } from './shared';

const COLUMN_TYPES = [
  { value: 'string', label: 'String' },
  { value: 'integer', label: 'Integer' },
  { value: 'float', label: 'Float' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'datetime', label: 'Datetime' },
  { value: 'email', label: 'Email' },
  { value: 'url', label: 'URL' },
  { value: 'enum', label: 'Enum' },
  { value: 'point', label: 'Point' },
  { value: 'relationship', label: 'Relationship' },
] as const;

export function ColumnsPanel({
  dbId,
  tableId,
  showCreate,
  setShowCreate,
}: {
  dbId: string;
  tableId: string;
  showCreate: boolean;
  setShowCreate: (v: boolean) => void;
}) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [permTarget, setPermTarget] = useState<Json | null>(null);

  const query = useQuery({
    queryKey: ['db-columns', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/columns`);
      return (res.data as { columns?: Json[] }).columns ?? [];
    },
  });
  const columns = query.data ?? [];
  const invalidate = () => qc.invalidateQueries({ queryKey: ['db-columns', dbId, tableId] });

  const del = async () => {
    if (!deleteTarget) return;
    await api.delete(`${base}/columns/${deleteTarget}`);
    setDeleteTarget(null);
    invalidate();
  };

  return (
    <div className="flex items-start gap-4">
      <div className="flex min-w-0 flex-1 flex-col gap-4">
        <div className="flex justify-end">
          <Button size="sm" onClick={() => setShowCreate(true)}>
            <Plus size={14} />
            Create column
          </Button>
        </div>

        {query.error ? (
          <ErrorState error={query.error} onRetry={query.refetch} />
        ) : query.isLoading ? (
          <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">
            Loading…
          </div>
        ) : columns.length === 0 ? (
          <EmptyState
            icon={Columns3}
            title="No columns yet"
            subtitle="Define your data schema by creating columns."
            actionLabel="Create column"
            onAction={() => setShowCreate(true)}
          />
        ) : (
          <div className="overflow-x-auto rounded-[var(--radius-10)] border border-border">
            <table className="w-full border-collapse text-left">
              <thead>
                <tr className="border-b border-border text-[length:var(--text-label)] font-medium text-text-muted">
                  <th className="px-4 py-2.5">Column name</th>
                  <th className="px-4 py-2.5">Type</th>
                  <th className="px-4 py-2.5">Required</th>
                  <th className="px-4 py-2.5">Default value</th>
                  <th className="px-4 py-2.5">Permissions</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {columns.map((col) => {
                  const key = str(col['key']);
                  const type = str(col['type']);
                  const required = col['required'] === true;
                  const encrypted = col['encrypted'] === true;
                  const def = str(col['default']) || '-';
                  const perms = (col['$permissions'] as string[] | undefined) ?? ['read', 'write'];
                  const Icon = columnTypeIcon(type);
                  return (
                    <tr
                      key={key}
                      onClick={() => setPermTarget(col)}
                      className="group cursor-pointer border-b border-[var(--fill)] last:border-0 hover:bg-fill"
                    >
                      <td className="px-4 py-3 text-[length:var(--text-body)] text-text-primary">
                        <span className="inline-flex items-center gap-2">
                          <Icon size={14} className="text-[var(--color-accent)]" />
                          {key}
                          {required && (
                            <span className="rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium text-[var(--color-accent)]">
                              required
                            </span>
                          )}
                          {encrypted && (
                            <span
                              title="Values are stored as opaque ciphertext at rest"
                              className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--status-success)_15%,transparent)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium text-[var(--status-success)]"
                            >
                              <Lock size={10} />
                              encrypted
                            </span>
                          )}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">{type}</td>
                      <td className="px-4 py-3">
                        {required ? (
                          <CheckSquare size={14} className="text-[var(--color-accent)]" />
                        ) : (
                          <Square size={14} className="text-text-subtle" />
                        )}
                      </td>
                      <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">{def}</td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {perms.includes('read') && <PermChip label="read" variant="success" />}
                          {perms.includes('write') && <PermChip label="write" variant="accent" />}
                          {!perms.includes('read') && !perms.includes('write') && (
                            <PermChip label="none" variant="danger" />
                          )}
                        </div>
                      </td>
                      <td className="px-2 py-3">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setDeleteTarget(key);
                          }}
                          className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
                          aria-label="Delete column"
                        >
                          <X size={14} />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {showCreate && (
        <CreateColumnPanel
          base={base}
          onClose={() => setShowCreate(false)}
          onCreated={() => {
            setShowCreate(false);
            invalidate();
          }}
        />
      )}

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete column"
        message="Are you sure? This action cannot be undone."
        onConfirm={del}
      />

      {permTarget && (
        <ColumnPermDialog
          base={base}
          column={permTarget}
          onClose={() => setPermTarget(null)}
          onSaved={() => {
            setPermTarget(null);
            invalidate();
          }}
        />
      )}
    </div>
  );
}

function PermChip({
  label,
  variant,
}: {
  label: string;
  variant: 'success' | 'accent' | 'danger';
}) {
  const color =
    variant === 'success'
      ? 'var(--status-success)'
      : variant === 'danger'
        ? 'var(--color-danger)'
        : 'var(--color-accent)';
  return (
    <span
      className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)` }}
    >
      {label}
    </span>
  );
}

function CreateColumnPanel({
  base,
  onClose,
  onCreated,
}: {
  base: string;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [key, setKey] = useState('');
  const [type, setType] = useState<string>('string');
  const [size, setSize] = useState('256');
  const [defaultVal, setDefaultVal] = useState('');
  const [elements, setElements] = useState('');
  const [required, setRequired] = useState(false);
  const [array, setArray] = useState(false);
  const [encrypted, setEncrypted] = useState(false);
  const [minLen, setMinLen] = useState('');
  const [maxLen, setMaxLen] = useState('');
  const [minVal, setMinVal] = useState('');
  const [maxVal, setMaxVal] = useState('');
  const [pattern, setPattern] = useState('');
  const [message, setMessage] = useState('');
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const showValidation = !['boolean', 'datetime', 'point', 'relationship'].includes(type);

  const create = async () => {
    if (!key.trim()) return;
    setCreating(true);
    setError(null);
    try {
      const data: Json = { key: key.trim(), required, array, encrypted };
      if (defaultVal.trim()) data['default'] = defaultVal.trim();
      if (type === 'string') data['size'] = Number.parseInt(size, 10) || 256;
      if (type === 'enum') {
        data['elements'] = elements
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean);
      }
      const validation: Json = {};
      if (type === 'string') {
        const a = Number.parseInt(minLen.trim(), 10);
        const b = Number.parseInt(maxLen.trim(), 10);
        if (!Number.isNaN(a)) validation['minLength'] = a;
        if (!Number.isNaN(b)) validation['maxLength'] = b;
      }
      if (type === 'integer' || type === 'float') {
        const a = Number.parseFloat(minVal.trim());
        const b = Number.parseFloat(maxVal.trim());
        if (!Number.isNaN(a)) validation['min'] = a;
        if (!Number.isNaN(b)) validation['max'] = b;
      }
      if (pattern.trim()) validation['pattern'] = pattern.trim();
      if (message.trim()) validation['message'] = message.trim();
      if (Object.keys(validation).length > 0) data['validation'] = validation;

      await api.post(`${base}/columns/${type}`, data);
      onCreated();
    } catch (e) {
      setError(friendlyError(e));
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="flex w-[320px] shrink-0 flex-col rounded-[var(--radius-10)] border border-border bg-surface">
      <div className="flex items-center justify-between px-4 pt-4">
        <div className="text-[length:var(--text-control)] font-semibold text-text-primary">
          Create column
        </div>
        <button
          type="button"
          onClick={onClose}
          className="text-text-subtle hover:text-text-primary"
          aria-label="Close"
        >
          <X size={16} />
        </button>
      </div>

      <div className="flex max-h-[560px] flex-col gap-3 overflow-y-auto px-4 py-4">
        <Field label="Key">
          <Input value={key} onChange={(e) => setKey(e.target.value)} placeholder="Enter key" />
        </Field>
        <Field label="Type">
          <Select value={type} onValueChange={setType}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {COLUMN_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        {type === 'string' && (
          <Field label="Size">
            <Input value={size} onChange={(e) => setSize(e.target.value)} placeholder="256" />
          </Field>
        )}
        {type === 'enum' && (
          <Field label="Elements (comma separated)">
            <Input
              value={elements}
              onChange={(e) => setElements(e.target.value)}
              placeholder="value1, value2, value3"
            />
          </Field>
        )}

        <Field label="Default (optional)">
          <Input
            value={defaultVal}
            onChange={(e) => setDefaultVal(e.target.value)}
            placeholder="Enter default value"
          />
        </Field>

        <ToggleRow
          label="Required"
          subtitle="Indicate whether this column is required."
          value={required}
          onChange={setRequired}
        />
        <ToggleRow
          label="Array"
          subtitle="Indicate whether this column is an array."
          value={array}
          onChange={setArray}
          disabled={encrypted}
        />
        <ToggleRow
          label="Encrypted"
          subtitle={
            array
              ? 'Not available for array columns — store a single encrypted JSON value instead.'
              : 'Store this column’s values as opaque ciphertext at rest. Requires field-level encryption to be configured on this instance.'
          }
          value={encrypted}
          onChange={setEncrypted}
          disabled={array}
        />

        {showValidation && (
          <>
            <div className="mt-2 text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-secondary">
              Validation
            </div>
            {type === 'string' && (
              <>
                <Field label="Min length">
                  <Input value={minLen} onChange={(e) => setMinLen(e.target.value)} placeholder="e.g. 3" />
                </Field>
                <Field label="Max length">
                  <Input value={maxLen} onChange={(e) => setMaxLen(e.target.value)} placeholder="e.g. 255" />
                </Field>
              </>
            )}
            {(type === 'integer' || type === 'float') && (
              <>
                <Field label="Min value">
                  <Input value={minVal} onChange={(e) => setMinVal(e.target.value)} placeholder="e.g. 0" />
                </Field>
                <Field label="Max value">
                  <Input value={maxVal} onChange={(e) => setMaxVal(e.target.value)} placeholder="e.g. 100" />
                </Field>
              </>
            )}
            {(type === 'string' || type === 'email') && (
              <>
                <Field label="Regex pattern">
                  <Input
                    value={pattern}
                    onChange={(e) => setPattern(e.target.value)}
                    placeholder="e.g. ^[a-zA-Z]+$"
                  />
                </Field>
                <Field label="Custom error message">
                  <Input
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    placeholder="e.g. Invalid format"
                  />
                </Field>
              </>
            )}
          </>
        )}

        {error && <div className="text-[length:var(--text-caption)] text-[var(--color-danger)]">{error}</div>}
      </div>

      <div className="flex justify-end gap-2 border-t border-border p-4">
        <Button variant="ghost" onClick={onClose}>
          Cancel
        </Button>
        <Button loading={creating} disabled={!key.trim()} onClick={create}>
          Create
        </Button>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function ToggleRow({
  label,
  subtitle,
  value,
  onChange,
  disabled,
}: {
  label: string;
  subtitle: string;
  value: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className={`flex items-start gap-3${disabled ? ' opacity-50' : ''}`}>
      <Switch checked={value} onCheckedChange={onChange} disabled={disabled} />
      <div className="min-w-0">
        <div className="text-[length:var(--text-body)] text-text-primary">{label}</div>
        <div className="text-[length:var(--text-caption)] text-text-muted">{subtitle}</div>
      </div>
    </div>
  );
}

function ColumnPermDialog({
  base,
  column,
  onClose,
  onSaved,
}: {
  base: string;
  column: Json;
  onClose: () => void;
  onSaved: () => void;
}) {
  const key = str(column['key']);
  const perms = (column['$permissions'] as string[] | undefined) ?? ['read', 'write'];
  const [allowRead, setAllowRead] = useState(perms.includes('read'));
  const [allowWrite, setAllowWrite] = useState(perms.includes('write'));
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      const updated = [...(allowRead ? ['read'] : []), ...(allowWrite ? ['write'] : [])];
      await api.post(`${base}/columns/${key}/permissions`, { permissions: updated });
      onSaved();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`Column permissions: ${key}`}
      submitLabel="Save"
      loading={saving}
      onSubmit={save}
    >
      <p className="text-[length:var(--text-body)] text-text-secondary">
        Control whether this column value can be read or written via the API.
      </p>
      <ToggleRow
        label="Allow read"
        subtitle="Column values are returned when fetching rows."
        value={allowRead}
        onChange={setAllowRead}
      />
      <ToggleRow
        label="Allow write"
        subtitle="Column values can be set when creating or updating rows."
        value={allowWrite}
        onChange={setAllowWrite}
      />
    </FormDialog>
  );
}
