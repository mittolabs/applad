import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowRight,
  BookOpen,
  Check,
  Code2,
  Database,
  FolderOpen,
  KeyRound,
  MessagesSquare,
  PartyPopper,
  Rocket,
  Terminal,
  Users,
  type LucideIcon,
} from 'lucide-react';
import { api } from '@/api/client';
import { useProject } from '@/api/queries';
import { CodeBlock } from '@/components/code-block';
import { IdText } from '@/components/id-text';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

/*
 * Get started — the project onboarding hub. Redesigned from the placeholder into
 * a working "connect your app + finish setup" page:
 *   1. live setup progress (real resource counts)
 *   2. connect snippets (SDK language switcher) + project credentials
 *   3. an actionable setup checklist that ticks off as resources are created
 *   4. next-step resource links
 */

interface StepDef {
  key: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  route: string;
  cta: string;
  countPath: string;
}

const STEPS: StepDef[] = [
  {
    key: 'database',
    label: 'Database',
    title: 'Create your first database',
    description: 'Model your data with tables, columns, relationships and row-level security.',
    icon: Database,
    route: 'databases',
    cta: 'New database',
    countPath: '/databases',
  },
  {
    key: 'auth',
    label: 'Auth',
    title: 'Add authentication',
    description: 'Email, OAuth (15 providers), magic links and MFA — enable it in one place.',
    icon: Users,
    route: 'auth',
    cta: 'Configure auth',
    countPath: '/users',
  },
  {
    key: 'storage',
    label: 'Storage',
    title: 'Set up file storage',
    description: 'Create a bucket for uploads with image transforms and antivirus scanning.',
    icon: FolderOpen,
    route: 'storage',
    cta: 'New bucket',
    countPath: '/storage/buckets',
  },
  {
    key: 'function',
    label: 'Functions',
    title: 'Deploy a function',
    description: 'Run serverless code in 8 runtimes, pre-warmed for low cold-start latency.',
    icon: Code2,
    route: 'functions',
    cta: 'New function',
    countPath: '/functions',
  },
];

function totalOf(d: Record<string, unknown>): number {
  if (typeof d.total === 'number') return d.total;
  const arr = Object.values(d).find(Array.isArray) as unknown[] | undefined;
  return arr?.length ?? 0;
}

export function GetStartedPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const { data: project } = useProject(projectId);
  const projectName = (project?.name as string | undefined) ?? 'your project';

  const go = (route: string) => navigate(`/project/${projectId}/${route}`);

  // Per-step completion from live resource counts.
  const { data: done = {} } = useQuery({
    queryKey: ['get-started-steps', projectId],
    enabled: !!projectId,
    staleTime: 30_000,
    queryFn: async () => {
      const entries = await Promise.all(
        STEPS.map(async (s) => {
          try {
            const r = await api.get(s.countPath, { params: { limit: 1 } });
            return [s.key, totalOf(r.data as Record<string, unknown>) > 0] as const;
          } catch {
            return [s.key, false] as const;
          }
        }),
      );
      return Object.fromEntries(entries) as Record<string, boolean>;
    },
  });

  const completed = STEPS.filter((s) => done[s.key]).length;
  const pct = Math.round((completed / STEPS.length) * 100);
  const allDone = completed === STEPS.length;

  const endpoint = typeof window !== 'undefined' ? window.location.origin : 'https://your-host';

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-8 p-6 md:p-8">
      {/* Header */}
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Get started</h1>
        <p className="mt-1 max-w-2xl text-[length:var(--text-body)] text-text-secondary">
          Everything you need to connect your app to{' '}
          <span className="text-text-primary">{projectName}</span> and ship your backend.
        </p>
      </div>

      {/* Progress banner */}
      <ProgressBanner completed={completed} total={STEPS.length} pct={pct} allDone={allDone} />

      {/* Connect + credentials */}
      <div className="grid min-w-0 gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
        <ConnectCard endpoint={endpoint} projectId={projectId ?? ''} />
        <CredentialsCard endpoint={endpoint} projectId={projectId ?? ''} onManageKeys={() => go('settings?tab=api-keys')} />
      </div>

      {/* Setup checklist */}
      <section className="flex flex-col gap-4">
        <div className="flex items-baseline justify-between">
          <h2 className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
            Finish setting up
          </h2>
          <span className="text-[length:var(--text-caption)] text-text-muted">
            {completed} of {STEPS.length} done
          </span>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {STEPS.map((s) => (
            <StepCard key={s.key} step={s} done={!!done[s.key]} onClick={() => go(s.route)} />
          ))}
        </div>
      </section>

      {/* Resources */}
      <section className="flex flex-col gap-4">
        <h2 className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
          Keep exploring
        </h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <ResourceLink icon={BookOpen} title="Documentation" subtitle="Guides & concepts" href="https://docs.applad.io" />
          <ResourceLink icon={Terminal} title="API reference" subtitle="Every endpoint" href="https://docs.applad.io/api" />
          <ResourceLink icon={Rocket} title="Deploy" subtitle="Ship to production" onClick={() => go('deploy')} />
          <ResourceLink icon={MessagesSquare} title="Community" subtitle="Join our Discord" href="https://discord.gg/applad" />
        </div>
      </section>
    </div>
  );
}

