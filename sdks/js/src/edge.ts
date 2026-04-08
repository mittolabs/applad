import type { Applad } from './client';

export class Edge {
  constructor(private client: Applad) {}

  /** Create an edge function. */
  create(name: string, code: string, opts?: { route?: string; timeout?: number }) {
    return this.client.call('POST', '/edge/functions', { name, code, ...opts });
  }

  /** List all edge functions. */
  list() {
    return this.client.call('GET', '/edge/functions');
  }

  /** Get an edge function by ID. */
  get(functionId: string) {
    return this.client.call('GET', `/edge/functions/${functionId}`);
  }

  /** Update an edge function. */
  update(functionId: string, opts: { name?: string; code?: string; route?: string; timeout?: number }) {
    return this.client.call('PUT', `/edge/functions/${functionId}`, opts);
  }

  /** Delete an edge function. */
  delete(functionId: string) {
    return this.client.call('DELETE', `/edge/functions/${functionId}`);
  }

  /** Invoke an edge function. */
  invoke(functionId: string, data?: Record<string, unknown>) {
    return this.client.call('POST', `/edge/functions/${functionId}/invoke`, data ?? {});
  }

  /** List edge function executions. */
  listExecutions(functionId: string) {
    return this.client.call('GET', `/edge/functions/${functionId}/executions`);
  }
}
