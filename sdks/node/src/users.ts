import type { ApplAdServer } from './client';

export class Users {
  constructor(private client: ApplAdServer) {}

  createUser(email: string, password: string, opts?: { userId?: string; name?: string }) {
    return this.client.call('POST', '/users', {
      userId: opts?.userId ?? 'unique()',
      email,
      password,
      ...(opts?.name && { name: opts.name }),
    });
  }

  listUsers(opts?: { limit?: number; offset?: number; search?: string }) {
    const params = new URLSearchParams();
    if (opts?.limit) params.set('limit', String(opts.limit));
    if (opts?.offset) params.set('offset', String(opts.offset));
    if (opts?.search) params.set('search', opts.search);
    const qs = params.toString();
    return this.client.call('GET', `/users${qs ? `?${qs}` : ''}`);
  }

  getUser(userId: string) {
    return this.client.call('GET', `/users/${userId}`);
  }

  deleteUser(userId: string) {
    return this.client.call('DELETE', `/users/${userId}`);
  }

  updateUserName(userId: string, name: string) {
    return this.client.call('PATCH', `/users/${userId}/name`, { name });
  }

  updateUserEmail(userId: string, email: string) {
    return this.client.call('PATCH', `/users/${userId}/email`, { email });
  }

  updateUserStatus(userId: string, status: boolean) {
    return this.client.call('PATCH', `/users/${userId}/status`, { status });
  }
}
