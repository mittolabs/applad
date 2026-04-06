"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Teams = void 0;
class Teams {
    constructor(client) {
        this.client = client;
    }
    create(name, opts) {
        return this.client.call('POST', '/teams', {
            teamId: opts?.teamId ?? 'unique()',
            name,
            roles: opts?.roles ?? [],
        });
    }
    list() {
        return this.client.call('GET', '/teams');
    }
    get(teamId) {
        return this.client.call('GET', `/teams/${teamId}`);
    }
    update(teamId, name) {
        return this.client.call('PUT', `/teams/${teamId}`, { name });
    }
    delete(teamId) {
        return this.client.call('DELETE', `/teams/${teamId}`);
    }
    // --- Memberships ---
    createMembership(teamId, email, roles, opts) {
        return this.client.call('POST', `/teams/${teamId}/memberships`, {
            email,
            roles,
            ...(opts?.userId && { userId: opts.userId }),
            ...(opts?.name && { name: opts.name }),
        });
    }
    listMemberships(teamId) {
        return this.client.call('GET', `/teams/${teamId}/memberships`);
    }
    getMembership(teamId, membershipId) {
        return this.client.call('GET', `/teams/${teamId}/memberships/${membershipId}`);
    }
    updateMembership(teamId, membershipId, roles) {
        return this.client.call('PATCH', `/teams/${teamId}/memberships/${membershipId}`, { roles });
    }
    deleteMembership(teamId, membershipId) {
        return this.client.call('DELETE', `/teams/${teamId}/memberships/${membershipId}`);
    }
}
exports.Teams = Teams;
