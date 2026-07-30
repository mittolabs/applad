import type { ApplAdServer } from './client';

/**
 * Edge functions.
 *
 * EXPERIMENTAL / NOT IMPLEMENTED on this build: the backend Edge service is
 * not yet wired up, so `invoke`, `create` and the other methods here currently
 * return HTTP 501 (Not Implemented). The API surface is kept stable for forward
 * compatibility; do not rely on it in production yet.
 */
export class Edge {
  constructor(private client: ApplAdServer) {}

  create(name: string, code: string, opts?: { route?: string; timeout?: number }) {
    return this.client.call('POST', '/edge/functions', { name, code, ...opts });
  }

  list() {
    return this.client.call('GET', '/edge/functions');
  }

  get(functionId: string) {
    return this.client.call('GET', `/edge/functions/${functionId}`);
  }

  update(functionId: string, opts: { name?: string; code?: string; route?: string; timeout?: number }) {
    return this.client.call('PUT', `/edge/functions/${functionId}`, opts);
  }

  delete(functionId: string) {
    return this.client.call('DELETE', `/edge/functions/${functionId}`);
  }

  invoke(functionId: string, data?: Record<string, unknown>) {
    return this.client.call('POST', `/edge/functions/${functionId}/invoke`, data ?? {});
  }

  listExecutions(functionId: string) {
    return this.client.call('GET', `/edge/functions/${functionId}/executions`);
  }
}
