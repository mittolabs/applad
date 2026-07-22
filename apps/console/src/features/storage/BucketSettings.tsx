import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ConfirmDialog } from '@/components/form-dialog';

/* Ports storage_page.dart _buildSettingsTab — bucket status, name, permissions,
 * file security, security settings, compression, max size, allowed extensions,
 * and a delete-bucket card. Toggles persist via PUT /storage/buckets/{id}. */

export function BucketSettings({
  bucketId,
  bucket,
  onDeleted,
}: {
  bucketId: string;
  bucket: Record<string, unknown> | undefined;
  onDeleted: () => void;
}) {
  const qc = useQueryClient();

  const name = (bucket?.['name'] as string) ?? '';
  const enabled = (bucket?.['enabled'] as boolean) ?? true;
  const fileSecurity = (bucket?.['fileSecurity'] as boolean) ?? false;
  // Unknown is not encrypted: this claimed data was encrypted before the
  // bucket had loaded, while its neighbours default to the safe side.
  const encryption = (bucket?.['encryption'] as boolean) ?? false;
  const antivirus = (bucket?.['antivirus'] as boolean) ?? false;
  const compression = (bucket?.['compression'] as string) ?? 'none';
  const maxSize = (bucket?.['maximumFileSize'] as number) ?? 0;
  const extensions = (bucket?.['allowedFileExtensions'] as string[]) ?? [];
  // The bucket model serializes permissions under $permissions; older writes
  // used a bare `permissions` key, so accept either when reading.
  const permissions =
    (bucket?.['$permissions'] as string[]) ?? (bucket?.['permissions'] as string[]) ?? [];
  const created = String(bucket?.['createdAt'] ?? bucket?.['$createdAt'] ?? '');
  const updated = String(bucket?.['updatedAt'] ?? bucket?.['$updatedAt'] ?? '');

  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      api.put(`/storage/buckets/${bucketId}`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['storage-bucket', bucketId] }),
    onError: (e) => setError(friendlyError(e)),
  });

  const del = useMutation({
    mutationFn: () => api.delete(`/storage/buckets/${bucketId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['/storage/buckets'] });
      onDeleted();
    },
  });

  const persist = (data: Record<string, unknown>) => {
    setError(null);
    update.mutate(data);
  };

  return (
    <div className="flex max-w-3xl flex-col gap-4">
      {error && (
        <div className="rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_30%,var(--border))] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[length:var(--text-body)] text-[var(--color-danger)]">
          {error}
        </div>
      )}

      {/* 1. Status */}
      <Section onUpdate={() => persist({ enabled: !enabled })} updating={update.isPending}>
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
              {name}
            </div>
            <div className="mt-1 text-[length:var(--text-label)] text-text-subtle">
              Created: {created || '—'}
            </div>
            <div className="text-[length:var(--text-label)] text-text-subtle">
              Last updated: {updated || '—'}
            </div>
          </div>
          <ToggleRow label="Enabled" checked={enabled} />
        </div>
      </Section>

      {/* 2. Name */}
      <Section title="Name">
        <NameField initial={name} onSave={(v) => persist({ name: v })} saving={update.isPending} />
      </Section>

      {/* 3. Permissions */}
      <Section title="Permissions" subtitle="Choose who can access your buckets and files.">
        <PermissionsTable
          permissions={permissions}
          saving={update.isPending}
          onSave={(perms) => persist({ permissions: perms })}
        />
      </Section>

      {/* 4. File security */}
      <Section
        title="File security"
        onUpdate={() => persist({ fileSecurity: !fileSecurity })}
        updating={update.isPending}
      >
        <ToggleRow label="File security" checked={fileSecurity} />
        <p className="mt-2 text-[length:var(--text-label)] text-text-subtle">
          {fileSecurity
            ? 'When file security is enabled, users will be able to access files for which they have been granted either file or bucket permissions.'
            : 'If file security is disabled, users can access files only if they have bucket permissions. File permissions will be ignored.'}
        </p>
      </Section>

      {/* 5. Security settings */}
      <Section
        title="Security settings"
        subtitle="Enable or disable security features for this bucket."
        onUpdate={() => persist({ encryption: !encryption })}
        updating={update.isPending}
      >
        <ToggleRow
          label="Encryption"
          checked={encryption}
          subtitle="Files inside this bucket will be encrypted. Files larger than 20MB will not be encrypted."
        />
        <div className="h-3" />
        <ToggleRow
          label="Antivirus"
          checked={antivirus}
          subtitle="Files inside this bucket will be scanned by the antivirus scanner."
        />
      </Section>

      {/* 6. Compression */}
      <Section
        title="Compression"
        subtitle="Choose an algorithm for compression. For files larger than 20MB, compression will be skipped."
      >
        <Label className="mb-1.5 block">Algorithm</Label>
        <Select value={compression} disabled>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">None</SelectItem>
            <SelectItem value="gzip">gzip</SelectItem>
            <SelectItem value="zstd">zstd</SelectItem>
          </SelectContent>
        </Select>
      </Section>

      {/* 7. Maximum file size */}
      <Section
        title="Maximum file size"
        subtitle="Set the maximum file size allowed in this bucket."
      >
        <div className="flex items-center gap-2">
          <span className="text-[length:var(--text-control)] text-text-primary">
            {maxSize > 0 ? (maxSize / (1024 * 1024)).toFixed(0) : 'Unlimited'}
          </span>
          {maxSize > 0 && <span className="text-[length:var(--text-body)] text-text-muted">MB</span>}
        </div>
      </Section>

      {/* 8. Allowed extensions */}
      <Section
        title="Allowed file extensions"
        subtitle="Allowed file extensions. A maximum of 100 file extensions can be added. Leave empty to allow all file types."
      >
        {extensions.length === 0 ? (
          <div className="text-[length:var(--text-body)] text-text-secondary">
            All file types allowed
          </div>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {extensions.map((ext) => (
              <span
                key={ext}
                className="rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] px-2 py-1 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary"
              >
                {ext}
              </span>
            ))}
          </div>
        )}
      </Section>

      {/* 9. Delete bucket */}
      <div className="rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_30%,var(--border))] bg-surface p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <div className="text-[length:var(--text-control)] font-medium text-[var(--color-danger)]">
              Delete bucket
            </div>
            <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
              The bucket will be permanently deleted, including all the files within it. This action
              is irreversible.
            </div>
          </div>
          <div className="rounded-[var(--radius)] border border-border bg-fill p-3">
            <div className="text-[length:var(--text-body)] font-medium text-text-primary">
              {name}
            </div>
            <div className="text-[length:var(--text-caption)] text-text-subtle">
              Last updated: {updated || '—'}
            </div>
          </div>
        </div>
        <div className="mt-3 flex justify-end">
          <Button
            variant="outline"
            className="border-[var(--color-danger)] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]"
            onClick={() => setConfirming(true)}
          >
            Delete
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Delete bucket"
        message="All files in this bucket will be permanently deleted."
        loading={del.isPending}
        onConfirm={() => del.mutate()}
      />
    </div>
  );
}

function Section({
  title,
  subtitle,
  children,
  onUpdate,
  updating,
}: {
  title?: string;
  subtitle?: string;
  children: React.ReactNode;
  onUpdate?: () => void;
  updating?: boolean;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      {title && (
        <div className="mb-4">
          <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
            {title}
          </div>
          {subtitle && (
            <div className="mt-1 text-[length:var(--text-body)] text-text-subtle">{subtitle}</div>
          )}
        </div>
      )}
      {children}
      {onUpdate && (
        <div className="mt-4 flex justify-end">
          <Button size="sm" loading={updating} onClick={onUpdate}>
            Update
          </Button>
        </div>
      )}
    </div>
  );
}

function ToggleRow({
  label,
  checked,
  subtitle,
}: {
  label: string;
  checked: boolean;
  subtitle?: string;
}) {
  // Reflects current server state; persisted by the section's Update button
  // (mirrors the Flutter toggles, whose onChanged is a no-op).
  const [on, setOn] = useState(checked);
  useEffect(() => setOn(checked), [checked]);
  return (
    <div>
      <div className="flex items-center gap-3">
        <Switch checked={on} onCheckedChange={setOn} />
        <span className="text-[length:var(--text-body)] font-medium text-text-primary">{label}</span>
      </div>
      {subtitle && (
        <div className="mt-1 pl-12 text-[length:var(--text-label)] text-text-subtle">{subtitle}</div>
      )}
    </div>
  );
}

function NameField({
  initial,
  onSave,
  saving,
}: {
  initial: string;
  onSave: (value: string) => void;
  saving?: boolean;
}) {
  const [value, setValue] = useState(initial);
  useEffect(() => setValue(initial), [initial]);
  return (
    <div>
      <Label className="mb-1.5 block">Name</Label>
      <div className="flex gap-3">
        <Input value={value} onChange={(e) => setValue(e.target.value)} className="flex-1" />
        <Button loading={saving} disabled={!value.trim()} onClick={() => onSave(value.trim())}>
          Update
        </Button>
      </div>
    </div>
  );
}

// Appwrite-style permission strings: action("role") — e.g. read("any"),
// create("users"), delete("guests"). The grid manages the three standard
// roles across the four actions; any permission string outside this space
// (a user- or team-scoped grant) is preserved untouched on save.
const ROLES: { token: string; label: string }[] = [
  { token: 'users', label: 'Users' },
  { token: 'guests', label: 'Guests' },
  { token: 'any', label: 'Any' },
];
const PERMS: { token: string; label: string }[] = [
  { token: 'create', label: 'Create' },
  { token: 'read', label: 'Read' },
  { token: 'update', label: 'Update' },
  { token: 'delete', label: 'Delete' },
];

const GRID_TOKENS = new Set<string>(
  ROLES.flatMap((r) => PERMS.map((p) => `${p.token}("${r.token}")`)),
);

function PermissionsTable({
  permissions,
  onSave,
  saving,
}: {
  permissions: string[];
  onSave: (permissions: string[]) => void;
  saving?: boolean;
}) {
  const [grid, setGrid] = useState<Set<string>>(() => new Set(permissions));
  const [baseline, setBaseline] = useState<string[]>(permissions);

  // Re-sync when the bucket reloads with a different set.
  useEffect(() => {
    const next = [...permissions].sort().join('|');
    const cur = [...baseline].sort().join('|');
    if (next !== cur) {
      setGrid(new Set(permissions));
      setBaseline(permissions);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [permissions]);

  const perm = (action: string, role: string) => `${action}("${role}")`;
  const has = (action: string, role: string) => grid.has(perm(action, role));
  const toggle = (action: string, role: string) => {
    setGrid((prev) => {
      const next = new Set(prev);
      const key = perm(action, role);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const dirty = [...grid].sort().join('|') !== [...baseline].sort().join('|');

  const save = () => {
    // Keep any grant the grid does not represent (e.g. user:/team: scopes).
    const preserved = baseline.filter((p) => !GRID_TOKENS.has(p));
    const gridPerms = [...grid].filter((p) => GRID_TOKENS.has(p));
    onSave([...preserved, ...gridPerms]);
    setBaseline([...preserved, ...gridPerms]);
  };

  return (
    <div>
      <div className="grid grid-cols-[80px_repeat(4,1fr)] items-center">
        <span />
        {PERMS.map((p) => (
          <span
            key={p.token}
            className="text-center text-[length:var(--text-caption)] font-medium text-text-muted"
          >
            {p.label}
          </span>
        ))}
      </div>
      {ROLES.map((role) => (
        <div key={role.token} className="grid grid-cols-[80px_repeat(4,1fr)] items-center py-1.5">
          <span className="text-[length:var(--text-body)] text-text-primary">{role.label}</span>
          {PERMS.map((p) => (
            <div key={p.token} className="flex justify-center">
              <Checkbox
                checked={has(p.token, role.token)}
                onCheckedChange={() => toggle(p.token, role.token)}
              />
            </div>
          ))}
        </div>
      ))}
      <div className="mt-4 flex justify-end">
        <Button size="sm" loading={saving} disabled={!dirty} onClick={save}>
          Update
        </Button>
      </div>
    </div>
  );
}
