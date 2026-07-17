import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { Activity, Eye, EyeOff, KeyRound, Monitor, ShieldAlert, ShieldCheck } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { useOrgs } from '@/api/queries';
import { StandaloneLayout } from '@/shell/StandaloneLayout';
import { PageTabs } from '@/components/page-tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { ConfirmDialog } from '@/components/form-dialog';
import { useTabIndex } from '@/hooks/use-tab-param';

const TABS = ['Overview', 'Sessions', 'Activity', 'Organizations'];

export function AccountPage() {
  const [tab, setTab] = useTabIndex(TABS);
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const [confirmSignOut, setConfirmSignOut] = useState(false);

  return (
    <StandaloneLayout showOrg={false}>
      <div className="mx-auto w-full max-w-3xl flex-1 px-6 py-8">
        <div className="flex items-center justify-between">
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
            {user?.name || 'Account'}
          </h1>
          <Button variant="ghost" size="sm" onClick={() => setConfirmSignOut(true)}>
            Sign out
          </Button>
        </div>
        <div className="mt-6">
          <PageTabs tabs={TABS} selected={tab} onChange={setTab} />
        </div>
        <div className="mt-6">
          {tab === 0 && <OverviewTab />}
          {tab === 1 && <PlaceholderTab icon={Monitor} text="Your active sign-in sessions will appear here." />}
          {tab === 2 && <PlaceholderTab icon={Activity} text="Your account activity will appear here." />}
          {tab === 3 && <OrganizationsTab />}
        </div>
      </div>
      <ConfirmDialog
        open={confirmSignOut}
        onOpenChange={setConfirmSignOut}
        title="Sign out"
        message="Are you sure you want to sign out?"
        confirmLabel="Sign out"
        destructive={false}
        onConfirm={() => {
          void logout();
          navigate('/login');
        }}
      />
    </StandaloneLayout>
  );
}

function OverviewTab() {
  const user = useAuthStore((s) => s.user);
  const init = useAuthStore((s) => s.init);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const [name, setName] = useState(user?.name ?? '');
  const [email, setEmail] = useState(user?.email ?? '');
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [mfaEnabled, setMfaEnabled] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [msg, setMsg] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);

  useEffect(() => {
    setName(user?.name ?? '');
    setEmail(user?.email ?? '');
  }, [user]);

  const saveProfile = useMutation({
    mutationFn: async () => {
      await api.patch('/console/me', { name, email });
    },
    onSuccess: () => {
      void init();
      setMsg({ kind: 'ok', text: 'Profile updated.' });
    },
    onError: (e) => setMsg({ kind: 'err', text: friendlyError(e) }),
  });

  const savePassword = useMutation({
    mutationFn: async () => {
      await api.patch('/console/me/password', { oldPassword: currentPassword, password: newPassword });
    },
    onSuccess: () => {
      setCurrentPassword('');
      setNewPassword('');
      setMsg({ kind: 'ok', text: 'Password changed.' });
    },
    onError: (e) => setMsg({ kind: 'err', text: friendlyError(e) }),
  });

  const del = useMutation({
    mutationFn: () => api.delete('/console/me'),
    onSuccess: () => {
      void logout();
      navigate('/login');
    },
  });

  return (
    <div className="flex flex-col gap-6">
      {msg && (
        <div
          className={`rounded-[var(--radius)] px-3 py-2 text-[length:var(--text-caption)] ${
            msg.kind === 'ok'
              ? 'bg-[color-mix(in_srgb,var(--status-success)_10%,transparent)] text-[var(--status-success)]'
              : 'bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] text-[var(--status-danger)]'
          }`}
        >
          {msg.text}
        </div>
      )}

      <Card title="Profile" subtitle="Update your display name and email address.">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Email</Label>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="flex justify-end">
            <Button loading={saveProfile.isPending} onClick={() => saveProfile.mutate()}>
              Save changes
            </Button>
          </div>
        </div>
      </Card>

      <Card
        title="Password"
        icon={KeyRound}
        subtitle="Choose a strong password you don't use elsewhere."
      >
        <div className="flex flex-col gap-4">
          <PasswordField label="Old password" value={currentPassword} onChange={setCurrentPassword} />
          <PasswordField label="New password" value={newPassword} onChange={setNewPassword} />
          <div className="flex justify-end">
            <Button
              loading={savePassword.isPending}
              disabled={!currentPassword || !newPassword}
              onClick={() => savePassword.mutate()}
            >
              Update
            </Button>
          </div>
        </div>
      </Card>

      <Card
        title="Multi-factor authentication"
        icon={ShieldCheck}
        subtitle="Enhance your account's security by requiring a second sign-in method."
      >
        <div className="flex items-center gap-2.5">
          <Switch checked={mfaEnabled} onCheckedChange={setMfaEnabled} />
          <span className="text-[length:var(--text-body)] text-text-primary">
            Multi-factor authentication
          </span>
        </div>
      </Card>

      <Card
        title="Delete account"
        icon={ShieldAlert}
        danger
        subtitle="Your account will be permanently deleted and access will be lost to all your teams and data. This action is irreversible."
      >
        <div className="flex flex-col items-end gap-3">
          <div className="flex w-full items-center gap-2.5 rounded-[var(--radius)] border border-border bg-surface-alt px-3 py-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-full bg-[var(--color-accent)] text-[length:var(--text-caption)] font-semibold text-white">
              {initials(user?.name, user?.email)}
            </div>
            <span className="text-[length:var(--text-body)] text-text-primary">
              {user?.name || user?.email}
            </span>
          </div>
          <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
            Delete
          </Button>
        </div>
      </Card>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete account"
        message="This permanently deletes your account. This cannot be undone."
        confirmLabel="Delete account"
        loading={del.isPending}
        onConfirm={() => del.mutate()}
      />
    </div>
  );
}

