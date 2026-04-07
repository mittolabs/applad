import React from 'react';
import { ApplAdClient, type ApplAdConfig } from './client';
import { Auth } from './auth';
import { Databases } from './databases';
import { Storage } from './storage';
import { Realtime } from './realtime';
import { Flags } from './flags';
import { Functions } from './functions';

// Re-export all classes and types
export { ApplAdClient, type ApplAdConfig } from './client';
export { Auth } from './auth';
export { Databases } from './databases';
export { Storage } from './storage';
export { Realtime, type RealtimeEvent, type RealtimeCallback } from './realtime';
export { Flags } from './flags';
export { Functions } from './functions';

// ---- React Context + Hooks ----

interface ApplAdContextValue {
  client: ApplAdClient;
  auth: Auth;
  databases: Databases;
  storage: Storage;
  realtime: Realtime;
  flags: Flags;
  functions: Functions;
}

const ApplAdContext = React.createContext<ApplAdContextValue | null>(null);

/**
 * Provider component that initializes the ApplAd client and makes all
 * services available via hooks.
 *
 * @example
 * <ApplAdProvider endpoint="https://my-applad.example.com" projectId="my-project">
 *   <App />
 * </ApplAdProvider>
 */
export function ApplAdProvider({
  endpoint,
  projectId,
  children,
}: ApplAdConfig & { children: React.ReactNode }) {
  const value = React.useMemo(() => {
    const client = new ApplAdClient({ endpoint, projectId });
    return {
      client,
      auth: new Auth(client),
      databases: new Databases(client),
      storage: new Storage(client),
      realtime: new Realtime(client),
      flags: new Flags(client),
      functions: new Functions(client),
    };
  }, [endpoint, projectId]);

  return React.createElement(ApplAdContext.Provider, { value }, children);
}

function useApplAdContext(): ApplAdContextValue {
  const ctx = React.useContext(ApplAdContext);
  if (!ctx) {
    throw new Error('useApplad() must be used within an <ApplAdProvider>');
  }
  return ctx;
}

/** Access the full ApplAd client and all services. */
export function useApplad() {
  return useApplAdContext();
}

/** Access the Auth service. */
export function useAuth(): Auth {
  return useApplAdContext().auth;
}

/** Access the Databases service. */
export function useDatabases(): Databases {
  return useApplAdContext().databases;
}

/** Access the Storage service. */
export function useStorage(): Storage {
  return useApplAdContext().storage;
}

/** Access the Realtime service. */
export function useRealtime(): Realtime {
  return useApplAdContext().realtime;
}

/** Access the Flags service. */
export function useFlags(): Flags {
  return useApplAdContext().flags;
}

/** Access the Functions service. */
export function useFunctions(): Functions {
  return useApplAdContext().functions;
}
