import { useMemo, useState } from 'react';
import { Apple, Check, Copy, Eye, EyeOff, GitBranch, Info, Music, type LucideIcon } from 'lucide-react';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { FormDialog, FormField } from '@/components/form-dialog';
import { AUTH_METHODS, OAUTH_PROVIDERS, type OAuthProvider } from './auth-config';

export function SettingsTab({ projectId }: { projectId: string }) {
  const [methodState, setMethodState] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(AUTH_METHODS.map((m) => [m.id, Boolean(m.defaultOn)])),
  );
  const [oauthState, setOauthState] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(OAUTH_PROVIDERS.map((p) => [p.id, false])),
  );
  const [configuring, setConfiguring] = useState<OAuthProvider | null>(null);

  return (
    <div className="pb-10">
      {/* Auth methods */}
      <SectionHeader title="Auth methods" subtitle="Enable the authentication methods you wish to use." />
      <div className="mt-4 grid grid-cols-1 divide-y divide-border overflow-hidden rounded-[var(--radius)] border border-border bg-surface md:grid-cols-2 md:divide-y-0">
        {AUTH_METHODS.map((m) => {
          const enabled = methodState[m.id] ?? false;
          const Icon = m.icon;
          return (
            <div
              key={m.id}
              className="flex items-center gap-3 border-border px-4 py-3.5 max-md:border-b max-md:last:border-b-0 md:[&:nth-child(odd)]:border-r md:[&:not(:nth-last-child(-n+2))]:border-b"
            >
              <div
                className="flex h-[30px] w-[30px] items-center justify-center rounded-[var(--radius-6)]"
                style={{
                  backgroundColor: enabled ? 'color-mix(in srgb, var(--color-accent) 12%, transparent)' : 'var(--fill)',
                  color: enabled ? 'var(--color-accent)' : 'var(--text-subtle)',
                }}
              >
                <Icon size={14} />
              </div>
              <div className="flex-1">
                <div className="text-[length:var(--text-body)] font-medium text-text-primary">{m.label}</div>
                <div className="text-[length:var(--text-caption)] text-text-subtle">{m.description}</div>
              </div>
              <Switch
                checked={enabled}
                onCheckedChange={(v) => setMethodState((s) => ({ ...s, [m.id]: v }))}
              />
            </div>
          );
        })}
      </div>

      {/* OAuth providers */}
      <div className="mt-8">
        <SectionHeader
          title="OAuth2 Providers"
          subtitle="Allow users to sign in with their existing third-party accounts."
        />
      </div>
      <div className="mt-4 grid grid-cols-2 gap-2.5 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5">
        {OAUTH_PROVIDERS.map((p) => (
          <ProviderCard
            key={p.id}
            provider={p}
            enabled={oauthState[p.id] ?? false}
            onToggle={(v) => setOauthState((s) => ({ ...s, [p.id]: v }))}
            onConfigure={() => setConfiguring(p)}
          />
        ))}
      </div>

      {configuring && (
        <OAuthConfigDialog
          provider={configuring}
          projectId={projectId}
          initialEnabled={oauthState[configuring.id] ?? false}
          onClose={() => setConfiguring(null)}
          onSave={(enabled) => {
            setOauthState((s) => ({ ...s, [configuring.id]: enabled }));
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
  onToggle,
  onConfigure,
}: {
  provider: OAuthProvider;
  enabled: boolean;
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
            onToggle(!enabled);
          }}
          className="ml-auto flex h-[18px] w-8 items-center rounded-full border border-border p-[3px] transition-colors"
          style={{ backgroundColor: enabled ? 'var(--color-accent)' : 'var(--fill)' }}
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
  initialEnabled,
  onClose,
  onSave,
}: {
  provider: OAuthProvider;
  projectId: string;
  initialEnabled: boolean;
  onClose: () => void;
  onSave: (enabled: boolean) => void;
}) {
  const [enabled, setEnabled] = useState(initialEnabled);
  const [values, setValues] = useState<Record<string, string>>({});
  const [revealed, setRevealed] = useState<Record<string, boolean>>({});
  const [copied, setCopied] = useState(false);

  // Must match the backend route: GET /account/sessions/oauth/{provider}/callback
  // (auth/handler.go). No `oauth2`, and no projectId segment — this is the exact
  // redirect URI users register with the provider.
  const redirectUri = useMemo(
    () => `https://your-domain.com/v1/account/sessions/oauth/${provider.id}/callback`,
    [provider.id, projectId],
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

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`${provider.name} OAuth2 settings`}
      subtitle="To use this authentication provider in your application, first fill in this form."
      submitLabel="Update"
      width={540}
      onSubmit={() => onSave(enabled)}
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
        return (
          <FormField key={f.key} label={f.label}>
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
    </FormDialog>
  );
}
