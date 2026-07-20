import { useEffect, useState } from 'react';
import {
  ArrowRight,
  Activity,
  Blocks,
  Check,
  Cloud,
  Code2,
  Copy,
  Database,
  FileText,
  Flag,
  FlaskConical,
  FolderOpen,
  Github,
  KeyRound,
  Mail,
  PencilRuler,
  Radio,
  Rocket,
  Server,
  ShieldCheck,
  Sparkles,
  Terminal,
  Workflow,
  type LucideIcon,
} from 'lucide-react';

const CONSOLE_URL = 'http://applad.dev.localhost';
const DOCS_URL = 'http://docs.applad.io.localhost';
const STATUS_URL = 'http://status.applad.io.localhost';
const GITHUB_URL = 'https://github.com/mittolabs/applad';
const ACCENT = '#3472a4';

export function App() {
  return (
    <div className="min-h-screen">
      <Nav />
      <Hero />
      <Lifecycle />
      <AiSpotlight />
      <WorkflowSpotlight />
      <Services />
      <Deploy />
      <CTA />
      <Footer />
    </div>
  );
}

function Nav() {
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);
  return (
    <header
      className={`fixed inset-x-0 top-0 z-50 transition-colors duration-200 ${
        scrolled ? 'border-b border-border bg-[color:var(--color-surface-alt)]/85 backdrop-blur' : 'border-b border-transparent'
      }`}
    >
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-2 px-6">
        <a href="/" className="flex items-center gap-2.5">
          <img src="/favicon.png" alt="" className="h-7 w-7 rounded-[7px] object-cover" />
          <span className="text-[17px] font-bold tracking-tight">applad</span>
        </a>
        <nav className="ml-8 hidden items-center gap-6 text-sm text-muted md:flex">
          <a href="#platform" className="transition-colors hover:text-text">Platform</a>
          <a href="#pricing" className="transition-colors hover:text-text">Pricing</a>
          <a href={DOCS_URL} className="transition-colors hover:text-text">Docs</a>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer" className="transition-colors hover:text-text">GitHub</a>
        </nav>
        <div className="ml-auto flex items-center gap-3">
          <a href={CONSOLE_URL} className="hidden text-sm text-muted transition-colors hover:text-text sm:block">Sign in</a>
          <a href={CONSOLE_URL} className="rounded-lg px-3 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90" style={{ background: ACCENT }}>
            Open console
          </a>
        </div>
      </div>
    </header>
  );
}

function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="pointer-events-none absolute inset-0" style={{ background: `radial-gradient(760px circle at 50% -6%, ${ACCENT}2e, transparent 60%)` }} />
      <div className="relative mx-auto max-w-6xl px-6 pt-28 md:pt-32">
        <div className="mx-auto max-w-3xl text-center">
          <a href={GITHUB_URL} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded-full border border-border px-3 py-1 text-xs text-muted transition-colors hover:text-text">
            <Sparkles size={13} style={{ color: ACCENT }} />
            Open-source · Cloud or self-hosted
          </a>
          <h1 className="mt-6 text-4xl font-bold leading-[1.05] tracking-tight md:text-[64px]">
            Build products,
            <br />
            not infrastructure.
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-muted">
            Applad is a complete AI-native development infrastructure for your apps. Plan, build,
            test, deploy and monitor in one platform, with SDKs that reach every part of it.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <a href={CONSOLE_URL} className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-opacity hover:opacity-90" style={{ background: ACCENT }}>
              Get started
              <ArrowRight size={16} />
            </a>
            <a href={DOCS_URL} className="inline-flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-text transition-colors hover:bg-surface">
              Read the docs
            </a>
          </div>
        </div>

        <div className="relative mx-auto mt-16 max-w-3xl [mask-image:linear-gradient(to_bottom,black_82%,transparent)]">
          <img
            src="/hero-databases.png"
            alt="The Applad console showing a users database table"
            className="block w-full rounded-xl border border-border shadow-[0_30px_90px_-24px_rgba(0,0,0,0.75)]"
            loading="lazy"
          />
        </div>
      </div>
    </section>
  );
}

// The software lifecycle Applad covers, stage by stage.
const LIFECYCLE: { icon: LucideIcon; name: string }[] = [
  { icon: PencilRuler, name: 'Plan' },
  { icon: Blocks, name: 'Build' },
  { icon: FlaskConical, name: 'Test' },
  { icon: Rocket, name: 'Deploy' },
  { icon: Workflow, name: 'Automate' },
  { icon: Activity, name: 'Monitor' },
];

