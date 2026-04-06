"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Deploy = void 0;
class Deploy {
    constructor(client) {
        this.client = client;
    }
    create(name, type, config) {
        return this.client.call('POST', '/deploy', { name, type, config: config ?? {} });
    }
    list() {
        return this.client.call('GET', '/deploy');
    }
    get(deploymentId) {
        return this.client.call('GET', `/deploy/${deploymentId}`);
    }
    update(deploymentId, data) {
        return this.client.call('PUT', `/deploy/${deploymentId}`, data);
    }
    updateStatus(deploymentId, status) {
        return this.client.call('PATCH', `/deploy/${deploymentId}`, { status });
    }
    delete(deploymentId) {
        return this.client.call('DELETE', `/deploy/${deploymentId}`);
    }
}
exports.Deploy = Deploy;