// ── Progress banner ──────────────────────────────────────────────────────────

function ProgressBanner({
  completed,
  total,
  pct,
  allDone,
}: {
  completed: number;
  total: number;
  pct: number;
  allDone: boolean;
}) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-[var(--radius-12)] border p-6',
        allDone
          ? 'border-[color-mix(in_srgb,var(--status-success)_40%,transparent)] bg-[color-mix(in_srgb,var(--status-success)_8%,transparent)]'
          : 'border-border bg-surface',
      )}
    >
      <div
        className="pointer-events-none absolute inset-0 opacity-60"
        style={{
          background:
            'radial-gradient(560px circle at 88% -20%, color-mix(in srgb, var(--color-accent) 16%, transparent), transparent 60%)',
        }}
      />
      <div className="relative flex items-center gap-4">
        <div
          className={cn(
            'flex h-12 w-12 shrink-0 items-center justify-center rounded-[var(--radius-10)]',
            allDone
              ? 'bg-[color-mix(in_srgb,var(--status-success)_18%,transparent)] text-[var(--status-success)]'
              : 'bg-[color-mix(in_srgb,var(--color-accent)_18%,transparent)] text-[var(--color-accent)]',
          )}
        >
          {allDone ? <PartyPopper size={22} /> : <Rocket size={22} />}
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
            {allDone ? "You're all set!" : 'Set up your backend'}
          </div>
          <div className="mt-0.5 text-[length:var(--text-body)] text-text-secondary">
            {allDone
              ? 'Every core service is configured. Build something great.'
              : `${completed} of ${total} steps complete — pick up where you left off below.`}
          </div>
        </div>
        <div className="hidden shrink-0 text-right sm:block">
          <div className="text-[length:var(--text-h2)] font-semibold text-text-primary">{pct}%</div>
        </div>
      </div>
      <div className="relative mt-4 h-2 overflow-hidden rounded-full bg-fill-active">
        <div
          className="h-full rounded-full transition-[width] duration-500"
          style={{
            width: `${pct}%`,
            background: allDone ? 'var(--status-success)' : 'var(--color-accent)',
          }}
        />
      </div>
    </div>
  );
}

// ── Connect card (SDK language switcher) ─────────────────────────────────────

const LANGS = ['JavaScript', 'Node.js', 'Dart', 'Go', 'Python'] as const;
type Lang = (typeof LANGS)[number];