function Lifecycle() {
  return (
    <section className="mx-auto max-w-6xl px-6 py-16">
      <p className="text-center text-xs font-medium uppercase tracking-[0.16em] text-muted/70">
        One platform for the entire software lifecycle
      </p>
      <div className="mt-8 flex flex-wrap items-center justify-center gap-x-12 gap-y-6">
        {LIFECYCLE.map((s) => (
          <div
            key={s.name}
            className="flex items-center gap-2 text-muted opacity-55 transition-opacity hover:opacity-90"
          >
            <s.icon size={22} strokeWidth={1.6} />
            <span className="text-lg font-semibold tracking-tight">{s.name}</span>
          </div>
        ))}
      </div>
    </section>
  );
}

function Spotlight({
  eyebrow,
  title,
  body,
  visual,
  flip,
}: {
  eyebrow: string;
  title: string;
  body: React.ReactNode;
  visual: React.ReactNode;
  flip?: boolean;
}) {
  return (
    <section className="mx-auto max-w-6xl px-6 py-24">
      <div className="grid items-center gap-14 lg:grid-cols-2">
        <div className={flip ? 'lg:order-2' : ''}>
          <div className="eyebrow">{eyebrow}</div>
          <h2 className="mt-3 text-3xl font-bold tracking-tight md:text-[40px] md:leading-[1.1]">{title}</h2>
          <div className="mt-5 max-w-md text-[17px] leading-relaxed text-muted">{body}</div>
        </div>
        <div className={flip ? 'lg:order-1' : ''}>{visual}</div>
      </div>
    </section>
  );
}

function AiSpotlight() {
  return (
    <Spotlight
      eyebrow="The lad"
      title="Just ask"
      body={
        <>
          The lad is an AI that works on your project, not just chats about it. It knows your
          schema, data and config, and takes real actions across the stack: designing tables,
          wiring auth, writing functions and workflows, connecting services and shipping. No
          snippets to paste. Prefer to build it yourself? The SDKs are right there.
        </>
      }
      visual={
        <div className="relative">
          <div
            className="pointer-events-none absolute -inset-6 -z-10"
            style={{ background: `radial-gradient(460px circle at 60% 30%, ${ACCENT}22, transparent 70%)` }}
          />
          <img
            src="/ai-popup.png"
            alt="Applad AI assistant answering a request inside the console"
            className="mx-auto block w-full max-w-[440px]"
            loading="lazy"
          />
        </div>
      }
    />
  );
}

function WorkflowSpotlight() {
  return (
    <Spotlight
      flip
      eyebrow="Workflows"
      title="Automate anything, visually"
      body={
        <>
          A native DAG engine reacts to database changes, webhooks and schedules. Chain queries,
          functions, conditions and messages without a separate service or a line of glue code.
        </>
      }
      visual={<FlowGraphic />}
    />
  );
}

function FlowGraphic() {
  return (
    <div className="relative">
      <div
        className="pointer-events-none absolute -inset-6 -z-10"
        style={{ background: `radial-gradient(460px circle at 40% 30%, ${ACCENT}22, transparent 70%)` }}
      />
      <div className="panel overflow-hidden">
        <img
          src="/workflow-dag.png"
          alt="A branching order-fulfillment workflow in the Applad DAG builder"
          className="block w-full"
          loading="lazy"
        />
      </div>
    </div>
  );
}

const SERVICES: { icon: LucideIcon; name: string }[] = [
  { icon: Database, name: 'Databases' },
  { icon: ShieldCheck, name: 'Authentication' },
  { icon: FolderOpen, name: 'Storage' },
  { icon: Code2, name: 'Functions' },
  { icon: Mail, name: 'Messaging' },
  { icon: Radio, name: 'Realtime' },
  { icon: FileText, name: 'Content' },
  { icon: Workflow, name: 'Workflows' },
  { icon: Flag, name: 'Feature flags' },
  { icon: KeyRound, name: 'Vault' },
  { icon: Rocket, name: 'Deploy' },
  { icon: Cloud, name: 'Sites & containers' },
];

