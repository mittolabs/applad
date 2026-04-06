"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Workflows = void 0;
class Workflows {
    constructor(client) {
        this.client = client;
    }
    create(name, opts) {
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
    get(workflowId) {
        return this.client.call('GET', `/workflows/${workflowId}`);
    }
    update(workflowId, data) {
        return this.client.call('PUT', `/workflows/${workflowId}`, data);
    }
    delete(workflowId) {
        return this.client.call('DELETE', `/workflows/${workflowId}`);
    }
    execute(workflowId, triggerData) {
        return this.client.call('POST', `/workflows/${workflowId}/execute`, {
            triggerData: triggerData ?? {},
        });
    }
    listExecutions(workflowId) {
        return this.client.call('GET', `/workflows/${workflowId}/executions`);
    }
    getExecution(workflowId, executionId) {
        return this.client.call('GET', `/workflows/${workflowId}/executions/${executionId}`);
    }
}
exports.Workflows = Workflows;
