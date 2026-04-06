import type { ApplAdServer } from './client';

export class Teams {
  constructor(private client: ApplAdServer) {}

  create(name: string, opts?: { teamId?: string; roles?: string[] }) {
    return this.client.call('POST', '/teams', {
      teamId: opts?.teamId ?? 'unique()',
      name,
      roles: opts?.roles ?? [],
    });
  }

  list() {
    return this.client.call('GET', '/teams');
  }

  get(teamId: string) {
    return this.client.call('GET', `/teams/${teamId}`);
  }

  update(teamId: string, name: string) {
    return this.client.call('PUT', `/teams/${teamId}`, { name });
  }

  delete(teamId: string) {
    return this.client.call('DELETE', `/teams/${teamId}`);
  }

  // --- Memberships ---

  createMembership(teamId: string, email: string, roles: string[], opts?: { userId?: string; name?: string }) {
    return this.client.call('POST', `/teams/${teamId}/memberships`, {
      email,
      roles,
      ...(opts?.userId && { userId: opts.userId }),
      ...(opts?.name && { name: opts.name }),
    });
  }

  listMemberships(teamId: string) {
    return this.client.call('GET', `/teams/${teamId}/memberships`);
  }

  getMembership(teamId: string, membershipId: string) {
    return this.client.call('GET', `/teams/${teamId}/memberships/${membershipId}`);
  }

  updateMembership(teamId: string, membershipId: string, roles: string[]) {
    return this.client.call('PATCH', `/teams/${teamId}/memberships/${membershipId}`, { roles });
  }

  deleteMembership(teamId: string, membershipId: string) {
    return this.client.call('DELETE', `/teams/${teamId}/memberships/${membershipId}`);
  }
}
