import { Radio } from 'lucide-react';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { AppBadge } from '@/components/app-badge';
import { useRealtimeStats } from './useRealtimeStats';

export function ChannelsTab({ projectId }: { projectId: string | undefined }) {
  const stats = useRealtimeStats(projectId);

  if (stats.isLoading) {
    return (
      <div className="flex h-40 items-center justify-center text-[length:var(--text-body)] text-text-muted">
        Loading channels…
      </div>
    );
  }
  if (stats.error) {
    return <ErrorState error={stats.error} onRetry={() => stats.refetch()} />;
  }

  const channels = ((stats.data?.channelList as Record<string, unknown>[] | undefined) ?? []);

  if (channels.length === 0) {
    return (
      <EmptyState
        icon={Radio}
        title="No active channels"
        subtitle="Channels appear here when clients connect and subscribe."
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="text-[length:var(--text-label)] text-text-secondary">
        {channels.length} active channel{channels.length === 1 ? '' : 's'}
      </div>
      <div className="flex flex-col gap-1.5">
        {channels.map((ch, i) => {
          const subs = Number(ch.subscribers ?? 0);
          return (
            <div
              key={String(ch.channel ?? i)}
              className="flex items-center gap-2.5 rounded-[var(--radius-6)] border border-border bg-surface px-3.5 py-2.5"
            >
              <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-status-success" />
              <span className="min-w-0 flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
                {String(ch.channel ?? '')}
              </span>
              <AppBadge label={`${subs} subscriber${subs === 1 ? '' : 's'}`} />
            </div>
          );
        })}
      </div>
    </div>
  );
}
