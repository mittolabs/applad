import { type ReactNode, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ChevronLeft, Info, Key, Loader2 } from 'lucide-react';
import { api } from '@/api/client';
import { ErrorState } from '@/components/error-state';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ConfirmDialog } from '@/components/form-dialog';
import {
  EXPIRY_OPTIONS,
  ScopeGroups,
  SCOPE_GROUPS,
  expiresAtIso,
  expiryPreview,
  formatLongDate,
} from './scopes';

/* Ports console/lib/features/settings/api_key_detail_page.dart — a full-page
 * (in-shell) API key editor: details, name, scopes, expiration, delete. */

export function ApiKeyDetailPage() {
  const { projectId, keyId } = useParams<{ projectId: string; keyId: string }>();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['apiKey', projectId, keyId],
    enabled: !!projectId && !!keyId,
    queryFn: async () => {
      const res = await api.get(`/projects/${projectId}/keys/${keyId}`);
      return res.data as Record<string, unknown>;
    },
  });

  if (!projectId || !keyId) return null;
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
      </div>
    );
  }
  if (error || !data) {
    return (
      <div className="p-6 md:p-8">
        <ErrorState error={error} onRetry={refetch} />
      </div>
    );
  }

  return (
    <KeyDetailBody
      key={String(data.$id)}
      projectId={projectId}
      keyId={keyId}
      keyData={data}
      onSaved={refetch}
    />
  );
}

