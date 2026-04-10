import type { ApplAdServer } from './client';

export class Functions {
  constructor(private client: ApplAdServer) {}

  create(name: string, runtime: string, opts?: { entrypoint?: string; timeout?: number; vars?: Record<string, string>; source?: string; cron?: string }) {
    return this.client.call('POST', '/functions', {
      name,
      runtime,
      entrypoint: opts?.entrypoint ?? 'index.handler',
      timeout: opts?.timeout ?? 15,
      vars: opts?.vars ?? {},
      source: opts?.source ?? '',
      cron: opts?.cron ?? '',
    });
  }

  list() {
    return this.client.call('GET', '/functions');
  }

  get(functionId: string) {
    return this.client.call('GET', `/functions/${functionId}`);
  }

  update(functionId: string, data: Record<string, unknown>) {
    return this.client.call('PUT', `/functions/${functionId}`, data);
  }

  delete(functionId: string) {
    return this.client.call('DELETE', `/functions/${functionId}`);
  }

  execute(functionId: string, data?: Record<string, unknown>) {
    return this.client.call('POST', `/functions/${functionId}/executions`, { data: data ?? {} });
  }

  listExecutions(functionId: string) {
    return this.client.call('GET', `/functions/${functionId}/executions`);
  }

  getExecution(functionId: string, executionId: string) {
    return this.client.call('GET', `/functions/${functionId}/executions/${executionId}`);
  }
}
