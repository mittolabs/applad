"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Functions = void 0;
class Functions {
    constructor(client) {
        this.client = client;
    }
    create(name, runtime, opts) {
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
    get(functionId) {
        return this.client.call('GET', `/functions/${functionId}`);
    }
    update(functionId, data) {
        return this.client.call('PUT', `/functions/${functionId}`, data);
    }
    delete(functionId) {
        return this.client.call('DELETE', `/functions/${functionId}`);
    }
    execute(functionId, data) {
        return this.client.call('POST', `/functions/${functionId}/executions`, { data: data ?? {} });
    }
    listExecutions(functionId) {
        return this.client.call('GET', `/functions/${functionId}/executions`);
    }
    getExecution(functionId, executionId) {
        return this.client.call('GET', `/functions/${functionId}/executions/${executionId}`);
    }
}
exports.Functions = Functions;
