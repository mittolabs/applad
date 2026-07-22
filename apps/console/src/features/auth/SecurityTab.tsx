import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { ErrorState } from '@/components/error-state';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';

type SecurityConfig = Record<string, unknown>;

const NUMBER_FIELDS: { key: string; label: string; description: string; def: number }[] = [
  { key: 'usersLimit', label: 'Users limit', description: 'Maximum number of users that can sign up. Set to 0 for unlimited.', def: 0 },
  { key: 'sessionLengthSeconds', label: 'Session length (seconds)', description: 'How long sessions remain valid before expiring.', def: 31536000 },
  { key: 'sessionsPerUser', label: 'Sessions per user', description: 'Maximum concurrent active sessions per user. Set to 0 for unlimited.', def: 10 },
  { key: 'passwordMinLength', label: 'Password minimum length', description: 'Minimum number of characters required for passwords.', def: 8 },
  { key: 'passwordHistory', label: 'Password history', description: 'Number of previous passwords to remember and disallow reuse. Set to 0 to disable.', def: 0 },
];

const TOGGLE_FIELDS: { key: string; label: string; description: string; def: boolean }[] = [
  { key: 'passwordDictionary', label: 'Password dictionary check', description: 'Reject commonly used or compromised passwords.', def: false },
  { key: 'passwordPersonalData', label: 'Personal data check', description: "Reject passwords that contain the user's name or email.", def: false },
  { key: 'mfaRequired', label: 'Require MFA', description: 'Require multi-factor authentication for all users.', def: false },
  { key: 'sessionAlerts', label: 'Session alerts', description: 'Send an email to the user when a new session is created.', def: false },
  { key: 'invalidateOnPasswordChange', label: 'Invalidate sessions on password change', description: 'Immediately revoke all active sessions when a user changes their password.', def: true },
];

export function SecurityTab({ projectId }: { projectId: string }) {
  const query = useQuery({
    queryKey: ['auth-security', projectId],
    queryFn: () =>
      api.get(`/projects/${projectId}/auth/security`).then((r) => r.data as SecurityConfig),
  });

  const [config, setConfig] = useState<SecurityConfig | null>(null);
  useEffect(() => {
    if (query.data) setConfig(query.data);
  }, [query.data]);

  const save = useMutation({
    mutationFn: (patch: SecurityConfig) => {
      const merged = { ...(config ?? {}), ...patch };
      return api.put(`/projects/${projectId}/auth/security`, merged).then((r) => r.data as SecurityConfig);
    },
    onSuccess: (data) => setConfig(data),
  });

  if (query.error) return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  if (!config) {
    return <div className="h-40 animate-pulse rounded-[var(--radius-10)] border border-border bg-surface" />;
  }

  const num = (k: string, def: number): number => {
    const v = config[k];
    if (typeof v === 'number') return v;
    const parsed = Number(v);
    return Number.isFinite(parsed) && v != null ? parsed : def;
  };
  const bool = (k: string, def: boolean): boolean =>
    typeof config[k] === 'boolean' ? (config[k] as boolean) : def;

  return (
    <div className="pb-8">
      <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">Security</h2>
      <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
        Configure security policies for your project.
      </p>

      <div className="mt-5 flex flex-col gap-2">
        {NUMBER_FIELDS.map((f) => (
          <NumberCard
            key={f.key}
            label={f.label}
            description={f.description}
            value={num(f.key, f.def)}
            saving={save.isPending}
            onSave={(v) => save.mutate({ [f.key]: v })}
          />
        ))}
        {TOGGLE_FIELDS.map((f) => (
          <ToggleCard
            key={f.key}
            label={f.label}
            description={f.description}
            value={bool(f.key, f.def)}
            onChange={(v) => save.mutate({ [f.key]: v })}
          />
        ))}
      </div>
    </div>
  );
}

function CardShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface p-4">
      {children}
    </div>
  );
}

function NumberCard({
  label,
  description,
  value,
  saving,
  onSave,
}: {
  label: string;
  description: string;
  value: number;
  saving: boolean;
  onSave: (value: number) => void;
}) {
  const [text, setText] = useState(String(value));
  useEffect(() => setText(String(value)), [value]);

  return (
    <CardShell>
      <div className="flex-1">
        <div className="text-[length:var(--text-control)] font-medium text-text-primary">{label}</div>
        <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">{description}</div>
      </div>
      <Input
        value={text}
        inputMode="numeric"
        onChange={(e) => setText(e.target.value.replace(/[^0-9]/g, ''))}
        className="h-9 w-24"
      />
      <Button
        variant="ghost"
        size="sm"
        disabled={saving}
        onClick={() => onSave(Number(text) || value)}
        style={{ color: 'var(--color-accent)' }}
      >
        Save
      </Button>
    </CardShell>
  );
}

function ToggleCard({
  label,
  description,
  value,
  onChange,
}: {
  label: string;
  description: string;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <CardShell>
      <div className="flex-1">
        <div className="text-[length:var(--text-control)] font-medium text-text-primary">{label}</div>
        <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">{description}</div>
      </div>
      <Switch checked={value} onCheckedChange={onChange} />
    </CardShell>
  );
}
