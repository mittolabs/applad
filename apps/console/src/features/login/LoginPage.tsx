import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  AlertCircle,
  Building2,
  CheckCircle2,
  Eye,
  EyeOff,
  GitBranch,
  Info,
  Loader2,
  LogIn,
} from 'lucide-react';
import { isAxiosError } from 'axios';
import { api, friendlyError } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';

/*
 * Login — faithful port of console/lib/features/login/login_page.dart.
 * Wide (>900px): 5:4 branding / form split with an abstract painted panel.
 * Narrow: form only, with the mascot logo row at the top.
 * Four modes: login, signup, forgot, reset — same flow + copy as Flutter.
 */

type Mode = 'login' | 'signup' | 'forgot' | 'reset';

const ACCENT = '#3472A4';
const VERSION = import.meta.env.VITE_APP_VERSION ?? '0.2.0';

const OAUTH_ERRORS: Record<string, string> = {
  signup_disabled: 'Account creation is disabled. Contact your administrator.',
  oauth_cancelled: 'Sign-in was cancelled.',
  oauth_unavailable: "That sign-in method isn't configured. Contact your administrator.",
  oauth_failed: 'OAuth sign-in failed. Please try again.',
};

export function LoginPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const login = useAuthStore((s) => s.login);
  const signup = useAuthStore((s) => s.signup);
  const loginWithToken = useAuthStore((s) => s.loginWithToken);

  // ?mode=signup lets the marketing site link straight to account creation.
  // If signup turns out to be disabled, the effect below drops back to login.
  const [mode, setMode] = useState<Mode>(params.get('mode') === 'signup' ? 'signup' : 'login');
  const [oauthLoading, setOauthLoading] = useState(false);
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [newPass, setNewPass] = useState('');
  const [confirm, setConfirm] = useState('');
  const [resetToken, setResetToken] = useState('');
  const [showPw, setShowPw] = useState(false);
  const [showNewPw, setShowNewPw] = useState(false);
  const [policyAccepted, setPolicyAccepted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [surfacedToken, setSurfacedToken] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  // Set once the backend answers a correct password with console_mfa_required:
  // the form then asks for the authenticator code and resubmits with it.
  const [mfaRequired, setMfaRequired] = useState(false);
  const [mfaCode, setMfaCode] = useState('');

  const isSignup = mode === 'signup';

  /*
   * How this instance handles new accounts. The hosted service runs with
   * signup open; a self-hosted one closes it behind the first account, which
   * is the "auto" default. firstRun means nobody has registered yet, so the
   * next account owns the instance. inviteOnly means registration is closed
   * and new accounts come from an invite link, which is redeemed at
   * /invite/:token rather than through this form.
   */
  const { data: signupStatus } = useQuery({
    queryKey: ['signup-status'],
    queryFn: async () => {
      const res = await api.get('/console/signup-status');
      const d = res.data as { signupEnabled?: boolean; firstRun?: boolean; inviteOnly?: boolean };
      return {
        signupEnabled: Boolean(d.signupEnabled ?? true),
        firstRun: Boolean(d.firstRun),
        inviteOnly: Boolean(d.inviteOnly),
      };
    },
  });
  const signupEnabled = signupStatus?.signupEnabled ?? true;
  const firstRun = signupStatus?.firstRun ?? false;
  const { data: providers = [] } = useQuery({
    queryKey: ['auth-providers'],
    queryFn: async () => {
      const res = await api.get('/console/auth-providers');
      return ((res.data as { providers?: string[] }).providers ?? []) as string[];
    },
  });

  // A fresh instance opens on account creation: that first account owns it.
  useEffect(() => {
    if (firstRun) setMode('signup');
  }, [firstRun]);

  // Registration closed means closed. Someone who was invited does not come
  // through this form at all — they follow the link in their invite, which
  // carries the token that creates their account.
  useEffect(() => {
    if (mode === 'signup' && !signupEnabled) setMode('login');
  }, [mode, signupEnabled]);

  // Handle OAuth / reset callbacks once on mount.
  useEffect(() => {
    const resetTok = params.get('reset_token');
    const consoleTok = params.get('console_token');
    const err = params.get('error');
    if (resetTok) {
      setMode('reset');
      setResetToken(resetTok);
      clearParams();
    } else if (consoleTok) {
      clearParams();
      setOauthLoading(true);
      void loginWithToken(consoleTok)
        .then(() => navigate('/onboarding'))
        .catch(() => {
          setError('OAuth sign-in failed. Please try again.');
          setOauthLoading(false);
        });
    } else if (err) {
      setError(OAUTH_ERRORS[err] ?? 'OAuth sign-in failed. Please try again.');
      clearParams();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function clearParams() {
    const p = new URLSearchParams(params);
    ['reset_token', 'console_token', 'error'].forEach((k) => p.delete(k));
    setParams(p, { replace: true });
  }

  const oauth = (provider: string) => {
    window.location.href = `/v1/console/auth/${provider}`;
  };

  const goMode = (m: Mode) => {
    setMode(m);
    setError(null);
    setSuccess(null);
    setSurfacedToken(null);
    setMfaRequired(false);
    setMfaCode('');
    if (m === 'signup') setPolicyAccepted(false);
  };

  const submitLogin = async () => {
    setError(null);
    setBusy(true);
    try {
      if (isSignup) {
        await signup(email, password, name);
        navigate('/onboarding');
      } else {
        await login(email, password, mfaRequired ? mfaCode : undefined);
        navigate('/projects');
      }
    } catch (e) {
      // A correct password on an MFA-enrolled account comes back as
      // console_mfa_required: reveal the code field instead of showing an
      // error. A bad code comes back as console_mfa_invalid.
      const type = isAxiosError(e)
        ? (e.response?.data as { type?: string } | undefined)?.type
        : undefined;
      if (type === 'console_mfa_required') {
        setMfaRequired(true);
        setError(null);
      } else if (type === 'console_mfa_invalid') {
        setMfaRequired(true);
        setError('Invalid authentication code. Try again.');
      } else {
        setError(friendlyError(e));
      }
    } finally {
      setBusy(false);
    }
  };

  const submitForgot = async () => {
    if (!email.trim()) {
      setError('Please enter your email address.');
      return;
    }
    setBusy(true);
    setError(null);
    setSuccess(null);
    setSurfacedToken(null);
    try {
      const res = await api.post('/console/password-reset/request', { email });
      const data = res.data as { emailSent?: boolean; token?: string };
      if (data.emailSent) setSuccess('Reset link sent — check your inbox.');
      else if (data.token) setSurfacedToken(data.token);
      else setSuccess('If that email is registered, a reset link has been sent.');
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    if (!resetToken.trim()) {
      setError('Please enter the reset token.');
      return;
    }
    if (newPass.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    if (newPass !== confirm) {
      setError('Passwords do not match.');
      return;
    }
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      await api.post('/console/password-reset/confirm', { token: resetToken, password: newPass });
      setSuccess('Password updated. You can now sign in.');
      setNewPass('');
      setConfirm('');
      setResetToken('');
      setTimeout(() => goMode('login'), 2000);
    } catch (e) {
      const msg = friendlyError(e);
      setError(
        /invalid|expired/i.test(msg)
          ? 'Invalid or expired token. Please request a new one.'
          : 'Something went wrong. Please try again.',
      );
    } finally {
      setBusy(false);
    }
  };

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (mode === 'forgot') void submitForgot();
    else if (mode === 'reset') void submitReset();
    else void submitLogin();
  };

  return (
    <div className="flex min-h-screen bg-background">
      {/* Branding panel — wide screens only, 5:4 with the form. */}
      <div className="relative hidden overflow-hidden border-r border-white/[0.06] min-[900px]:flex min-[900px]:flex-[5]">
        <PanelShapes />
        <div className="relative z-10 flex flex-col justify-center pb-12 pl-[120px] pr-14 pt-12">
          <LogoRow size={52} wordmark={28} />
          <h1 className="mt-12 text-[52px] font-bold leading-[1.15] tracking-[-0.01em] text-white/[0.88]">
            {isSignup ? 'Build products,' : 'Go from idea'}
            <br />
            {isSignup ? 'not infrastructure.' : 'to production today.'}
          </h1>
          <p className="mt-4 text-[15px] leading-[1.5] text-white/[0.35]">
            {isSignup
              ? 'Plan, build, test, deploy and monitor in one platform.'
              : 'Everything your app needs, without compromise.'}
          </p>
        </div>
      </div>

      {/* Form panel. */}
      <div className="flex flex-1 flex-col min-[900px]:flex-[4]">
        <div className="flex flex-1 items-center justify-center overflow-y-auto px-6 py-12 min-[900px]:px-12">
          <div className="w-full max-w-[380px]">
            {oauthLoading ? (
              <LoadingOverlay />
            ) : mode === 'forgot' ? (
              <ForgotForm
                email={email}
                setEmail={setEmail}
                busy={busy}
                error={error}
                success={success}
                surfacedToken={surfacedToken}
                onSubmit={onSubmit}
                onUseToken={() => {
                  setResetToken(surfacedToken!);
                  goMode('reset');
                  setResetToken(surfacedToken!);
                }}
                onBack={() => goMode('login')}
              />
            ) : mode === 'reset' ? (
              <ResetForm
                resetToken={resetToken}
                setResetToken={setResetToken}
                newPass={newPass}
                setNewPass={setNewPass}
                confirm={confirm}
                setConfirm={setConfirm}
                showNewPw={showNewPw}
                setShowNewPw={setShowNewPw}
                busy={busy}
                error={error}
                success={success}
                onSubmit={onSubmit}
                onBack={() => goMode('login')}
              />
            ) : (
              <LoginSignupForm
                isSignup={isSignup}
                signupEnabled={signupEnabled}
                firstRun={firstRun}
                providers={providers}
                name={name}
                setName={setName}
                email={email}
                setEmail={setEmail}
                password={password}
                setPassword={setPassword}
                showPw={showPw}
                setShowPw={setShowPw}
                mfaRequired={mfaRequired}
                mfaCode={mfaCode}
                setMfaCode={setMfaCode}
                policyAccepted={policyAccepted}
                setPolicyAccepted={setPolicyAccepted}
                busy={busy}
                error={error}
                onSubmit={onSubmit}
                onOauth={oauth}
                onMode={goMode}
              />
            )}
          </div>
        </div>
        <div className="pb-4 text-center text-[11px] text-text-subtle">v{VERSION}</div>
      </div>
    </div>
  );
}

// ── Login / Sign-up form ─────────────────────────────────────────────────────

function LoginSignupForm({
  isSignup,
  signupEnabled,
  firstRun,
  providers,
  name,
  setName,
  email,
  setEmail,
  password,
  setPassword,
  showPw,
  setShowPw,
  mfaRequired,
  mfaCode,
  setMfaCode,
  policyAccepted,
  setPolicyAccepted,
  busy,
  error,
  onSubmit,
  onOauth,
  onMode,
}: {
  isSignup: boolean;
  signupEnabled: boolean;
  firstRun: boolean;
  providers: string[];
  name: string;
  setName: (v: string) => void;
  email: string;
  setEmail: (v: string) => void;
  password: string;
  setPassword: (v: string) => void;
  showPw: boolean;
  setShowPw: (v: boolean) => void;
  mfaRequired: boolean;
  mfaCode: string;
  setMfaCode: (v: string) => void;
  policyAccepted: boolean;
  setPolicyAccepted: (v: boolean) => void;
  busy: boolean;
  error: string | null;
  onSubmit: (e: React.FormEvent) => void;
  onOauth: (p: string) => void;
  onMode: (m: Mode) => void;
}) {
  const extraProviders = providers.filter((p) => p !== 'github');
  return (
    <form onSubmit={onSubmit} className="flex flex-col">
      <LogoRow className="mb-7 min-[900px]:hidden" size={48} wordmark={24} />
      <Heading>{isSignup ? (firstRun ? 'Create the owner account' : 'Sign up') : 'Sign in'}</Heading>
      {firstRun && isSignup && (
        <p className="mt-2 text-[13px] leading-[1.5] text-text-muted">
          The first account owns this instance. Invite your team once you're in.
        </p>
      )}

      <div className="h-8" />

      {!isSignup && (
        <>
          <SocialButton provider="github" onClick={() => onOauth('github')} />
          {extraProviders.map((p) => (
            <SocialButton key={p} provider={p} onClick={() => onOauth(p)} />
          ))}
          <div className="h-2.5" />
          <OrDivider />
          <div className="h-2.5" />
        </>
      )}

      {isSignup && (
        <>
          <FieldLabel>Name</FieldLabel>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Your name" autoComplete="name" />
          <div className="h-5" />
        </>
      )}

      <FieldLabel>Email</FieldLabel>
      <Input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Your email"
        autoComplete="email"
        required
      />
      <div className="h-5" />

      <FieldLabel>Password</FieldLabel>
      <PasswordInput
        value={password}
        onChange={setPassword}
        show={showPw}
        onToggle={() => setShowPw(!showPw)}
        autoComplete={isSignup ? 'new-password' : 'current-password'}
      />

      {isSignup && (
        <div className="mt-2 flex items-center gap-1.5 text-[12px] text-text-subtle">
          <Info size={14} />
          Password must be at least 8 characters
        </div>
      )}

      {!isSignup && mfaRequired && (
        <>
          <div className="h-5" />
          <FieldLabel>Authentication code</FieldLabel>
          <Input
            value={mfaCode}
            onChange={(e) => setMfaCode(e.target.value.replace(/\s/g, ''))}
            placeholder="6-digit code or recovery code"
            autoComplete="one-time-code"
            inputMode="numeric"
            autoFocus
            required
          />
          <div className="mt-2 flex items-center gap-1.5 text-[12px] text-text-subtle">
            <Info size={14} />
            Enter the code from your authenticator app
          </div>
        </>
      )}

      {error && <ErrorBanner message={error} />}
      <div className="h-6" />

      {isSignup && (
        <>
          <PolicyCheckbox checked={policyAccepted} onChange={setPolicyAccepted} />
          <div className="h-4" />
        </>
      )}

      <SubmitButton busy={busy} disabled={isSignup && !policyAccepted}>
        {isSignup ? 'Sign up' : mfaRequired ? 'Verify code' : 'Sign in'}
      </SubmitButton>
      <div className="h-4" />

      {/* Links row */}
      <div className="flex items-center justify-center">
        {!isSignup && <TextLink onClick={() => onMode('forgot')}>Forgot password?</TextLink>}
        {!isSignup && signupEnabled && <span className="px-3 text-text-subtle">|</span>}
        {signupEnabled &&
          (isSignup ? (
            <TextLink onClick={() => onMode('login')}>Already got an account? Sign in</TextLink>
          ) : (
            <TextLink primary onClick={() => onMode('signup')}>
              Sign up
            </TextLink>
          ))}
      </div>

      {!isSignup && (
        <p className="mt-4 text-center text-[12px] leading-[1.5] text-text-subtle">
          By signing in, you agree to our <span className="underline">Terms</span> and{' '}
          <span className="underline">Privacy Policy</span>.
        </p>
      )}
    </form>
  );
}

// ── Forgot form ──────────────────────────────────────────────────────────────

function ForgotForm({
  email,
  setEmail,
  busy,
  error,
  success,
  surfacedToken,
  onSubmit,
  onUseToken,
  onBack,
}: {
  email: string;
  setEmail: (v: string) => void;
  busy: boolean;
  error: string | null;
  success: string | null;
  surfacedToken: string | null;
  onSubmit: (e: React.FormEvent) => void;
  onUseToken: () => void;
  onBack: () => void;
}) {
  return (
    <form onSubmit={onSubmit} className="flex flex-col">
      <LogoRow className="mb-7 min-[900px]:hidden" size={48} wordmark={24} />
      <Heading>Reset password</Heading>
      <p className="mt-2 text-[13px] leading-[1.5] text-text-muted">
        Enter your email and we&apos;ll send you a reset link.
      </p>
      <div className="h-7" />

      <FieldLabel>Email</FieldLabel>
      <Input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Your email"
        autoComplete="email"
      />

      {surfacedToken && (
        <div
          className="mt-5 rounded-[8px] border p-3.5"
          style={{
            backgroundColor: `color-mix(in srgb, ${ACCENT} 8%, transparent)`,
            borderColor: `color-mix(in srgb, ${ACCENT} 25%, transparent)`,
          }}
        >
          <div className="flex items-center gap-1.5 text-[12px] font-semibold text-text-secondary">
            <Info size={14} style={{ color: ACCENT }} />
            SMTP not configured — use this token:
          </div>
          <div
            className="mt-2 break-all font-[family-name:var(--font-mono)] text-[12px]"
            style={{ color: ACCENT }}
          >
            {surfacedToken}
          </div>
          <button
            type="button"
            onClick={onUseToken}
            className="mt-2 text-[12px] font-semibold underline"
            style={{ color: ACCENT }}
          >
            Use this token →
          </button>
        </div>
      )}

      {success && <SuccessBanner message={success} className="mt-4" />}
      {error && <ErrorBanner message={error} />}
      <div className="h-6" />

      <SubmitButton busy={busy}>Send reset link</SubmitButton>
      <div className="h-5" />
      <div className="text-center">
        <TextLink onClick={onBack}>Back to sign in</TextLink>
      </div>
    </form>
  );
}

// ── Reset form ───────────────────────────────────────────────────────────────

function ResetForm({
  resetToken,
  setResetToken,
  newPass,
  setNewPass,
  confirm,
  setConfirm,
  showNewPw,
  setShowNewPw,
  busy,
  error,
  success,
  onSubmit,
  onBack,
}: {
  resetToken: string;
  setResetToken: (v: string) => void;
  newPass: string;
  setNewPass: (v: string) => void;
  confirm: string;
  setConfirm: (v: string) => void;
  showNewPw: boolean;
  setShowNewPw: (v: boolean) => void;
  busy: boolean;
  error: string | null;
  success: string | null;
  onSubmit: (e: React.FormEvent) => void;
  onBack: () => void;
}) {
  return (
    <form onSubmit={onSubmit} className="flex flex-col">
      <LogoRow className="mb-7 min-[900px]:hidden" size={48} wordmark={24} />
      <Heading>Set new password</Heading>
      <p className="mt-2 text-[13px] text-text-muted">
        Enter your reset token and choose a new password.
      </p>
      <div className="h-7" />

      <FieldLabel>Reset token</FieldLabel>
      <Input
        value={resetToken}
        onChange={(e) => setResetToken(e.target.value)}
        placeholder="Paste your token here"
        autoComplete="one-time-code"
      />
      <div className="h-5" />

      <FieldLabel>New password</FieldLabel>
      <PasswordInput
        value={newPass}
        onChange={setNewPass}
        show={showNewPw}
        onToggle={() => setShowNewPw(!showNewPw)}
        autoComplete="new-password"
      />
      <div className="h-5" />

      <FieldLabel>Confirm password</FieldLabel>
      <Input
        type="password"
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
        placeholder="Repeat new password"
        autoComplete="new-password"
      />

      {success && <SuccessBanner message={success} className="mt-4" />}
      {error && <ErrorBanner message={error} />}
      <div className="h-6" />

      <SubmitButton busy={busy}>Set new password</SubmitButton>
      <div className="h-5" />
      <div className="text-center">
        <TextLink onClick={onBack}>Back to sign in</TextLink>
      </div>
    </form>
  );
}

// ── Shared pieces ────────────────────────────────────────────────────────────

function Heading({ children }: { children: React.ReactNode }) {
  return <h2 className="text-[28px] font-bold text-text-primary">{children}</h2>;
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <span className="mb-1.5 text-[13px] font-medium text-text-secondary">{children}</span>;
}

function LogoRow({
  size,
  wordmark,
  className,
}: {
  size: number;
  wordmark: number;
  className?: string;
}) {
  return (
    <div className={cn('flex items-center', className)}>
      {/* Radial glow BEHIND the robot so blue reaches its outline. The mascot art
       * is the robot on a transparent canvas, so a box-shadow would only light the
       * clip edge and leave a dark ring — a full radial gradient fills up to the
       * robot's body, matching the Flutter panel glow. */}
      <span
        className="relative inline-flex items-center justify-center"
        style={{ width: size, height: size }}
      >
        <span
          aria-hidden
          className="pointer-events-none absolute rounded-full"
          style={{
            width: size * 1.4,
            height: size * 1.4,
            background: `radial-gradient(circle, ${ACCENT}f2 24%, ${ACCENT}99 46%, transparent 66%)`,
          }}
        />
        <img
          src="/applad-mascot-head.png"
          alt="Applad"
          width={size}
          height={size}
          className="relative"
          style={{ width: size, height: size }}
        />
      </span>
      <span
        className="ml-3.5 font-bold tracking-[-0.014em] text-text-primary"
        style={{ fontSize: wordmark }}
      >
        applad
      </span>
    </div>
  );
}

function SocialButton({ provider, onClick }: { provider: string; onClick: () => void }) {
  const label =
    provider === 'github'
      ? 'Continue with GitHub'
      : provider === 'google'
        ? 'Continue with Google'
        : provider === 'sso'
          ? 'Continue with SSO'
          : `Continue with ${provider[0].toUpperCase()}${provider.slice(1)}`;
  const icon =
    provider === 'github' ? (
      <GitBranch size={16} />
    ) : provider === 'google' ? (
      <span className="text-[15px] font-bold" style={{ color: '#4285F4' }}>
        G
      </span>
    ) : provider === 'sso' ? (
      <Building2 size={16} />
    ) : (
      <LogIn size={16} />
    );
  return (
    <button
      type="button"
      onClick={onClick}
      className="mb-1.5 flex h-[var(--control-h)] w-full items-center justify-center gap-2.5 rounded-[8px] border border-field-border bg-surface text-[13px] font-medium text-text-primary transition-colors hover:bg-fill"
    >
      {icon}
      {label}
    </button>
  );
}

function OrDivider() {
  return (
    <div className="flex items-center">
      <div className="h-px flex-1 bg-field-border" />
      <span className="px-3 text-[12px] text-text-subtle">or</span>
      <div className="h-px flex-1 bg-field-border" />
    </div>
  );
}

function PasswordInput({
  value,
  onChange,
  show,
  onToggle,
  autoComplete,
}: {
  value: string;
  onChange: (v: string) => void;
  show: boolean;
  onToggle: () => void;
  autoComplete?: string;
}) {
  return (
    <div className="relative">
      <Input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Your password"
        autoComplete={autoComplete}
        required
        className="pr-9"
      />
      <button
        type="button"
        onClick={onToggle}
        className="absolute right-2 top-1/2 -translate-y-1/2 text-text-subtle transition-colors hover:text-text-primary"
        aria-label={show ? 'Hide password' : 'Show password'}
      >
        {show ? <EyeOff size={16} /> : <Eye size={16} />}
      </button>
    </div>
  );
}

function PolicyCheckbox({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex cursor-pointer items-start gap-2.5">
      <Checkbox checked={checked} onCheckedChange={(v) => onChange(!!v)} className="mt-0.5" />
      <span className="text-[12px] leading-[1.6] text-text-subtle">
        By registering, you agree that you have read, understand, and acknowledge our{' '}
        <span className="text-text-secondary underline">Privacy Policy</span> and accept our{' '}
        <span className="text-text-secondary underline">Terms of Use</span>.
      </span>
    </label>
  );
}

function SubmitButton({
  children,
  busy,
  disabled,
}: {
  children: React.ReactNode;
  busy?: boolean;
  disabled?: boolean;
}) {
  return (
    <button
      type="submit"
      disabled={busy || disabled}
      className="flex h-[var(--control-h)] w-full items-center justify-center rounded-[8px] text-[13px] font-semibold text-white transition-opacity disabled:opacity-40"
      style={{ backgroundColor: ACCENT }}
    >
      {busy ? <Loader2 size={16} className="animate-spin" /> : children}
    </button>
  );
}

function TextLink({
  children,
  onClick,
  primary,
}: {
  children: React.ReactNode;
  onClick: () => void;
  primary?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="text-[13px] font-medium underline"
      style={{ color: primary ? ACCENT : 'var(--text-secondary)' }}
    >
      {children}
    </button>
  );
}

function ErrorBanner({ message }: { message: string }) {
  return (
    <div
      className="mt-4 flex items-center gap-2 rounded-[8px] border px-3 py-2.5 text-[12px]"
      style={{
        color: '#ef4444',
        backgroundColor: 'rgba(239,68,68,0.08)',
        borderColor: 'rgba(239,68,68,0.2)',
      }}
    >
      <AlertCircle size={14} className="shrink-0" />
      {message}
    </div>
  );
}

function SuccessBanner({ message, className }: { message: string; className?: string }) {
  return (
    <div
      className={cn('flex items-center gap-2 rounded-[8px] border px-3 py-2.5 text-[12px]', className)}
      style={{
        color: '#22c55e',
        backgroundColor: 'rgba(34,197,94,0.08)',
        borderColor: 'rgba(34,197,94,0.25)',
      }}
    >
      <CheckCircle2 size={14} className="shrink-0" />
      {message}
    </div>
  );
}

function LoadingOverlay() {
  return (
    <div className="flex h-[300px] flex-col items-center justify-center gap-4">
      <Loader2 size={28} className="animate-spin" style={{ color: ACCENT }} />
      <span className="text-[14px] text-text-muted">Signing you in…</span>
    </div>
  );
}

/* Abstract swept background — ports login_page.dart `_PanelShapes`.
 * viewBox is 0-100 with preserveAspectRatio=none so the fractional path
 * coordinates stretch to fill the panel exactly like the Flutter CustomPaint.
 * Gradients use userSpaceOnUse to span the whole panel, not each shape. */
function PanelShapes() {
  return (
    <svg
      className="absolute inset-0 h-full w-full"
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      aria-hidden
    >
      <defs>
        <linearGradient id="ps1" gradientUnits="userSpaceOnUse" x1="100" y1="0" x2="0" y2="100">
          <stop offset="0" stopColor="#fff" stopOpacity="0.055" />
          <stop offset="1" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
        <linearGradient id="ps2" gradientUnits="userSpaceOnUse" x1="100" y1="0" x2="0" y2="50">
          <stop offset="0" stopColor="#fff" stopOpacity="0.07" />
          <stop offset="1" stopColor="#fff" stopOpacity="0.01" />
        </linearGradient>
        <linearGradient id="ps3" gradientUnits="userSpaceOnUse" x1="0" y1="50" x2="100" y2="100">
          <stop offset="0" stopColor="#fff" stopOpacity="0.025" />
          <stop offset="1" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
        <linearGradient id="ps4" gradientUnits="userSpaceOnUse" x1="50" y1="0" x2="50" y2="100">
          <stop offset="0" stopColor="#fff" stopOpacity="0.10" />
          <stop offset="1" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d="M25 0 L100 0 L100 55 C85 42 55 28 5 18 Z" fill="url(#ps1)" />
      <path d="M55 0 L100 0 L100 32 C90 22 75 14 48 7 Z" fill="url(#ps2)" />
      <path d="M0 60 C15 55 30 70 10 100 L0 100 Z" fill="url(#ps3)" />
      <path d="M24 0 L30 0 C18 8 8 14 4 19 C0 14 6 9 19 0 Z" fill="url(#ps4)" />
    </svg>
  );
}
