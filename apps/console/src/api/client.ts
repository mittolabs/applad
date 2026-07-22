import axios, { AxiosError, type AxiosInstance } from 'axios';

/*
 * API client — ports console/lib/core/api/client.dart.
 * Base URL: relative "/v1" (works behind the proxy and via the Vite dev proxy);
 * override with VITE_API_URL for direct access.
 * Headers set imperatively (mirrors the Dart client's header setters):
 *   Authorization: Bearer <jwt>      setAuthToken
 *   X-Applad-Project: <projectId>    setProject
 *   X-Console-User-ID/Email/Name     setConsoleUser
 */

const TOKEN_KEY = 'applad_console_token';

function baseURL(): string {
  const raw = import.meta.env.VITE_API_URL ?? '/v1';
  return raw.endsWith('/') ? raw.slice(0, -1) : raw;
}

export const api: AxiosInstance = axios.create({
  baseURL: baseURL(),
  headers: { 'Content-Type': 'application/json' },
});

export function setAuthToken(token: string | null) {
  if (token) api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  else delete api.defaults.headers.common['Authorization'];
}

export function setProject(projectId: string | null) {
  if (projectId) api.defaults.headers.common['X-Applad-Project'] = projectId;
  else delete api.defaults.headers.common['X-Applad-Project'];
}

export function setConsoleUser(u: { id: string; email: string; name: string } | null) {
  if (u) {
    api.defaults.headers.common['X-Console-User-ID'] = u.id;
    api.defaults.headers.common['X-Console-User-Email'] = u.email;
    api.defaults.headers.common['X-Console-User-Name'] = u.name;
  } else {
    delete api.defaults.headers.common['X-Console-User-ID'];
    delete api.defaults.headers.common['X-Console-User-Email'];
    delete api.defaults.headers.common['X-Console-User-Name'];
  }
}

// Restore token from localStorage on module load so refresh keeps the session.
const saved = localStorage.getItem(TOKEN_KEY);
if (saved) setAuthToken(saved);

/**
 * On 401 the token is cleared and the router guard bounces to /login. We
 * surface that here via a subscriber the auth store registers, to avoid a hard
 * import cycle.
 *
 * 403 deliberately does not sign anybody out. It means the credentials were
 * accepted and the action was refused — a row the caller cannot write, a
 * closed signup — and ending the session over it logs people out of the whole
 * console because one resource said no.
 */
let onAuthError: (() => void) | null = null;
export function registerAuthErrorHandler(fn: () => void) {
  onAuthError = fn;
}

api.interceptors.response.use(
  (res) => res,
  (error: AxiosError) => {
    const status = error.response?.status;
    if (status === 401) onAuthError?.();
    return Promise.reject(error);
  },
);

/** Human-readable error message — ports app_error_state.dart friendlyError(). */
export function friendlyError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as
      | { message?: string; error?: string }
      | undefined;
    if (data?.message) return data.message;
    if (data?.error) return data.error;
    const status = error.response?.status;
    if (status === 401 || status === 403) return 'Access denied.';
    if (status && status >= 500) return 'Server error. Please try again.';
    if (error.code === 'ERR_NETWORK') return 'Network error. Check your connection.';
    if (error.code === 'ECONNABORTED') return 'Request timed out.';
    if (status) return `Server returned an error (${status}).`;
  }
  if (error instanceof Error) return error.message;
  return 'Something went wrong.';
}

export { TOKEN_KEY };