function Services() {
  return (
    <section id="platform" className="relative py-24">
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background: `linear-gradient(to bottom, var(--color-bg) 3%, transparent 26%, transparent 60%, var(--color-bg) 97%), radial-gradient(900px circle at 50% 42%, ${ACCENT}18, transparent 62%)`,
        }}
      />
      <div className="relative mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="eyebrow">The platform</div>
          <h2 className="mt-3 text-3xl font-bold tracking-tight md:text-[40px]">Everything, one API</h2>
          <p className="mt-4 text-[17px] text-muted">
            Every service shares the same auth model, console and REST API, so they compose instead
            of fighting each other. Reach for one, or all of them.
          </p>
        </div>
        <div className="mx-auto mt-12 grid max-w-4xl grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 md:grid-cols-4">
          {SERVICES.map((s) => (
            <div key={s.name} className="group flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-white/[0.06] bg-white/[0.03] text-muted transition-colors group-hover:text-[color:var(--color-accent)]">
                <s.icon size={16} />
              </span>
              <span className="text-sm text-text">{s.name}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function Deploy() {
  // install.sh is served from this same host, so follow it: applad.io.localhost
  // in dev, applad.io in production.
  const installOrigin = typeof window !== 'undefined' ? window.location.origin : 'https://applad.io';
  const installCmd = `curl -fsSL ${installOrigin}/install.sh | sh`;
  return (
    <section id="pricing" className="mx-auto max-w-6xl px-6 py-24">
      <div className="mx-auto mb-12 max-w-2xl text-center">
        <div className="eyebrow">Pricing</div>
        <h2 className="mt-3 text-3xl font-bold tracking-tight md:text-[40px]">Cloud or self-hosted</h2>
        <p className="mt-4 text-[17px] text-muted">
          Applad is open-source and free to run yourself. Prefer it managed? Start Applad Cloud with a
          free trial. Move between them whenever you like.
        </p>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <div className="panel flex flex-col p-8">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl" style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--color-text)' }}>
            <Server size={21} />
          </div>
          <h3 className="mt-5 text-xl font-semibold">Self-hosted</h3>
          <div className="mt-3 flex items-baseline gap-2">
            <span className="text-4xl font-bold tracking-tight">Free</span>
            <span className="text-sm text-muted">forever, BSD-3-Clause licensed</span>
          </div>
          <p className="mt-3 text-muted">Own every byte. Your infrastructure, your data, every feature included.</p>
          <div className="mt-5 rounded-xl border border-white/[0.06] bg-black/30 p-4">
            <div className="mb-2 flex items-center justify-between text-[11px] text-muted">
              <span className="flex items-center gap-2">
                <Terminal size={13} />
                one command
              </span>
              <CopyButton text={installCmd} />
            </div>
            <pre className="overflow-x-auto font-[family-name:var(--font-mono)] text-[13px] leading-relaxed">
              <code>
                <span className="select-none text-muted">$ </span>
                <span className="text-text">{installCmd}</span>
              </code>
            </pre>
          </div>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer" className="mt-7 inline-flex w-fit items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-text transition-colors hover:bg-surface">
            <Github size={15} />
            View on GitHub
          </a>
        </div>

        <div className="panel flex flex-col p-8">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl" style={{ background: `${ACCENT}26`, color: ACCENT }}>
            <Cloud size={21} />
          </div>
          <h3 className="mt-5 text-xl font-semibold">Applad Cloud</h3>
          <div className="mt-3 flex items-baseline gap-2">
            <span className="text-4xl font-bold tracking-tight">Free trial</span>
            <span className="text-sm text-muted">then paid, pricing coming soon</span>
          </div>
          <p className="mt-3 text-muted">Fully managed. We run the servers, backups and scaling. You build.</p>
          <ul className="mt-5 flex flex-col gap-2.5 text-sm text-muted">
            {['Try every feature during your trial', 'Managed backups, updates & scaling', 'Global regions'].map((t) => (
              <li key={t} className="flex items-center gap-2.5">
                <Check size={15} style={{ color: ACCENT }} />
                {t}
              </li>
            ))}
          </ul>
          <a href={CONSOLE_URL} className="mt-7 inline-flex w-fit items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-90" style={{ background: ACCENT }}>
            Start free trial
            <ArrowRight size={15} />
          </a>
        </div>
      </div>
      <p className="mt-8 text-center text-sm text-muted">
        Need SSO, audit logs and an SLA?{' '}
        <a href="mailto:hello@applad.io" className="text-text underline-offset-4 hover:underline">
          Contact us about Enterprise
        </a>
      </p>
    </section>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      ta.remove();
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };
  return (
    <button
      type="button"
      onClick={copy}
      aria-label={copied ? 'Copied' : 'Copy command'}
      className="flex items-center gap-1 rounded-md px-1.5 py-0.5 font-medium transition-colors hover:text-text"
      style={{ color: copied ? ACCENT : undefined }}
    >
      {copied ? <Check size={12} /> : <Copy size={12} />}
      {copied ? 'Copied' : 'Copy'}
    </button>
  );
}

function CTA() {
  return (
    <section className="relative overflow-hidden">
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background: `linear-gradient(to bottom, var(--color-bg) 4%, transparent 30%, transparent 62%, var(--color-bg) 98%), radial-gradient(760px circle at 50% 52%, ${ACCENT}22, transparent 62%)`,
        }}
      />
      <div className="relative mx-auto max-w-3xl px-6 py-28 text-center">
        <h2 className="text-3xl font-bold tracking-tight md:text-[44px]">Go from idea to production today.</h2>
        <p className="mx-auto mt-4 max-w-md text-[17px] text-muted">
          Create a project, ask the assistant to scaffold it, and connect your app with one of the SDKs.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <a href={CONSOLE_URL} className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-opacity hover:opacity-90" style={{ background: ACCENT }}>
            Open the console
            <ArrowRight size={16} />
          </a>
          <a href={DOCS_URL} className="inline-flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-text transition-colors hover:bg-surface">
            Read the docs
          </a>
        </div>
      </div>
    </section>
  );
}

function Footer() {
  const groups: { title: string; links: { label: string; href: string; external?: boolean }[] }[] = [
    {
      title: 'Product',
      links: [
        { label: 'Platform', href: '#platform' },
        { label: 'Pricing', href: '#pricing' },
        { label: 'Cloud', href: CONSOLE_URL },
        { label: 'Self-hosted', href: GITHUB_URL, external: true },
      ],
    },
    {
      title: 'Developers',
      links: [
        { label: 'Documentation', href: DOCS_URL },
        { label: 'Quickstart', href: `${DOCS_URL}/getting-started` },
        { label: 'SDKs', href: `${DOCS_URL}/sdks` },
        { label: 'GitHub', href: GITHUB_URL, external: true },
      ],
    },
    {
      title: 'Company',
      links: [
        { label: 'Sign in', href: CONSOLE_URL },
        { label: 'Get started', href: CONSOLE_URL },
      ],
    },
  ];
  return (
    <footer className="mt-12">
      <div className="mx-auto max-w-6xl px-6 pb-10 pt-14">
        <div className="grid gap-10 md:grid-cols-[1.4fr_repeat(3,1fr)]">
          <div className="max-w-xs">
            <a href="/" className="flex items-center gap-2.5">
              <img src="/favicon.png" alt="" className="h-7 w-7 rounded-[7px] object-cover" />
              <span className="text-[17px] font-bold tracking-tight">applad</span>
            </a>
            <p className="mt-4 text-sm leading-relaxed text-muted">
              The complete AI-native development infrastructure for your apps. Plan, build, test,
              deploy and monitor. Cloud or self-hosted.
            </p>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              className="mt-5 inline-flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-muted transition-colors hover:text-text"
            >
              <Github size={15} />
              Star on GitHub
            </a>
          </div>
          {groups.map((g) => (
            <div key={g.title}>
              <div className="text-xs font-semibold uppercase tracking-[0.14em] text-muted/70">{g.title}</div>
              <ul className="mt-4 space-y-3 text-sm">
                {g.links.map((l) => (
                  <li key={l.label}>
                    <a
                      href={l.href}
                      {...(l.external ? { target: '_blank', rel: 'noreferrer' } : {})}
                      className="text-muted transition-colors hover:text-text"
                    >
                      {l.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-12 flex flex-col items-center justify-between gap-3 border-t border-border pt-6 text-sm text-muted sm:flex-row">
          <span>© 2026 Mittolabs LTD. All rights reserved.</span>
          <a href={STATUS_URL} className="flex items-center gap-1.5 transition-colors hover:text-text">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: '#10b981' }} />
            All systems operational
          </a>
        </div>
      </div>
    </footer>
  );
}
