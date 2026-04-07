import { Linking } from 'react-native';
import type { ApplAdClient } from './client';

export class Auth {
  constructor(private client: ApplAdClient) {}

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

  /** Create an email session (login). Returns the session and persists the JWT. */
  async createEmailSession(email: string, password: string) {
    const session = await this.client.call<{ jwt: string }>('POST', '/account/sessions/email', {
      email,
      password,
    });
    if (session?.jwt) {
      await this.client.setSession(session.jwt);
    }
    return session;
  }

  /** Create an anonymous session. */
  async createAnonymousSession() {
    const session = await this.client.call<{ jwt: string }>('POST', '/account/sessions/anonymous');
    if (session?.jwt) {
      await this.client.setSession(session.jwt);
    }
    return session;
  }

  /** Create a phone session (sends OTP). */
  createPhoneSession(phone: string) {
    return this.client.call('POST', '/account/sessions/phone', { phone });
  }

  /** Verify a phone OTP to complete the phone session. */
  async verifyPhoneOTP(userId: string, secret: string) {
    const session = await this.client.call<{ jwt: string }>('POST', '/account/sessions/phone/verify', {
      userId,
      secret,
    });
    if (session?.jwt) {
      await this.client.setSession(session.jwt);
    }
    return session;
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

  /** Delete all sessions and clear local session. */
  async deleteSessions() {
    await this.client.call('DELETE', '/account/sessions');
    await this.client.clearSession();
  }

  /** Get a short-lived JWT. */
  async getJWT(): Promise<string> {
    const res = await this.client.call<{ jwt: string }>('POST', '/account/jwt');
    return res.jwt;
  }

  /** Logout: delete current session and clear local storage. */
  async logout() {
    try {
      await this.client.call('DELETE', '/account/sessions/current');
    } catch {
      // ignore error if session already expired
    }
    await this.client.clearSession();
  }

  // ---- OAuth helpers for React Native ----

  /**
   * Build an OAuth2 redirect URL for the given provider.
   * Opens the URL in the device browser via React Native Linking.
   *
   * @param provider - OAuth provider name (e.g. "google", "github", "apple")
   * @param successUrl - Deep link URL your app handles on success (e.g. "myapp://oauth/success")
   * @param failureUrl - Deep link URL your app handles on failure (e.g. "myapp://oauth/failure")
   */
  getOAuthUrl(provider: string, successUrl: string, failureUrl: string): string {
    const params = new URLSearchParams({
      project: this.client.projectId,
      success: successUrl,
      failure: failureUrl,
    });
    return `${this.client.endpoint}/v1/account/sessions/oauth2/${provider}?${params.toString()}`;
  }

  /**
   * Open an OAuth2 login flow in the device browser.
   * Your app should handle the deep link callback to capture the session token.
   */
  async openOAuth(provider: string, successUrl: string, failureUrl: string): Promise<void> {
    const url = this.getOAuthUrl(provider, successUrl, failureUrl);
    await Linking.openURL(url);
  }
}
