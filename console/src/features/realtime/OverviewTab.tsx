import { useState } from 'react';
import type { LucideIcon } from 'lucide-react';
import {
  AlertTriangle,
  Check,
  Copy,
  Database,
  GitBranch,
  Plug,
  Radio,
  Users,
} from 'lucide-react';
import { CodeBlock } from '@/components/code-block';
import { useRealtimeStats } from './useRealtimeStats';

const ACCENT = 'var(--color-accent)';

function StatCard({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="flex-1 rounded-[var(--radius)] border border-border bg-surface p-5">
      <Icon size={16} className="text-text-muted" />
      <div className="mt-3 text-[length:var(--text-h2)] font-bold text-text-primary">{value}</div>
      <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">{label}</div>
    </div>
  );
}

function InfoRow({ icon: Icon, title, body }: { icon: LucideIcon; title: string; body: string }) {
  return (
    <div className="flex items-start gap-3">
      <div className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-[var(--radius-6)] bg-fill text-text-muted">
        <Icon size={14} />
      </div>
      <div className="flex flex-col gap-0.5">
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">{title}</div>
        <div className="text-[length:var(--text-label)] text-text-secondary">{body}</div>
      </div>
    </div>
  );
}

function ChannelPatternRow({ pattern, description }: { pattern: string; description: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(pattern);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius-6)] border border-border bg-surface px-3.5 py-2.5">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className="truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)]"
          style={{ color: ACCENT }}
        >
          {pattern}
        </span>
        <span className="text-[length:var(--text-caption)] text-text-secondary">{description}</span>
      </div>
      <button
        type="button"
        onClick={copy}
        aria-label="Copy pattern"
        className="cursor-pointer text-text-muted transition-colors hover:text-text-primary"
      >
        {copied ? <Check size={13} className="text-status-success" /> : <Copy size={13} />}
      </button>
    </div>
  );
}

function SectionTitle({ children }: { children: string }) {
  return (
    <div className="text-[length:var(--text-control)] font-semibold text-text-primary">{children}</div>
  );
}

export function OverviewTab({ projectId }: { projectId: string | undefined }) {
  const stats = useRealtimeStats(projectId);
  const data = stats.data ?? {};

  const wsProto = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss' : 'ws';
  const wsHost = typeof window !== 'undefined' ? window.location.host : 'localhost';
  const wsUrl = `${wsProto}://${wsHost}/v1/realtime?project=${projectId ?? '{projectId}'}`;

  return (
    <div className="flex flex-col gap-8">
      {/* Stat cards */}
      {stats.isLoading ? (
        <div className="flex h-20 items-center justify-center text-[length:var(--text-body)] text-text-muted">
          Loading stats…
        </div>
      ) : stats.error ? (
        <div className="flex items-center gap-2 rounded-[var(--radius)] border border-status-danger/20 bg-status-danger/[0.08] px-4 py-4">
          <AlertTriangle size={14} className="text-status-danger" />
          <span className="text-[length:var(--text-label)] text-text-secondary">Could not load stats</span>
        </div>
      ) : (
        <div className="flex gap-4">
          <StatCard icon={Users} label="Active connections" value={String(data.connections ?? 0)} />
          <StatCard icon={Radio} label="Active channels" value={String(data.channels ?? 0)} />
        </div>
      )}

      {/* How it works */}
      <div className="flex flex-col gap-3">
        <SectionTitle>How it works</SectionTitle>
        <InfoRow
          icon={Plug}
          title="WebSocket connection"
          body="Clients connect via WebSocket and subscribe to channels. The server pushes events in real time — no polling needed."
        />
        <InfoRow
          icon={Database}
          title="Database changes"
          body="Row inserts, updates, and deletes in your databases are automatically broadcast to subscribers via PostgreSQL LISTEN/NOTIFY."
        />
        <InfoRow
          icon={GitBranch}
          title="Workflow events"
          body="Workflow execution status changes are published to realtime channels so clients can react instantly."
        />
      </div>

      {/* Connection URL */}
      <div className="flex flex-col gap-3">
        <SectionTitle>Connect</SectionTitle>
        <CodeBlock code={wsUrl} language="websocket" />
      </div>

      {/* Channel patterns */}
      <div className="flex flex-col gap-3">
        <SectionTitle>Channel patterns</SectionTitle>
        <div className="flex flex-col gap-2">
          <ChannelPatternRow
            pattern="projects.{projectId}.databases.rows"
            description="All database row changes across the project"
          />
          <ChannelPatternRow
            pattern="databases.{projectId}.{databaseId}"
            description="All row changes in a specific database"
          />
          <ChannelPatternRow
            pattern="databases.{projectId}.{databaseId}.{tableName}"
            description="Row changes in a specific table"
          />
        </div>
      </div>
    </div>
  );
}