function KeyDetailBody({
  projectId,
  keyId,
  keyData,
  onSaved,
}: {
  projectId: string;
  keyId: string;
  keyData: Record<string, unknown>;
  onSaved: () => void;
}) {
  const navigate = useNavigate();

  const name = String(keyData.name ?? 'Unnamed');
  const secretPrefix = String(keyData.secretPrefix ?? '');
  const createdAt = keyData.$createdAt ? String(keyData.$createdAt) : null;
  const expireIso = keyData.expire ? String(keyData.expire) : null;

  const [nameInput, setNameInput] = useState(name);
  const [scopes, setScopes] = useState<Set<string>>(
    new Set((keyData.scopes as string[] | undefined) ?? []),
  );
  const [expiry, setExpiry] = useState('never');
  const [customDate, setCustomDate] = useState('');
  const [nameDirty, setNameDirty] = useState(false);
  const [scopesDirty, setScopesDirty] = useState(false);
  const [expiryDirty, setExpiryDirty] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const patch = (body: Record<string, unknown>) =>
    api.patch(`/projects/${projectId}/keys/${keyId}`, body);

  const saveName = useMutation({
    mutationFn: () => patch({ name: nameInput.trim() }),
    onSuccess: () => {
      setNameDirty(false);
      onSaved();
    },
  });

  const saveScopes = useMutation({
    mutationFn: () => patch({ scopes: [...scopes] }),
    onSuccess: () => {
      setScopesDirty(false);
      onSaved();
    },
  });

  const saveExpiry = useMutation({
    mutationFn: () => patch({ expiresAt: expiresAtIso(expiry, customDate) ?? '' }),
    onSuccess: () => {
      setExpiryDirty(false);
      onSaved();
    },
  });

  const del = useMutation({
    mutationFn: () => api.delete(`/projects/${projectId}/keys/${keyId}`),
    onSuccess: () => navigate(`/project/${projectId}/settings?tab=api-keys`),
  });

  const toggleScope = (scope: string) => {
    setScopesDirty(true);
    setScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) next.delete(scope);
      else next.add(scope);
      return next;
    });
  };
  const toggleGroup = (group: string) => {
    setScopesDirty(true);
    setScopes((prev) => {
      const next = new Set(prev);
      const groupScopes = SCOPE_GROUPS[group];
      const all = groupScopes.every((s) => next.has(s));
      groupScopes.forEach((s) => (all ? next.delete(s) : next.add(s)));
      return next;
    });
  };
  const selectAll = () => {
    setScopesDirty(true);
    setScopes(new Set(Object.values(SCOPE_GROUPS).flat()));
  };
  const deselectAll = () => {
    setScopesDirty(true);
    setScopes(new Set());
  };

  const preview = expiryPreview(expiry, customDate);
  const expiryDisplay = formatLongDate(expireIso) ?? 'Never';
  const createdDisplay = formatLongDate(createdAt) ?? createdAt ?? '—';

  return (
    <div className="mx-auto flex w-full max-w-[900px] flex-col gap-6 p-6 md:p-8">
      {/* Back / breadcrumb */}
      <button
        type="button"
        onClick={() => navigate(`/project/${projectId}/settings?tab=api-keys`)}
        className="flex w-fit items-center gap-2.5"
      >
        <ChevronLeft size={14} className="text-text-subtle" />
        <span className="text-[length:var(--text-title)] font-semibold text-text-primary">
          {name}
        </span>
        <span className="rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--color-accent)_12%,transparent)] px-2 py-1 text-[length:var(--text-caption)] font-medium text-[var(--color-accent)]">
          API Secret
        </span>
      </button>

      {/* Key details */}
      <SectionRow label="Key details">
        <div className="grid grid-cols-2 gap-4">
          <MetaItem label="Name" value={name} />
          <MetaItem label="Created" value={createdDisplay} />
          <MetaItem label="Expiration date" value={expiryDisplay} />
          <div>
            <div className="text-[length:var(--text-caption)] font-medium text-text-subtle">
              Secret
            </div>
            <div className="mt-1 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-secondary">
              {secretPrefix ? `${secretPrefix}···` : '•'.repeat(12)}
            </div>
            <div className="mt-1 text-[length:var(--text-2xs)] text-text-subtle">
              Full secret only shown once at creation
            </div>
          </div>
        </div>
      </SectionRow>

      {/* Name */}
      <SectionRow
        label="Name"
        description="Choose any name that will help you distinguish between API keys."
      >
        <div className="flex flex-col gap-3">
          <Input
            value={nameInput}
            placeholder="Key name"
            onChange={(e) => {
              setNameInput(e.target.value);
              setNameDirty(true);
            }}
          />
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={!nameDirty || !nameInput.trim()}
              loading={saveName.isPending}
              onClick={() => saveName.mutate()}
            >
              Save
            </Button>
          </div>
        </div>
      </SectionRow>

      {/* Scopes */}
      <SectionRow
        label="Scopes"
        description="Choose which permission scopes to grant your application. Only grant the permissions you need."
      >
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-3.5">
            <button
              type="button"
              onClick={selectAll}
              className="text-[length:var(--text-label)] text-[var(--color-accent)]"
            >
              Select all
            </button>
            <button
              type="button"
              onClick={deselectAll}
              className="text-[length:var(--text-label)] text-text-subtle hover:text-text-secondary"
            >
              Deselect all
            </button>
          </div>
          <ScopeGroups selected={scopes} onToggleScope={toggleScope} onToggleGroup={toggleGroup} />
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={!scopesDirty}
              loading={saveScopes.isPending}
              onClick={() => saveScopes.mutate()}
            >
              Save
            </Button>
          </div>
        </div>
      </SectionRow>

      {/* Expiration */}
      <SectionRow
        label="Expiration date"
        description="Set a date after which your API key will expire."
      >
        <div className="flex flex-col gap-3">
          <Select
            value={expiry}
            onValueChange={(v) => {
              setExpiry(v);
              setExpiryDirty(true);
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {EXPIRY_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {expiry === 'custom' && (
            <Input
              type="date"
              value={customDate}
              onChange={(e) => {
                setCustomDate(e.target.value);
                setExpiryDirty(true);
              }}
            />
          )}
          {preview && (
            <div className="flex items-center gap-1.5 text-[length:var(--text-label)] text-text-subtle">
              <Info size={12} />
              {preview}
            </div>
          )}
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={!expiryDirty}
              loading={saveExpiry.isPending}
              onClick={() => saveExpiry.mutate()}
            >
              Save
            </Button>
          </div>
        </div>
      </SectionRow>

      {/* Delete */}
      <SectionRow
        label="Delete API key"
        description="The API key will be permanently deleted. This action is irreversible."
      >
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2.5">
            <Key size={14} className="text-text-subtle" />
            <div>
              <div className="text-[length:var(--text-body)] font-medium text-text-primary">
                {name}
              </div>
              {createdAt && (
                <div className="text-[length:var(--text-caption)] text-text-subtle">
                  Created {createdDisplay}
                </div>
              )}
            </div>
          </div>
          <div className="h-px bg-border" />
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              className="border-[var(--color-danger)] text-[var(--status-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)]"
              onClick={() => setConfirmDelete(true)}
            >
              Delete
            </Button>
          </div>
        </div>
      </SectionRow>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete API key"
        message="Any applications using this key will lose access immediately. This action is irreversible."
        loading={del.isPending}
        onConfirm={() => del.mutate()}
      />
    </div>
  );
}

function SectionRow({
  label,
  description,
  children,
}: {
  label: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4 md:flex-row md:gap-8">
      <div className="w-full shrink-0 md:w-[220px]">
        <div className="text-[length:var(--text-control)] font-medium text-text-primary">
          {label}
        </div>
        {description && (
          <div className="mt-1.5 text-[length:var(--text-label)] text-text-subtle">
            {description}
          </div>
        )}
      </div>
      <div className="flex-1 rounded-[var(--radius-10)] border border-border bg-surface p-5">
        {children}
      </div>
    </div>
  );
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[length:var(--text-caption)] font-medium text-text-subtle">{label}</div>
      <div className="mt-1 text-[length:var(--text-body)] text-text-primary">{value}</div>
    </div>
  );
}
