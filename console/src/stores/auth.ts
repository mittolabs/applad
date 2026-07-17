import { create } from 'zustand';
import {
  api,
  registerAuthErrorHandler,
  setAuthToken,
  setConsoleUser,
  TOKEN_KEY,
} from '@/api/client';

/*
 * Auth store — ports console/lib/core/providers/auth_provider.dart.
 * Token persisted to localStorage['applad_console_token']. On load we restore
 * it and fetch /console/me. login/signup/loginWithToken/logout mirror the Dart
 * notifier. On 401/403 the api layer calls back here to clear + redirect.
 */

export interface ConsoleUser {
  id: string;
  email: string;
  name: string;
}

function parseUser(data: Record<string, unknown>): ConsoleUser {
  return {
    id: String(data['$id'] ?? data['id'] ?? ''),
    email: String(data['email'] ?? ''),
    name: String(data['name'] ?? ''),
  };
}

interface AuthState {
  token: string | null;
  user: ConsoleUser | null;
  status: 'loading' | 'authenticated' | 'unauthenticated';
  init: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, name: string) => Promise<void>;
  loginWithToken: (token: string) => Promise<void>;
  logout: () => Promise<void>;
}

function persistToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
  setAuthToken(token);
}

async function fetchMe(): Promise<ConsoleUser> {
  const res = await api.get('/console/me');
  const user = parseUser(res.data as Record<string, unknown>);
  setConsoleUser(user);
  return user;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem(TOKEN_KEY),
  user: null,
  status: 'loading',

  init: async () => {
    const token = get().token;
    if (!token) {
      set({ status: 'unauthenticated' });
      return;
    }
    setAuthToken(token);
    try {
      const user = await fetchMe();
      set({ user, status: 'authenticated' });
    } catch {
      persistToken(null);
      set({ token: null, user: null, status: 'unauthenticated' });
    }
  },

  login: async (email, password) => {
    const res = await api.post('/console/login', { email, password });
    const token = String((res.data as { token: string }).token);
    persistToken(token);
    const user = parseUser((res.data as { user: Record<string, unknown> }).user);
    setConsoleUser(user);
    set({ token, user, status: 'authenticated' });
  },

  signup: async (email, password, name) => {
    const res = await api.post('/console/signup', { email, password, name });
    const token = String((res.data as { token: string }).token);
    persistToken(token);
    const user = parseUser((res.data as { user: Record<string, unknown> }).user);
    setConsoleUser(user);
    set({ token, user, status: 'authenticated' });
  },

  loginWithToken: async (token) => {
    persistToken(token);
    set({ token, status: 'loading' });
    try {
      const user = await fetchMe();
      set({ user, status: 'authenticated' });
    } catch {
      persistToken(null);
      set({ token: null, user: null, status: 'unauthenticated' });
    }
  },

  logout: async () => {
    try {
      await api.post('/console/logout');
    } catch {
      /* best-effort */
    }
    persistToken(null);
    setConsoleUser(null);
    set({ token: null, user: null, status: 'unauthenticated' });
  },
}));

// Wire the api layer's 401/403 handler to clear the session.
registerAuthErrorHandler(() => {
  persistToken(null);
  setConsoleUser(null);
  useAuthStore.setState({ token: null, user: null, status: 'unauthenticated' });
});
