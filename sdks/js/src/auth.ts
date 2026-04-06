import type { Applad } from './client';

export class Auth {
  constructor(private client: Applad) {}

  /** Create a new account. */
  createAccount(email: string, password: string, opts?: { userId?: string; name?: string }) {
    return this.client.call('POST', '/account', {
      userId: opts?.userId ?? 'unique()',
      email,
      password,
      ...(opts?.name && { name: opts.name }),
    });
  }

  /** Get the currently logged-in user. */
  getAccount() {
    return this.client.call('GET', '/account');
  }

  /** Update current user's name. */
  updateName(name: string) {
    return this.client.call('PATCH', '/account/name', { name });
  }

  /** Update current user's email (requires password). */
  updateEmail(email: string, password: string) {
    return this.client.call('PATCH', '/account/email', { email, password });
  }

  /** Update current user's password. */
  updatePassword(password: string, oldPassword?: string) {
    return this.client.call('PATCH', '/account/password', {
      password,
      ...(oldPassword && { oldPassword }),
    });
  }

  /** Update current user's preferences. */
  updatePrefs(prefs: Record<string, unknown>) {
    return this.client.call('PATCH', '/account/prefs', { prefs });
  }

  /** Delete current user's account. */
  deleteAccount() {
    return this.client.call('DELETE', '/account');
  }

  /** Create an email session (login). */
  createEmailSession(email: string, password: string) {
    return this.client.call('POST', '/account/sessions/email', { email, password });
  }

  /** Create an anonymous session. */
  createAnonymousSession() {
    return this.client.call('POST', '/account/sessions/anonymous');
  }

  /** List all sessions. */
  listSessions() {
    return this.client.call('GET', '/account/sessions');
  }

  /** Get a session by ID. */
  getSession(sessionId: string) {
    return this.client.call('GET', `/account/sessions/${sessionId}`);
  }

  /** Delete a session. */
  deleteSession(sessionId: string) {
    return this.client.call('DELETE', `/account/sessions/${sessionId}`);
  }

  /** Delete all sessions. */
  deleteSessions() {
    return this.client.call('DELETE', '/account/sessions');
  }

  /** Get a short-lived JWT. */
  async getJWT(): Promise<string> {
    const res = await this.client.call<{ jwt: string }>('POST', '/account/jwt');
    return res.jwt;
  }
}
