import { create } from 'zustand';

/* Experiments store — ports experiments_provider.dart.
 * Console-level feature toggles, JSON-persisted to
 * localStorage['applad_experiments']. Gates the AI chat button, etc. */
const KEY = 'applad_experiments';

export type ExperimentKey =
  | 'aiChat'
  | 'search'
  | 'cache'
  | 'edgeFunctions'
  | 'vectors'
  | 'regions';

const DEFAULTS: Record<ExperimentKey, boolean> = {
  aiChat: false,
  search: false,
  cache: false,
  edgeFunctions: false,
  vectors: false,
  regions: false,
};

function readInitial(): Record<ExperimentKey, boolean> {
  try {
    const raw = localStorage.getItem(KEY);
    if (raw) return { ...DEFAULTS, ...JSON.parse(raw) };
  } catch {
    /* ignore malformed */
  }
  return { ...DEFAULTS };
}

interface ExperimentsState {
  flags: Record<ExperimentKey, boolean>;
  toggle: (key: ExperimentKey) => void;
  enableAll: () => void;
  disableAll: () => void;
}

function persist(flags: Record<ExperimentKey, boolean>) {
  localStorage.setItem(KEY, JSON.stringify(flags));
}

export const useExperimentsStore = create<ExperimentsState>((set, get) => ({
  flags: readInitial(),
  toggle: (key) => {
    const flags = { ...get().flags, [key]: !get().flags[key] };
    persist(flags);
    set({ flags });
  },
  enableAll: () => {
    const flags = Object.fromEntries(
      Object.keys(DEFAULTS).map((k) => [k, true]),
    ) as Record<ExperimentKey, boolean>;
    persist(flags);
    set({ flags });
  },
  disableAll: () => {
    const flags = { ...DEFAULTS };
    persist(flags);
    set({ flags });
  },
}));
