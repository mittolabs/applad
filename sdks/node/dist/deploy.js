"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Deploy = void 0;
// Deploy targets are the deployable unit. The API mounts them under
// /deploy/targets (there is no flat /deploy resource); triggering a deploy
// runs the target as an execution.
class Deploy {
    constructor(client) {
        this.client = client;
    }
    create(name, type, config) {
        return this.client.call('POST', '/deploy/targets', { name, type, ...(config ?? {}) });
    }
    list() {
        return this.client.call('GET', '/deploy/targets');
    }
    get(targetId) {
        return this.client.call('GET', `/deploy/targets/${targetId}`);
    }
    update(targetId, data) {
        return this.client.call('PUT', `/deploy/targets/${targetId}`, data);
    }
    delete(targetId) {
        return this.client.call('DELETE', `/deploy/targets/${targetId}`);
    }
    // Trigger a deploy of the target. Returns the created execution.
    deploy(targetId, options) {
        return this.client.call('POST', `/deploy/targets/${targetId}/executions`, options ?? {});
    }
}
exports.Deploy = Deploy;
