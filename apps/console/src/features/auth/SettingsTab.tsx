import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Apple, Check, Copy, Eye, EyeOff, GitBranch, Info, Music, Trash2, type LucideIcon } from 'lucide-react';
import { friendlyError } from '@/api/client';
import { ErrorState } from '@/components/error-state';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { FormDialog, FormField } from '@/components/form-dialog';
import { AUTH_METHODS, OAUTH_PROVIDERS, type OAuthProvider } from './auth-config';
import {
  deleteOAuthProvider,
  listOAuthProviders,
  setOAuthProvider,
  type OAuthProviderConfig,
} from './oauth-api';

export function SettingsTab({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [configuring, setConfiguring] = useState<OAuthProvider | null>(null);

  const query = useQuery({
    queryKey: ['oauth-providers', projectId],
    queryFn: () => listOAuthProviders(projectId),
  });

  const byProvider = useMemo(() => {
    const map: Record<string, OAuthProviderConfig> = {};
    for (const c of query.data ?? []) map[c.provider] = c;
    return map;
  }, [query.data]);

  const invalidate = () => qc.invalidateQueries({ queryKey: ['oauth-providers', projectId] });

  // Card-level toggle: turning a provider on requires a client id, so send the
  // user to the dialog when none is stored yet; otherwise flip enabled in place
  // (an empty secret preserves the stored one).
  const toggle = useMutation({
    mutationFn: ({ provider, enabled }: { provider: string; enabled: boolean }) =>
      setOAuthProvider(projectId, provider, {
        clientId: byProvider[provider]?.clientId ?? '',
        enabled,
      }),
    onSuccess: invalidate,
  });

  const onCardToggle = (p: OAuthProvider, next: boolean) => {
    const cfg = byProvider[p.id];
    if (next && !cfg?.clientId) {
      setConfiguring(p);
      return;
    }
    toggle.mutate({ provider: p.id, enabled: next });
  };

  return (
    <div className="pb-10">
      {/* Auth methods — instance-level capabilities, not per-project settings. */}
      <SectionHeader
        title="Auth methods"
        subtitle="Authentication methods available at the instance level. These are configured through the server environment and SDK, not saved per project."
      />
      <div className="mt-4 grid grid-cols-1 divide-y divide-border overflow-hidden rounded-[var(--radius)] border border-border bg-surface md:grid-cols-2 md:divide-y-0">
        {AUTH_METHODS.map((m) => {
          const on = Boolean(m.defaultOn);
          const Icon = m.icon;
          return (
            <div
              key={m.id}
              className="flex items-center gap-3 border-border px-4 py-3.5 max-md:border-b max-md:last:border-b-0 md:[&:nth-child(odd)]:border-r md:[&:not(:nth-last-child(-n+2))]:border-b"
            >
              <div
                className="flex h-[30px] w-[30px] items-center justify-center rounded-[var(--radius-6)]"
                style={{
                  backgroundColor: on ? 'color-mix(in srgb, var(--color-accent) 12%, transparent)' : 'var(--fill)',
                  color: on ? 'var(--color-accent)' : 'var(--text-subtle)',
                }}
              >
                <Icon size={14} />
              </div>
              <div className="flex-1">
                <div className="text-[length:var(--text-body)] font-medium text-text-primary">{m.label}</div>
                <div className="text-[length:var(--text-caption)] text-text-subtle">{m.description}</div>
              </div>
              <span className="text-[length:var(--text-caption)] text-text-subtle">
                {on ? 'Default on' : 'Available'}
              </span>
            </div>
          );
        })}
      </div>

      {/* OAuth providers — persisted per project via the config API. */}
      <div className="mt-8">
        <SectionHeader
          title="OAuth2 Providers"
          subtitle="Allow users to sign in with their existing third-party accounts."
        />
      </div>

      {query.error ? (
        <div className="mt-4">
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </div>
      ) : (
        <>
          {toggle.error && (
            <div className="mt-3 text-[length:var(--text-caption)] text-[var(--color-danger)]">
              {friendlyError(toggle.error)}
            </div>
          )}
          <div className="mt-4 grid grid-cols-2 gap-2.5 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5">
            {OAUTH_PROVIDERS.map((p) => (
              <ProviderCard
                key={p.id}
                provider={p}
                enabled={byProvider[p.id]?.enabled ?? false}
                loading={query.isLoading}
                onToggle={(v) => onCardToggle(p, v)}
                onConfigure={() => setConfiguring(p)}
              />
            ))}
          </div>
        </>
      )}

      {configuring && (
        <OAuthConfigDialog
          provider={configuring}
          projectId={projectId}
          config={byProvider[configuring.id]}
          onClose={() => setConfiguring(null)}
          onSaved={() => {
            invalidate();
            setConfiguring(null);
          }}
        />
      )}
    </div>
  );
}

function SectionHeader({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div>
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">{title}</div>
      <div className="mt-0.5 text-[length:var(--text-body)] text-text-muted">{subtitle}</div>
    </div>
  );
}

// Ports auth_page.dart `_iconFor`: four providers ship no letter glyph and
// render a Lucide icon instead of a first-char fallback.
const PROVIDER_ICON: Record<string, LucideIcon> = {
  github: GitBranch,
  apple: Apple,
  spotify: Music,
  gitlab: GitBranch,
};

function ProviderBadge({ provider, size = 28 }: { provider: OAuthProvider; size?: number }) {
  const Icon = PROVIDER_ICON[provider.id];
  return (
    <div
      className="flex items-center justify-center rounded-[var(--radius-7)] font-bold"
      style={{
        width: size,
        height: size,
        backgroundColor: `color-mix(in srgb, ${provider.color} 15%, transparent)`,
        color: provider.color,
        fontSize: size * 0.4,
      }}
    >
      {Icon ? <Icon size={size * 0.5} /> : provider.letter || provider.name[0]}
    </div>
  );
}

function ProviderCard({
  provider,
  enabled,
  loading,
  onToggle,
  onConfigure,
}: {
  provider: OAuthProvider;
  enabled: boolean;
  loading: boolean;
  onToggle: (v: boolean) => void;
  onConfigure: () => void;
}) {
  const green = '#10B981';
  return (
    <button
      type="button"
      onClick={onConfigure}
      className="flex flex-col items-start rounded-[var(--radius)] border bg-surface p-3.5 text-left transition-colors hover:bg-fill-hover"
      style={{ borderColor: enabled ? `color-mix(in srgb, ${green} 35%, transparent)` : 'var(--border)' }}
    >
      <div className="flex w-full items-center">
        <ProviderBadge provider={provider} />
        <span
          role="switch"
          aria-checked={enabled}
          onClick={(e) => {
            e.stopPropagation();
            if (!loading) onToggle(!enabled);
          }}
          className="ml-auto flex h-[18px] w-8 items-center rounded-full border border-border p-[3px] transition-colors"
          style={{ backgroundColor: enabled ? 'var(--color-accent)' : 'var(--fill)', opacity: loading ? 0.5 : 1 }}
        >
          <span
            className="h-3 w-3 rounded-full bg-white transition-transform"
            style={{ transform: enabled ? 'translateX(14px)' : 'translateX(0)' }}
          />
        </span>
      </div>
      <div className="mt-2.5 w-full truncate text-[length:var(--text-label)] font-medium text-text-primary">
        {provider.name}
      </div>
      <div
        className="mt-1 text-[length:var(--text-caption)]"
        style={{ color: enabled ? green : 'var(--text-subtle)', fontWeight: enabled ? 500 : 400 }}
      >
        {enabled ? 'enabled' : 'disabled'}
      </div>
    </button>
  );
}

function OAuthConfigDialog({
  provider,
  projectId,
  config,
  onClose,
  onSaved,
}: {
  provider: OAuthProvider;
  projectId: string;
  config?: OAuthProviderConfig;
  onClose: () => void;
  onSaved: () => void;
}) {
  // The first text field maps to clientId, the first secret/multiline field to
  // clientSecret — the two values the backend stores for any provider.
  const idField = useMemo(
    () => provider.fields.find((f) => (f.type ?? 'text') === 'text') ?? provider.fields[0],
    [provider],
  );
  const secretField = useMemo(
    () => provider.fields.find((f) => f.type === 'secret' || f.type === 'multiline'),
    [provider],
  );

  // Fields that are neither the client id nor the secret map to `extra`:
  // provider-specific, non-secret identifiers (Microsoft tenantId, Apple
  // keyId/teamId) the backend used to drop on the floor.
  const extraFields = useMemo(
    () => provider.fields.filter((f) => f !== idField && f !== secretField),
    [provider, idField, secretField],
  );

  const [enabled, setEnabled] = useState(config?.enabled ?? true);
  // Prefill the client id and any stored aux fields; secrets are never returned
  // so their fields start empty and, left empty, keep the stored value.
  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    if (idField) init[idField.key] = config?.clientId ?? '';
    for (const f of provider.fields) {
      if (f === idField || f === secretField) continue;
      init[f.key] = config?.extra?.[f.key] ?? '';
    }
    return init;
  });
  const [revealed, setRevealed] = useState<Record<string, boolean>>({});
  const [copied, setCopied] = useState(false);

  const save = useMutation({
    mutationFn: () => {
      const extra: Record<string, string> = {};
      for (const f of extraFields) extra[f.key] = (values[f.key] ?? '').trim();
      return setOAuthProvider(projectId, provider.id, {
        clientId: (idField ? values[idField.key] : '')?.trim() ?? '',
        clientSecret: secretField ? (values[secretField.key] ?? '') : '',
        extra: extraFields.length > 0 ? extra : undefined,
        enabled,
      });
    },
    onSuccess: onSaved,
  });

  const remove = useMutation({
    mutationFn: () => deleteOAuthProvider(projectId, provider.id),
    onSuccess: onSaved,
  });

  // Must match the backend route: GET /account/sessions/oauth/{provider}/callback
  // (auth/handler.go). No `oauth2`, and no projectId segment — this is the exact
  // redirect URI users register with the provider.
  const redirectUri = useMemo(
    () => `https://your-domain.com/v1/account/sessions/oauth/${provider.id}/callback`,
    [provider.id],
  );
  const note =
    provider.setupNote ??
    `To complete set up, add this OAuth2 redirect URI to your ${provider.name} app configuration.`;

  const copyUri = async () => {
    try {
      await navigator.clipboard.writeText(redirectUri);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };

  const configured = Boolean(config?.clientId);
  const enableWithoutId = enabled && idField && !(values[idField.key] ?? '').trim();

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`${provider.name} OAuth2 settings`}
      subtitle="To use this authentication provider in your application, first fill in this form."
      submitLabel="Update"
      width={540}
      loading={save.isPending}
      submitDisabled={Boolean(enableWithoutId) || save.isPending || remove.isPending}
      onSubmit={() => save.mutate()}
    >
      {/* Enabled toggle */}
      <div className="flex items-center gap-2.5">
        <Switch checked={enabled} onCheckedChange={setEnabled} />
        <span
          className="text-[length:var(--text-control)] font-medium"
          style={{ color: enabled ? 'var(--text-primary)' : 'var(--text-muted)' }}
        >
          {enabled ? 'Enabled' : 'Disabled'}
        </span>
      </div>

      {/* Provider fields */}
      {provider.fields.map((f) => {
        const type = f.type ?? 'text';
        if (type === 'multiline') {
          return (
            <FormField key={f.key} label={f.label}>
              <Textarea
                placeholder={f.hint}
                rows={5}
                value={values[f.key] ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [f.key]: e.target.value }))}
              />
            </FormField>
          );
        }
        const isSecret = type === 'secret';
        const shown = revealed[f.key] ?? false;
        const secretHint = isSecret && configured ? 'Leave blank to keep the stored secret' : undefined;
        return (
          <FormField key={f.key} label={f.label} hint={secretHint}>
            <div className="relative">
              <Input
                type={isSecret && !shown ? 'password' : 'text'}
                placeholder={f.hint}
                value={values[f.key] ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [f.key]: e.target.value }))}
                className={isSecret ? 'pr-9' : undefined}
              />
              {isSecret && (
                <button
                  type="button"
                  onClick={() => setRevealed((r) => ({ ...r, [f.key]: !shown }))}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-subtle hover:text-text-secondary"
                  aria-label={shown ? 'Hide' : 'Show'}
                >
                  {shown ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              )}
            </div>
          </FormField>
        );
      })}

      {(save.error || remove.error) && (
        <div className="text-[length:var(--text-caption)] text-[var(--color-danger)]">
          {friendlyError(save.error ?? remove.error)}
        </div>
      )}

      {/* Info box */}
      <div
        className="flex items-start gap-2.5 rounded-[var(--radius)] p-3.5"
        style={{
          backgroundColor: 'color-mix(in srgb, var(--color-accent) 7%, transparent)',
          border: '1px solid color-mix(in srgb, var(--color-accent) 20%, transparent)',
        }}
      >
        <Info size={14} className="mt-0.5 shrink-0" style={{ color: 'var(--color-accent)' }} />
        <span className="text-[length:var(--text-label)] leading-relaxed text-text-secondary">{note}</span>
      </div>

      {/* Redirect URI */}
      <FormField label="URI">
        <div className="relative">
          <Input readOnly value={redirectUri} className="bg-surface-alt pr-9 text-text-muted" />
          <button
            type="button"
            onClick={copyUri}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-subtle hover:text-text-secondary"
            aria-label="Copy redirect URI"
          >
            {copied ? <Check size={14} className="text-status-success" /> : <Copy size={14} />}
          </button>
        </div>
      </FormField>

      {/* Remove a stored configuration entirely. */}
      {configured && (
        <button
          type="button"
          onClick={() => remove.mutate()}
          disabled={remove.isPending}
          className="flex items-center gap-1.5 self-start text-[length:var(--text-caption)] text-[var(--color-danger)] hover:underline disabled:opacity-50"
        >
          <Trash2 size={13} /> Remove configuration
        </button>
      )}
    </FormDialog>
  );
}
