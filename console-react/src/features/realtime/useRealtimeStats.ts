import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';

/* Shared realtime stats poll — GET /realtime/stats every 5s.
 * Response: { connections, channels, channelList: [{ channel, subscribers }] }.
 * Both tabs subscribe to the same query key so TanStack dedupes the request. */
export function useRealtimeStats(projectId: string | undefined) {
  return useQuery({
    queryKey: ['realtime', 'stats', projectId],
    queryFn: async () =>
      (await api.get('/realtime/stats')).data as Record<string, unknown>,
    refetchInterval: 5000,
  });
}
