import type { Applad } from './client';

export interface CaptureErrorOptions {
  errorType?: string;
  level?: 'fatal' | 'error' | 'warning' | 'info';
  stackTrace?: string;
  breadcrumbs?: Record<string, unknown>[];
  userContext?: Record<string, unknown>;
  requestContext?: Record<string, unknown>;
  runtimeContext?: Record<string, unknown>;
  tags?: Record<string, unknown>;
  environment?: string;
  release?: string;
}

export interface CaptureLogOptions {
  level?: 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  source?: string;
  environment?: string;
  release?: string;
  meta?: Record<string, unknown>;
  traceId?: string;
  spanId?: string;
}

export interface RecordPerfOptions {
  method?: string;
  p50Ms?: number;
  p75Ms?: number;
  p95Ms?: number;
  p99Ms?: number;
  rps?: number;
  errorPct?: number;
  reqCount?: number;
}

export interface WebVitals {
  pageUrl?: string;
  lcp?: number;
  fid?: number;
  cls?: number;
  ttfb?: number;
  fcp?: number;
  inp?: number;
}

export class Observe {
  constructor(private client: Applad) {}

  // ── Overview ────────────────────────────────────────────────────────────────

  getOverview() {
    return this.client.call('GET', '/observe/overview');
  }

  // ── Errors ──────────────────────────────────────────────────────────────────

  captureError(title: string, options: CaptureErrorOptions = {}) {
    return this.client.call('POST', '/observe/errors', { title, ...options });
  }

  listErrors(params: { status?: string; level?: string; limit?: number } = {}) {
    const q = new URLSearchParams();
    if (params.status) q.set('status', params.status);
    if (params.level) q.set('level', params.level);
    if (params.limit) q.set('limit', String(params.limit));
    return this.client.call('GET', `/observe/errors?${q}`);
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

  unresolveError(errorId: string) {
    return this.client.call('PATCH', `/observe/errors/${errorId}/unresolve`);
  }

  setErrorPriority(errorId: string, priority: 'P1' | 'P2' | 'P3' | 'P4') {
    return this.client.call('PATCH', `/observe/errors/${errorId}/priority`, { priority });
  }

  assignError(errorId: string, assignee: string) {
    return this.client.call('PATCH', `/observe/errors/${errorId}/assign`, { assignee });
  }

  addNote(errorId: string, text: string) {
    return this.client.call('POST', `/observe/errors/${errorId}/activity`, { text });
  }

  bulkResolve(ids: string[]) {
    return this.client.call('POST', '/observe/errors/bulk', { ids, action: 'resolve' });
  }

  bulkIgnore(ids: string[]) {
    return this.client.call('POST', '/observe/errors/bulk', { ids, action: 'ignore' });
  }

  // ── Logs ────────────────────────────────────────────────────────────────────

  captureLog(message: string, options: CaptureLogOptions = {}) {
    return this.client.call('POST', '/observe/logs', { message, ...options });
  }

  listLogs(params: { level?: string; source?: string; limit?: number } = {}) {
    const q = new URLSearchParams();
    if (params.level) q.set('level', params.level);
    if (params.source) q.set('source', params.source);
    if (params.limit) q.set('limit', String(params.limit));
    return this.client.call('GET', `/observe/logs?${q}`);
  }

  // ── Performance ─────────────────────────────────────────────────────────────

  getPerformance() {
    return this.client.call('GET', '/observe/performance');
  }

  recordPerf(path: string, options: RecordPerfOptions = {}) {
    return this.client.call('POST', '/observe/performance', { path, ...options });
  }

  reportWebVitals(vitals: WebVitals) {
    return this.client.call('POST', '/observe/performance/vitals', vitals);
  }

  // ── Releases ────────────────────────────────────────────────────────────────

  listReleases() {
    return this.client.call('GET', '/observe/releases');
  }

  createRelease(version: string, options: { environment?: string; commits?: Record<string, unknown>[] } = {}) {
    return this.client.call('POST', '/observe/releases', { version, ...options });
  }

  getRelease(releaseId: string) {
    return this.client.call('GET', `/observe/releases/${releaseId}`);
  }

  // ── Replays ─────────────────────────────────────────────────────────────────

  listReplays(limit?: number) {
    const q = limit ? `?limit=${limit}` : '';
    return this.client.call('GET', `/observe/replays${q}`);
  }

  createReplay(data: {
    sessionId?: string;
    userId?: string;
    user?: string;
    url?: string;
    browser?: string;
    os?: string;
    country?: string;
    durationSecs?: number;
    errorCount?: number;
    hasRageClick?: boolean;
    hasDeadClick?: boolean;
    events?: Record<string, unknown>[];
    network?: Record<string, unknown>[];
    console?: Record<string, unknown>[];
  }) {
    return this.client.call('POST', '/observe/replays', data);
  }

  getReplay(replayId: string) {
    return this.client.call('GET', `/observe/replays/${replayId}`);
  }

  // ── Uptime ──────────────────────────────────────────────────────────────────

  listMonitors() {
    return this.client.call('GET', '/observe/uptime');
  }

  createMonitor(name: string, url: string, options: { checkType?: string; intervalSecs?: number; keyword?: string } = {}) {
    return this.client.call('POST', '/observe/uptime', { name, url, ...options });
  }

  deleteMonitor(monitorId: string) {
    return this.client.call('DELETE', `/observe/uptime/${monitorId}`);
  }

  // ── Crons ───────────────────────────────────────────────────────────────────

  listCronMonitors() {
    return this.client.call('GET', '/observe/crons');
  }

  createCronMonitor(name: string, schedule: string, options: { timezone?: string; gracePeriod?: number } = {}) {
    return this.client.call('POST', '/observe/crons', { name, schedule, ...options });
  }

  toggleCronMonitor(monitorId: string) {
    return this.client.call('PATCH', `/observe/crons/${monitorId}/toggle`);
  }

  deleteCronMonitor(monitorId: string) {
    return this.client.call('DELETE', `/observe/crons/${monitorId}`);
  }

  cronCheckin(monitorId: string, options: { status?: 'ok' | 'failed'; durationMs?: number; errorMsg?: string } = {}) {
    return this.client.call('POST', `/observe/crons/${monitorId}/checkin`, options);
  }

  // ── Alerts ──────────────────────────────────────────────────────────────────

  listAlerts() {
    return this.client.call('GET', '/observe/alerts');
  }

  createAlertRule(name: string, options: {
    metric?: string;
    operator?: string;
    threshold?: number;
    window?: string;
    severity?: string;
    channel?: string;
  } = {}) {
    return this.client.call('POST', '/observe/alerts', { name, ...options });
  }

  toggleAlertRule(ruleId: string) {
    return this.client.call('PATCH', `/observe/alerts/${ruleId}/toggle`);
  }

  deleteAlertRule(ruleId: string) {
    return this.client.call('DELETE', `/observe/alerts/${ruleId}`);
  }
}