function snippetsFor(lang: Lang, endpoint: string, projectId: string): { install: string; init: string; lang: string } {
  const pid = projectId || 'YOUR_PROJECT_ID';
  switch (lang) {
    case 'JavaScript':
      return {
        lang: 'ts',
        install: 'npm install @mittolabs/applad',
        init: `import { Applad } from '@mittolabs/applad';\n\nconst applad = new Applad({\n  endpoint: '${endpoint}',\n  projectId: '${pid}',\n});`,
      };
    case 'Node.js':
      return {
        lang: 'ts',
        install: 'npm install @mittolabs/applad-node',
        init: `import { Applad } from '@mittolabs/applad-node';\n\nconst applad = new Applad({\n  endpoint: '${endpoint}',\n  projectId: '${pid}',\n  apiKey: process.env.APPLAD_API_KEY,\n});`,
      };
    case 'Dart':
      return {
        lang: 'dart',
        install: 'flutter pub add applad',
        init: `import 'package:applad/applad.dart';\n\nfinal applad = Applad(\n  endpoint: '${endpoint}',\n  projectId: '${pid}',\n);`,
      };
    case 'Go':
      return {
        lang: 'go',
        install: 'go get github.com/mittolabs/applad-go',
        init: `import applad "github.com/mittolabs/applad-go"\n\nclient := applad.New(applad.Config{\n  Endpoint:  "${endpoint}",\n  ProjectID: "${pid}",\n  APIKey:    os.Getenv("APPLAD_API_KEY"),\n})`,
      };
    case 'Python':
      return {
        lang: 'python',
        install: 'pip install applad',
        init: `from applad import Applad\n\napplad = Applad(\n    endpoint="${endpoint}",\n    project_id="${pid}",\n    api_key=os.environ["APPLAD_API_KEY"],\n)`,
      };
  }
}

function ConnectCard({ endpoint, projectId }: { endpoint: string; projectId: string }) {
  const [lang, setLang] = useState<Lang>('JavaScript');
  const snip = useMemo(() => snippetsFor(lang, endpoint, projectId), [lang, endpoint, projectId]);
  return (
    <div className="flex min-w-0 flex-col rounded-[var(--radius-12)] border border-border bg-surface">
      <div className="flex items-center gap-2 border-b border-border px-5 py-3.5">
        <Terminal size={16} className="text-text-secondary" />
        <span className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
          Connect your app
        </span>
      </div>

      {/* Language tabs */}
      <div className="flex flex-wrap gap-1 px-4 pt-4">
        {LANGS.map((l) => (
          <button
            key={l}
            onClick={() => setLang(l)}
            className={cn(
              'rounded-[var(--radius-6)] px-2.5 py-1 text-[length:var(--text-caption)] transition-colors',
              lang === l
                ? 'bg-fill-active font-medium text-text-primary'
                : 'text-text-muted hover:bg-fill hover:text-text-secondary',
            )}
          >
            {l}
          </button>
        ))}
      </div>

      <div className="flex min-w-0 flex-col gap-4 p-4">
        <div className="min-w-0">
          <div className="mb-1.5 text-[length:var(--text-label)] font-medium text-text-secondary">
            1. Install the SDK
          </div>
          <CodeBlock code={snip.install} language="bash" />
        </div>
        <div className="min-w-0">
          <div className="mb-1.5 text-[length:var(--text-label)] font-medium text-text-secondary">
            2. Initialize the client
          </div>
          <CodeBlock code={snip.init} language={snip.lang} />
        </div>
      </div>
    </div>
  );
}

// ── Credentials card ─────────────────────────────────────────────────────────

function CredentialsCard({
  endpoint,
  projectId,
  onManageKeys,
}: {
  endpoint: string;
  projectId: string;
  onManageKeys: () => void;
}) {
  return (
    <div className="flex h-fit flex-col gap-4 rounded-[var(--radius-12)] border border-border bg-surface p-5">
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        Project credentials
      </div>

      <Field label="API endpoint">
        <CopyValue value={endpoint} />
      </Field>

      <Field label="Project ID">
        <IdText id={projectId} previewLength={18} />
      </Field>

      <Field label="API key">
        <p className="text-[length:var(--text-caption)] text-text-muted">
          Server SDKs authenticate with a project API key. Create and manage keys in settings.
        </p>
        <Button variant="outline" size="sm" className="mt-2 w-full" onClick={onManageKeys}>
          <KeyRound size={14} />
          Manage API keys
        </Button>
      </Field>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1.5 text-[length:var(--text-label)] font-medium text-text-secondary">
        {label}
      </div>
      {children}
    </div>
  );
}

