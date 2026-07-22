import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  BookOpen,
  Github,
  LifeBuoy,
  LogOut,
  MessageCircle,
  MessageSquare,
  MessagesSquare,
  Monitor,
  Moon,
  MoreHorizontal,
  Sun,
  User as UserIcon,
  X,
} from 'lucide-react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { ConfirmDialog } from '@/components/form-dialog';
import { useAuthStore } from '@/stores/auth';
import { useThemeStore, type ThemeMode } from '@/stores/theme';
import { cn } from '@/lib/utils';

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

/* Feedback — ports navbar_popovers.dart _FeedbackPanel: header + subtitle,
 * Category chips, labeled textarea, Cancel/Submit. Simulated submit (stub). */
const FEEDBACK_CATEGORIES = ['Bug report', 'Feature request', 'General'];

export function FeedbackButton() {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button className="text-[length:var(--text-body)] text-text-secondary transition-colors hover:text-text-primary">
          Feedback
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[440px] max-w-[calc(100vw-1.5rem)] p-0">
        <FeedbackPanelBody onClose={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
}

/* The Feedback panel body, reusable inside either the inline nav button or the
 * compact overflow menu. State resets naturally on unmount when closed. */
function FeedbackPanelBody({ onClose }: { onClose: () => void }) {
  const [sent, setSent] = useState(false);
  const [category, setCategory] = useState('General');
  const [text, setText] = useState('');
  const close = onClose;

  return (
    <>
      {sent ? (
          <div className="flex flex-col items-center gap-2 px-6 py-10 text-center">
            <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
              Thanks for the feedback!
            </div>
            <div className="text-[length:var(--text-body)] text-text-muted">
              We appreciate you helping us improve Applad.
            </div>
          </div>
        ) : (
          <>
            <div className="flex items-start justify-between gap-3 px-5 pt-5">
              <div>
                <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
                  Feedback
                </div>
                <div className="mt-1 text-[length:var(--text-body)] text-text-muted">
                  Applad evolves with your input. Share your thoughts and help us improve.
                </div>
              </div>
              <button
                onClick={close}
                className="rounded-[var(--radius-6)] p-1 text-text-muted transition-colors hover:bg-fill hover:text-text-primary"
                aria-label="Close"
              >
                <X size={16} />
              </button>
            </div>

            <div className="mt-4 h-px bg-border" />

            <div className="flex flex-col gap-4 px-5 py-4">
              <div>
                <div className="mb-2 text-[length:var(--text-label)] font-medium text-text-secondary">
                  Category
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {FEEDBACK_CATEGORIES.map((c) => (
                    <button
                      key={c}
                      onClick={() => setCategory(c)}
                      className={cn(
                        'rounded-[var(--radius-6)] border px-3 py-1.5 text-[length:var(--text-caption)] transition-colors',
                        category === c
                          ? 'border-[var(--color-accent)] bg-fill-active text-text-primary'
                          : 'border-border text-text-muted hover:text-text-secondary',
                      )}
                    >
                      {c}
                    </button>
                  ))}
                </div>
              </div>
              <div>
                <div className="mb-1.5 text-[length:var(--text-label)] font-medium text-text-secondary">
                  Tell us more about your experience
                </div>
                <Textarea
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  placeholder="Share your suggestions and feature requests…"
                  rows={5}
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 px-5 pb-5">
              <Button variant="ghost" size="sm" onClick={close}>
                Cancel
              </Button>
              <Button size="sm" onClick={() => setSent(true)} disabled={!text.trim()}>
                Submit
              </Button>
            </div>
          </>
        )}
    </>
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
 * the Feedback + Support buttons collapse into a single 3-dot menu. Selecting a
 * row swaps the popover content in place (anchored to the same trigger), so the
 * full panels are reachable without leaving the top bar. */
export function NavOverflowMenu() {
  const [view, setView] = useState<'menu' | 'feedback' | 'support' | null>(null);
  const width =
    view === 'feedback' ? 'w-[440px] p-0' : view === 'support' ? 'w-[340px] p-2' : 'w-[180px] p-1';
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
            <MenuItem icon={MessageSquare} label="Feedback" onClick={() => setView('feedback')} />
            <MenuItem icon={LifeBuoy} label="Support" onClick={() => setView('support')} />
          </>
        )}
        {view === 'feedback' && <FeedbackPanelBody onClose={() => setView(null)} />}
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