function OrganizationsTab() {
  const { data: orgs = [] } = useOrgs();
  const navigate = useNavigate();
  return (
    <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
      {orgs.length === 0 && (
        <div className="px-4 py-10 text-center text-[length:var(--text-body)] text-text-muted">
          You're not a member of any organizations.
        </div>
      )}
      {orgs.map((o) => (
        <button
          key={o.$id}
          onClick={() => navigate(`/org/${o.$id}/projects`)}
          className="flex w-full items-center justify-between border-b border-[var(--fill)] px-4 py-3 text-left last:border-0 hover:bg-fill"
        >
          <span className="text-[length:var(--text-body)] text-text-primary">{o.name}</span>
          <span className="text-[length:var(--text-caption)] text-text-subtle">View →</span>
        </button>
      ))}
    </div>
  );
}

/* Two-column account section (ports account_page.dart _AccountSection):
 * title + subtitle on the left, controls on the right. */
function Card({
  title,
  subtitle,
  icon: Icon,
  danger,
  children,
}: {
  title: string;
  subtitle?: string;
  icon?: typeof KeyRound;
  danger?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div
      className={`rounded-[var(--radius-10)] border bg-surface p-5 md:p-6 ${
        danger ? 'border-[color-mix(in_srgb,var(--color-danger)_40%,var(--border))]' : 'border-border'
      }`}
    >
      <div className="flex flex-col gap-5 md:flex-row md:gap-8">
        <div className="md:w-2/5">
          <div
            className={`flex items-center gap-2 text-[length:var(--text-control)] font-medium ${
              danger ? 'text-[var(--status-danger)]' : 'text-text-primary'
            }`}
          >
            {Icon && <Icon size={16} />}
            {title}
          </div>
          {subtitle && (
            <div className="mt-2 text-[length:var(--text-body)] text-text-muted">{subtitle}</div>
          )}
        </div>
        <div className="flex-1">{children}</div>
      </div>
    </div>
  );
}

/* Password field with a show/hide (eye) toggle. */
function PasswordField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  const [show, setShow] = useState(false);
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      <div className="relative">
        <Input
          type={show ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Enter password"
          className="pr-9"
        />
        <button
          type="button"
          onClick={() => setShow((v) => !v)}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-text-subtle transition-colors hover:text-text-primary"
          aria-label={show ? 'Hide password' : 'Show password'}
        >
          {show ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      </div>
    </div>
  );
}

function initials(name?: string, email?: string): string {
  const base = (name || email || '?').trim();
  const parts = base.split(/[\s@.]+/).filter(Boolean);
  return ((parts[0]?.[0] ?? '?') + (parts[1]?.[0] ?? '')).toUpperCase();
}

function PlaceholderTab({ icon: Icon, text }: { icon: typeof Monitor; text: string }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-[var(--radius-10)] border border-dashed border-border py-16 text-center">
      <Icon size={24} className="text-text-subtle" />
      <div className="text-[length:var(--text-body)] text-text-muted">{text}</div>
    </div>
  );
}
