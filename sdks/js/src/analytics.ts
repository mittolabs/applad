import type { Applad } from './client';

export class Analytics {
  constructor(private client: Applad) {}

  /** Track a custom event. */
  trackEvent(event: string, properties?: Record<string, unknown>) {
    return this.client.call('POST', '/analytics/events', { event, ...(properties && { properties }) });
  }

  /** List recorded events with optional filters. */
  listEvents(params?: { event?: string; startDate?: string; endDate?: string; limit?: number }) {
    const qs = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : '';
    return this.client.call('GET', `/analytics/events${qs}`);
  }

  /** Get aggregated event stats. */
  getStats(params?: { event?: string; startDate?: string; endDate?: string; interval?: string }) {
    const qs = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : '';
    return this.client.call('GET', `/analytics/stats${qs}`);
  }

  /** Get real-time active user count. */
  getRealtimeCount() {
    return this.client.call('GET', '/analytics/realtime');
  }
}
