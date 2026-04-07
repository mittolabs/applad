import type { ApplAdClient } from './client';

export class Functions {
  constructor(private client: ApplAdClient) {}

  /** Invoke a function by ID. */
  invoke(functionId: string, data?: Record<string, unknown>) {
    return this.client.call('POST', `/functions/${functionId}/executions`, {
      data: data ?? {},
    });
  }

  /** List executions for a function. */
  listExecutions(functionId: string) {
    return this.client.call('GET', `/functions/${functionId}/executions`);
  }

  /** Get a specific execution by ID. */
  getExecution(functionId: string, executionId: string) {
    return this.client.call('GET', `/functions/${functionId}/executions/${executionId}`);
  }
}
