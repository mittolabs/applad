import type { ApplAdServer } from './client';

// Deploy targets are the deployable unit. The API mounts them under
// /deploy/targets (there is no flat /deploy resource); triggering a deploy
// runs the target as an execution.
export class Deploy {
  constructor(private client: ApplAdServer) {}

  create(name: string, type: string, config?: Record<string, unknown>) {
    return this.client.call('POST', '/deploy/targets', { name, type, ...(config ?? {}) });
  }

  list() {
    return this.client.call('GET', '/deploy/targets');
  }

  get(targetId: string) {
    return this.client.call('GET', `/deploy/targets/${targetId}`);
  }

  update(targetId: string, data: Record<string, unknown>) {
    return this.client.call('PUT', `/deploy/targets/${targetId}`, data);
  }

  delete(targetId: string) {
    return this.client.call('DELETE', `/deploy/targets/${targetId}`);
  }

  // Trigger a deploy of the target. Returns the created execution.
  deploy(targetId: string, options?: { request?: string; trigger?: string }) {
    return this.client.call('POST', `/deploy/targets/${targetId}/executions`, options ?? {});
  }
}
