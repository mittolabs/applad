import type { ApplAdServer } from './client';

export class Deploy {
  constructor(private client: ApplAdServer) {}

  create(name: string, type: string, config?: Record<string, unknown>) {
    return this.client.call('POST', '/deploy', { name, type, config: config ?? {} });
  }

  list() {
    return this.client.call('GET', '/deploy');
  }

  get(deploymentId: string) {
    return this.client.call('GET', `/deploy/${deploymentId}`);
  }

  update(deploymentId: string, data: Record<string, unknown>) {
    return this.client.call('PUT', `/deploy/${deploymentId}`, data);
  }

  updateStatus(deploymentId: string, status: string) {
    return this.client.call('PATCH', `/deploy/${deploymentId}`, { status });
  }

  delete(deploymentId: string) {
    return this.client.call('DELETE', `/deploy/${deploymentId}`);
  }
}
