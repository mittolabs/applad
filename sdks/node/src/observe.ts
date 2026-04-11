import type { ApplAdServer } from './client';

export class Observe {
  constructor(private client: ApplAdServer) {}

  // ── Overview ────────────────────────────────────────────────────────────────

  getOverview() {
    return this.client.call('GET', '/observe/overview');
  }

  // ── Errors ──────────────────────────────────────────────────────────────────

  captureError(title: string, opts: {
    errorType?: string;
    level?: string;
    stackTrace?: string;
    breadcrumbs?: Record<string, unknown>[];
    userContext?: Record<string, unknown>;
    requestContext?: Record<string, unknown>;
    runtimeContext?: Record<string, unknown>;
    tags?: Record<string, unknown>;
    environment?: string;
    release?: string;
  } = {}) {
    return this.client.call('POST', '/observe/errors', { title, ...opts });
  }

  listErrors(params: { status?: string; level?: string; limit?: number } = {}) {
    const q = new URLSearchParams();
    if (params.status) q.set('status', params.status);
    if (params.level) q.set('level', params.level);
    if (params.limit) q.set('limit', String(params.limit));
    const qs = q.toString();
    return this.client.call('GET', `/observe/errors${qs ? '?' + qs : ''}`);
  }

  getError(errorId: string) {
    return this.client.call('GET', `/observe/errors/${errorId}`);
  }

  resolveError(errorId: string) {
    return this.client.call('PATCH', `/observe/errors/${errorId}/resolve`);
  }

  ignoreError(errorId: string) {
    return this.client.call('PATCH', `/observe/errors/${errorId}/ignore`);
  }

  setErrorPriority(errorId: string, priority: 'P1' | 'P2' | 'P3' | 'P4') {
    return this.client.call('PATCH', `/observe/errors/${errorId}/priority`, { priority });
  }

  assignError(errorId: string, assignee: string) {
    return this.client.call('PATCH', `/observe/errors/${errorId}/assign`, { assignee });
  }

  // ── Logs ────────────────────────────────────────────────────────────────────

  captureLog(message: string, opts: {
    level?: string;
    source?: string;
    environment?: string;
    release?: string;
    meta?: Record<string, unknown>;
    traceId?: string;
    spanId?: string;
  } = {}) {
    return this.client.call('POST', '/observe/logs', { message, ...opts });
  }

  listLogs(params: { level?: string; source?: string; limit?: number } = {}) {
    const q = new URLSearchParams();
    if (params.level) q.set('level', params.level);
    if (params.source) q.set('source', params.source);
    if (params.limit) q.set('limit', String(params.limit));
    const qs = q.toString();
    return this.client.call('GET', `/observe/logs${qs ? '?' + qs : ''}`);
  }

  // ── Performance ─────────────────────────────────────────────────────────────

  getPerformance() {
    return this.client.call('GET', '/observe/performance');
  }

  recordPerf(path: string, opts: {
    method?: string;
    p50Ms?: number; p75Ms?: number; p95Ms?: number; p99Ms?: number;
    rps?: number; errorPct?: number; reqCount?: number;
  } = {}) {
    return this.client.call('POST', '/observe/performance', { path, ...opts });
  }

  reportWebVitals(vitals: {
    pageUrl?: string;
    lcp?: number; fid?: number; cls?: number;
    ttfb?: number; fcp?: number; inp?: number;
  }) {
    return this.client.call('POST', '/observe/performance/vitals', vitals);
  }

  // ── Releases ────────────────────────────────────────────────────────────────

  listReleases() {
    return this.client.call('GET', '/observe/releases');
  }

  createRelease(version: string, opts: { environment?: string; commits?: Record<string, unknown>[] } = {}) {
    return this.client.call('POST', '/observe/releases', { version, ...opts });
  }

  // ── Cron checkin ────────────────────────────────────────────────────────────

  cronCheckin(monitorId: string, opts: { status?: 'ok' | 'failed'; durationMs?: number; errorMsg?: string } = {}) {
    return this.client.call('POST', `/observe/crons/${monitorId}/checkin`, opts);
  }
}
