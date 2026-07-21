import { useState } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, friendlyError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { Button } from '@/components/ui/button';
import { toast } from '@/components/toast';

/*
 * Invite redemption.
 *
 * Separate from signup on purpose: the token in the URL is what grants the
 * account, and the address comes from the invite rather than from whoever is
 * filling in the form. That is why this works on a private instance where
 * registration is closed.
 */

export function InvitePage() {
  const { token = '' } = useParams();
  const navigate = useNavigate();
  const loginWithToken = useAuthStore((s) => s.loginWithToken);

  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  const {
    data: invite,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['invite', token],
    queryFn: async () => {
      const res = await api.get(`/console/invites/${token}`);
      return res.data as {
        email: string;
        name: string;
        role: string;
        organizationName: string;
        hasAccount: boolean;
      };
    },
    retry: false,
  });

  const redeem = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await api.post(`/console/invites/${token}/redeem`, { name, password });
      const { token: jwt } = res.data as { token: string };
      await loginWithToken(jwt);
      navigate('/projects', { replace: true });
    } catch (err) {
      toast.error(friendlyError(err));
      setBusy(false);
    }
  };

  if (isLoading) {
    return <Centered>Checking invite…</Centered>;
  }

  if (isError || !invite) {
    return (
      <Centered>
        <h1 className="text-[20px] font-semibold text-text-primary">This invite is no longer valid</h1>
        <p className="mt-2 text-[13px] text-text-muted">
          It may have already been used, or been withdrawn. Ask whoever invited you to send another.
        </p>
        <Link to="/login" className="mt-6 text-[13px] text-[var(--color-accent)] hover:underline">
          Go to sign in
        </Link>
      </Centered>
    );
  }

  // Already registered: the invite is accepted from inside the console, so
  // there is nothing to create here.
  if (invite.hasAccount) {
    return (
      <Centered>
        <h1 className="text-[20px] font-semibold text-text-primary">
          You already have an account
        </h1>
        <p className="mt-2 text-[13px] text-text-muted">
          Sign in as <span className="text-text-primary">{invite.email}</span> to join{' '}
          {invite.organizationName}.
        </p>
        <Link to="/login" className="mt-6 text-[13px] text-[var(--color-accent)] hover:underline">
          Go to sign in
        </Link>
      </Centered>
    );
  }

  return (
    <Centered>
      <h1 className="text-[20px] font-semibold text-text-primary">
        Join {invite.organizationName}
      </h1>
      <p className="mt-2 text-[13px] text-text-muted">
        Invited as <span className="text-text-primary">{invite.email}</span> · {invite.role}
      </p>

      <form onSubmit={redeem} className="mt-6 flex w-full flex-col gap-4 text-left">
        <label className="flex flex-col gap-1.5">
          <span className="text-[length:var(--text-label)] text-text-secondary">Your name</span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={invite.name || 'Your name'}
            autoFocus
            className="h-9 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 text-[length:var(--text-body)] text-text-primary outline-none focus:border-[var(--color-accent)]"
          />
        </label>
        <label className="flex flex-col gap-1.5">
          <span className="text-[length:var(--text-label)] text-text-secondary">
            Choose a password
          </span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="At least 8 characters"
            autoComplete="new-password"
            className="h-9 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 text-[length:var(--text-body)] text-text-primary outline-none focus:border-[var(--color-accent)]"
          />
        </label>
        <Button type="submit" loading={busy} disabled={password.length < 8}>
          Accept invite
        </Button>
      </form>
    </Centered>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-6">
      <div className="flex w-full max-w-[360px] flex-col items-center text-center">{children}</div>
    </div>
  );
}
