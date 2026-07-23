import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  BookOpen,
  Github,
  LifeBuoy,
  LogOut,
  MessageCircle,
  MessagesSquare,
  Monitor,
  Moon,
  MoreHorizontal,
  Sun,
  User as UserIcon,
} from 'lucide-react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { ConfirmDialog } from '@/components/form-dialog';
import { useAuthStore } from '@/stores/auth';
import { useThemeStore, type ThemeMode } from '@/stores/theme';
import { cn } from '@/lib/utils';
import { extensionNavActions } from '@/extensions';

function initials(name: string, email: string): string {
  const base = name.trim() || email.trim();
  const parts = base.split(/[\s@.]+/).filter(Boolean);
  return (parts[0]?.[0] ?? '?').toUpperCase() + (parts[1]?.[0]?.toUpperCase() ?? '');
}

/* UserMenu — avatar → account, sign out (confirm), Light/Dark/System toggle.
 * Ports navbar_popovers.dart UserMenuButton + _ThemeToggle. */
export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const { mode, setMode } = useThemeStore();
  const navigate = useNavigate();
  const [confirmOut, setConfirmOut] = useState(false);

  return (
    <>
      <Popover>
        <PopoverTrigger asChild>
          <button
            className="flex h-[34px] w-[34px] items-center justify-center rounded-full bg-[var(--color-accent)] text-[length:var(--text-label)] font-semibold text-white"
            aria-label="Account menu"
          >
            {user ? initials(user.name, user.email) : '?'}
          </button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-[230px] p-2">
          <div className="px-2 py-1.5">
            <div className="truncate text-[length:var(--text-body)] font-medium text-text-primary">
              {user?.name || 'Account'}
            </div>
            <div className="truncate text-[length:var(--text-caption)] text-text-muted">
              {user?.email}
            </div>
          </div>
          <div className="my-1 h-px bg-border" />
          <MenuItem icon={UserIcon} label="Account" onClick={() => navigate('/account')} />
          <div className="my-1 h-px bg-border" />
          <div className="px-2 py-1 text-[length:var(--text-caption)] text-text-muted">
            Theme
          </div>
          <div className="flex gap-1 px-1 pb-1">
            {(
              [
                ['light', Sun],
                ['dark', Moon],
                ['system', Monitor],
              ] as [ThemeMode, typeof Sun][]
            ).map(([m, Icon]) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={cn(
                  'flex flex-1 flex-col items-center gap-1 rounded-[var(--radius-6)] py-1.5 text-[length:var(--text-2xs)] capitalize transition-colors',
                  mode === m
                    ? 'bg-fill-active text-text-primary'
                    : 'text-text-muted hover:bg-fill',
                )}
              >
                <Icon size={14} />
                {m}
              </button>
            ))}
          </div>
          <div className="my-1 h-px bg-border" />
          <MenuItem icon={LogOut} label="Sign out" destructive onClick={() => setConfirmOut(true)} />
        </PopoverContent>
      </Popover>
      <ConfirmDialog
        open={confirmOut}
        onOpenChange={setConfirmOut}
        title="Sign out"
        message="Are you sure you want to sign out?"
        confirmLabel="Sign out"
        onConfirm={() => {
          void logout();
          navigate('/login');
        }}
      />
    </>
  );
}

function MenuItem({
  icon: Icon,
  label,
  onClick,
  destructive,
}: {
  icon: typeof UserIcon;
  label: string;
  onClick: () => void;
  destructive?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex w-full items-center gap-2 rounded-[var(--radius-6)] px-2 py-1.5 text-[length:var(--text-body)] transition-colors hover:bg-fill',
        destructive ? 'text-[var(--color-danger)]' : 'text-text-secondary',
      )}
    >
      <Icon size={14} />
      {label}
    </button>
  );
}

/* Support — Discord / GitHub / Docs cards (faithful stub). */
export function SupportButton() {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button className="text-[length:var(--text-body)] text-text-secondary transition-colors hover:text-text-primary">
          Support
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[340px] max-w-[calc(100vw-1.5rem)] p-2">
        <SupportPanelBody />
      </PopoverContent>
    </Popover>
  );
}

function SupportPanelBody() {
  return (
    <>
      <SupportCard icon={MessagesSquare} title="Discord" subtitle="Chat with the community" href="https://discord.gg/applad" />
      <SupportCard icon={Github} title="GitHub" subtitle="Open an issue" href="https://github.com/mittolabs/applad/issues" />
      <SupportCard icon={BookOpen} title="Docs" subtitle="Read the documentation" href="https://docs.applad.io" />
    </>
  );
}

/* Compact-nav overflow menu — ports shell.dart `_NavOverflowMenu`. Below 780px
 * the top-bar buttons collapse into a single 3-dot menu. Selecting a row swaps
 * the popover content in place (anchored to the same trigger), so the full
 * panels are reachable without leaving the top bar.
 *
 * Modules compiled into this build contribute their own entries; a default build
 * contributes none and this is just Support. */
export function NavOverflowMenu() {
  const [view, setView] = useState<'menu' | 'support' | null>(null);
  const actions = extensionNavActions();
  const width = view === 'support' ? 'w-[340px] p-2' : 'w-[180px] p-1';
  return (
    <Popover open={view !== null} onOpenChange={(o) => setView(o ? 'menu' : null)}>
      <PopoverTrigger asChild>
        <button
          className="flex h-[34px] w-[34px] items-center justify-center rounded-[var(--radius-8)] text-text-muted transition-colors hover:bg-fill hover:text-text-primary"
          title="More"
          aria-label="More"
        >
          <MoreHorizontal size={17} />
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className={width}>
        {view === 'menu' && (
          <>
            <MenuItem icon={LifeBuoy} label="Support" onClick={() => setView('support')} />
            {actions.map((Action, i) => (
              <Action key={i} />
            ))}
          </>
        )}
        {view === 'support' && <SupportPanelBody />}
      </PopoverContent>
    </Popover>
  );
}

function SupportCard({
  icon: Icon,
  title,
  subtitle,
  href,
}: {
  icon: typeof Github;
  title: string;
  subtitle: string;
  href: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="flex items-center gap-3 rounded-[var(--radius)] px-2 py-2 transition-colors hover:bg-fill"
    >
      <div className="flex h-8 w-8 items-center justify-center rounded-[var(--radius)] bg-fill text-text-secondary">
        <Icon size={16} />
      </div>
      <div>
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">{title}</div>
        <div className="text-[length:var(--text-caption)] text-text-muted">{subtitle}</div>
      </div>
    </a>
  );
}

export { MessageCircle };
