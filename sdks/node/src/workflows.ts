import type { ApplAdServer } from './client';

export class Workflows {
  constructor(private client: ApplAdServer) {}

  create(
    name: string,
    opts?: {
      description?: string;
      triggerType?: string;
      triggerConfig?: Record<string, unknown>;
      nodes?: unknown[];
      edges?: unknown[];
    }
  ) {
    return this.client.call('POST', '/workflows', {
      name,
      description: opts?.description ?? '',
      triggerType: opts?.triggerType ?? 'manual',
      triggerConfig: opts?.triggerConfig ?? {},
      nodes: opts?.nodes ?? [],
      edges: opts?.edges ?? [],
    });
  }

  list() {
    return this.client.call('GET', '/workflows');
  }

  get(workflowId: string) {
    return this.client.call('GET', `/workflows/${workflowId}`);
  }

  update(workflowId: string, data: Record<string, unknown>) {
    return this.client.call('PUT', `/workflows/${workflowId}`, data);
  }

  delete(workflowId: string) {
    return this.client.call('DELETE', `/workflows/${workflowId}`);
  }

  execute(workflowId: string, triggerData?: Record<string, unknown>) {
    return this.client.call('POST', `/workflows/${workflowId}/execute`, {
      triggerData: triggerData ?? {},
    });
  }

  listExecutions(workflowId: string) {
    return this.client.call('GET', `/workflows/${workflowId}/executions`);
  }

  getExecution(workflowId: string, executionId: string) {
    return this.client.call('GET', `/workflows/${workflowId}/executions/${executionId}`);
  }
}
