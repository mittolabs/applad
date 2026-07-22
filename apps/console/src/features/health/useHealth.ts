import { useQuery } from '@tanstack/react-query';
import { api, friendlyError } from '@/api/client';

export type HealthCheck = {
  label: string;
  path: string;
  status: string;
  ping: number;
  error?: string;
};

export type HealthData = {
  status: string;
  checks: HealthCheck[];
  timestamp: string;
};

const ENDPOINTS: [string, string][] = [
  ['/health', 'Gateway'],
  ['/health/db', 'PostgreSQL'],
  ['/health/cache', 'Redis'],
];

async function fetchCheck(path: string, label: string): Promise<HealthCheck> {
  try {
    const payload = (await api.get(path)).data as Record<string, unknown>;
    return {
      label,
      path,
      status: String(payload.status ?? 'fail'),
      ping: Number(payload.ping ?? 0),
    };
  } catch (error) {
    return { label, path, status: 'fail', ping: 0, error: friendlyError(error) };
  }
}

/* Health dashboard poll — fans out to /health, /health/db, /health/cache
 * every ~10s and derives an overall status (pass / warn / fail). */
export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: async (): Promise<HealthData> => {
      const checks = await Promise.all(ENDPOINTS.map(([path, label]) => fetchCheck(path, label)));
      const overall = checks.every((c) => c.status === 'pass')
        ? 'pass'
        : checks.some((c) => c.status === 'pass')
          ? 'warn'
          : 'fail';
      return { status: overall, checks, timestamp: new Date().toISOString() };
    },
    refetchInterval: 10000,
  });
}
