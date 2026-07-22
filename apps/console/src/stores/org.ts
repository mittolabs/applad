import { create } from 'zustand';

/* Org store — ports org_provider.dart currentOrgProvider.
 * Persists the selected org id to localStorage['applad_current_org'].
 * Org list itself is fetched via useOrgs() (TanStack Query). */
const KEY = 'applad_current_org';

interface OrgState {
  currentOrgId: string | null;
  setCurrentOrg: (id: string | null) => void;
}

export const useOrgStore = create<OrgState>((set) => ({
  currentOrgId: localStorage.getItem(KEY),
  setCurrentOrg: (id) => {
    if (id) localStorage.setItem(KEY, id);
    else localStorage.removeItem(KEY);
    set({ currentOrgId: id });
  },
}));
