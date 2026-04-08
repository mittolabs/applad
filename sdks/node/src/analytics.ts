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
}
