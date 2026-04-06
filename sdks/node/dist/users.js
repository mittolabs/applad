"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Users = void 0;
class Users {
    constructor(client) {
        this.client = client;
    }
    createUser(email, password, opts) {
        return this.client.call('POST', '/users', {
            userId: opts?.userId ?? 'unique()',
            email,
            password,
            ...(opts?.name && { name: opts.name }),
        });
    }
    listUsers(opts) {
        const params = new URLSearchParams();
        if (opts?.limit)
            params.set('limit', String(opts.limit));
        if (opts?.offset)
            params.set('offset', String(opts.offset));
        if (opts?.search)
            params.set('search', opts.search);
        const qs = params.toString();
        return this.client.call('GET', `/users${qs ? `?${qs}` : ''}`);
    }
    getUser(userId) {
        return this.client.call('GET', `/users/${userId}`);
    }
    deleteUser(userId) {
        return this.client.call('DELETE', `/users/${userId}`);
    }
    updateUserName(userId, name) {
        return this.client.call('PATCH', `/users/${userId}/name`, { name });
    }
    updateUserEmail(userId, email) {
        return this.client.call('PATCH', `/users/${userId}/email`, { email });
    }
    updateUserStatus(userId, status) {
        return this.client.call('PATCH', `/users/${userId}/status`, { status });
    }
}
exports.Users = Users;
