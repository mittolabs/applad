import type { ApplAdServer } from './client';

export class Analytics {
  constructor(private client: ApplAdServer) {}

  trackEvent(event: string, properties?: Record<string, unknown>) {
    return this.client.call('POST', '/analytics/events', { event, ...(properties && { properties }) });
  }

  listEvents(params?: { event?: string; startDate?: string; endDate?: string; limit?: number }) {
    const qs = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : '';
    return this.client.call('GET', `/analytics/events${qs}`);
  }

  getStats(params?: { event?: string; startDate?: string; endDate?: string; interval?: string }) {
    const qs = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : '';
    return this.client.call('GET', `/analytics/stats${qs}`);
  }

  getRealtimeCount() {
    return this.client.call('GET', '/analytics/realtime');
  }

  /** Summary for the Analytics landing page: events, active users, request
   * latency and average uptime, over the last 24 hours. */
  getOverview() {
    return this.client.call('GET', '/analytics/overview');
  }

  /** Per-route request latency measured by the platform over the last 24h. */
  getPerformance() {
    return this.client.call('GET', '/analytics/performance');
  }

  // ── Uptime monitors ───────────────────────────────────────────────────────

  listMonitors() {
    return this.client.call('GET', '/analytics/uptime');
  }

  createMonitor(options: {
    name: string;
    url: string;
    checkType?: string;
    intervalSecs?: number;
    keyword?: string;
  }) {
    return this.client.call('POST', '/analytics/uptime', options);
  }

  deleteMonitor(monitorId: string) {
    return this.client.call('DELETE', `/analytics/uptime/${monitorId}`);
  }

  // ── Cron monitors ─────────────────────────────────────────────────────────

  listCronMonitors() {
    return this.client.call('GET', '/analytics/crons');
  }

  createCronMonitor(options: {
    name: string;
    schedule: string;
    timezone?: string;
    gracePeriod?: number;
  }) {
    return this.client.call('POST', '/analytics/crons', options);
  }

  deleteCronMonitor(monitorId: string) {
    return this.client.call('DELETE', `/analytics/crons/${monitorId}`);
  }

  /** Report a run of a scheduled job. A monitor that hears nothing within its
   * grace period is marked missed. */
  cronCheckin(
    monitorId: string,
    options?: { status?: string; durationMs?: number; errorMsg?: string },
  ) {
    return this.client.call('POST', `/analytics/crons/${monitorId}/checkin`, options ?? {});
  }
}