function CopyValue({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };
  return (
    <button
      onClick={copy}
      className="flex w-full items-center gap-2 rounded-[var(--radius-6)] border border-border bg-field-fill px-2.5 py-1.5 text-left transition-colors hover:border-field-border"
      title="Copy"
    >
      <span className="min-w-0 flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-primary">
        {value}
      </span>
      {copied ? (
        <Check size={13} className="shrink-0 text-[var(--status-success)]" />
      ) : (
        <Code2 size={13} className="shrink-0 text-text-subtle" />
      )}
    </button>
  );
}

// ── Step card ────────────────────────────────────────────────────────────────

function StepCard({ step, done, onClick }: { step: StepDef; done: boolean; onClick: () => void }) {
  const Icon = step.icon;
  return (
    <button
      onClick={onClick}
      className={cn(
        'group flex flex-col rounded-[var(--radius-10)] border p-4 text-left transition-colors',
        done
          ? 'border-[color-mix(in_srgb,var(--status-success)_35%,transparent)] bg-[color-mix(in_srgb,var(--status-success)_6%,transparent)]'
          : 'border-border bg-surface hover:border-field-border hover:bg-fill',
      )}
    >
      <div className="flex items-center gap-3">
        <div
          className={cn(
            'flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-8)]',
            done
              ? 'bg-[color-mix(in_srgb,var(--status-success)_18%,transparent)] text-[var(--status-success)]'
              : 'bg-fill-active text-text-secondary',
          )}
        >
          {done ? <Check size={18} /> : <Icon size={17} />}
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-[length:var(--text-label)] font-medium uppercase tracking-wide text-text-muted">
            {step.label}
          </div>
          <div className="text-[length:var(--text-body)] font-semibold text-text-primary">
            {step.title}
          </div>
        </div>
        {done && (
          <span className="shrink-0 rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--status-success)_14%,transparent)] px-1.5 py-0.5 text-[length:var(--text-2xs)] font-medium text-[var(--status-success)]">
            Done
          </span>
        )}
      </div>
      <p className="mt-2.5 text-[length:var(--text-caption)] leading-relaxed text-text-muted">
        {step.description}
      </p>
      <div className="mt-3 flex items-center gap-1 text-[length:var(--text-caption)] font-medium text-[var(--color-accent)]">
        {done ? 'Open' : step.cta}
        <ArrowRight size={13} className="transition-transform group-hover:translate-x-0.5" />
      </div>
    </button>
  );
}

// ── Resource link ────────────────────────────────────────────────────────────

function ResourceLink({
  icon: Icon,
  title,
  subtitle,
  href,
  onClick,
}: {
  icon: LucideIcon;
  title: string;
  subtitle: string;
  href?: string;
  onClick?: () => void;
}) {
  const className =
    'flex items-center gap-3 rounded-[var(--radius-10)] border border-border bg-surface p-3.5 text-left transition-colors hover:border-field-border hover:bg-fill';
  const inner = (
    <>
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-8)] bg-fill-active text-text-secondary">
        <Icon size={16} />
      </div>
      <div className="min-w-0">
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">{title}</div>
        <div className="truncate text-[length:var(--text-caption)] text-text-muted">{subtitle}</div>
      </div>
    </>
  );
  if (href) {
    return (
      <a href={href} target="_blank" rel="noreferrer" className={className}>
        {inner}
      </a>
    );
  }
  return (
    <button onClick={onClick} className={className}>
      {inner}
    </button>
  );
}
