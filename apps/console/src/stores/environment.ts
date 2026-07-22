import { create } from 'zustand';

/* Environment store — ports environment_provider.dart currentEnvironmentProvider.
 * Persists selected env id to localStorage['applad_current_env']. The env list
 * (GET /deploy/environments) is fetched via a query hook; default auto-selected. */
const KEY = 'applad_current_env';

interface EnvState {
  currentEnvId: string | null;
  setCurrentEnv: (id: string | null) => void;
}

export const useEnvStore = create<EnvState>((set) => ({
  currentEnvId: localStorage.getItem(KEY),
  setCurrentEnv: (id) => {
    if (id) localStorage.setItem(KEY, id);
    else localStorage.removeItem(KEY);
    set({ currentEnvId: id });
  },
}));
