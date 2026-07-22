import { create } from 'zustand';

/**
 * Theme store — mirrors console/lib/core/providers/theme_provider.dart.
 * Persists to localStorage['applad_theme'] ("light" | "dark" | "system").
 * Flutter stored a bool (true=light); we widen to include "system" to match
 * the UserMenu's Light/Dark/System toggle in navbar_popovers.dart.
 */
export type ThemeMode = 'light' | 'dark' | 'system';

const KEY = 'applad_theme';

function readInitial(): ThemeMode {
  const raw = localStorage.getItem(KEY);
  if (raw === 'light' || raw === 'dark' || raw === 'system') return raw;
  return 'dark';
}

function systemPrefersLight(): boolean {
  return window.matchMedia('(prefers-color-scheme: light)').matches;
}

/** Resolve a mode to the concrete applied theme. */
export function resolveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') return systemPrefersLight() ? 'light' : 'dark';
  return mode;
}

export function applyTheme(mode: ThemeMode) {
  document.documentElement.setAttribute('data-theme', resolveTheme(mode));
}

interface ThemeState {
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  toggle: () => void;
}

/** Monaco editor theme matching the current app theme. */
export function useMonacoTheme(): 'vs-dark' | 'light' {
  const mode = useThemeStore((s) => s.mode);
  return resolveTheme(mode) === 'light' ? 'light' : 'vs-dark';
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: readInitial(),
  setMode: (mode) => {
    localStorage.setItem(KEY, mode);
    applyTheme(mode);
    set({ mode });
  },
  toggle: () => {
    const next = resolveTheme(get().mode) === 'light' ? 'dark' : 'light';
    get().setMode(next);
  },
}));
