import type {
  ClientErrorReport,
  ExtensionModule,
  ExtensionNavItem,
  ExtensionRegistry,
  ExtensionRoute,
} from './types';

export type {
  ClientErrorReport,
  ExtensionModule,
  ExtensionNavItem,
  ExtensionRegistry,
  ExtensionRoute,
};

/**
 * The registry for THIS build.
 *
 * A default build registers no modules. A composed build replaces this file with
 * one that imports its modules, so nothing about those modules needs to exist
 * here: not the code, not an import, not a feature flag naming them.
 */
export const registry: ExtensionRegistry = { modules: [] };

export function extensionRoutes(): ExtensionRoute[] {
  return registry.modules.flatMap((m) => m.routes ?? []);
}

export function extensionStandaloneRoutes(): ExtensionRoute[] {
  return registry.modules.flatMap((m) => m.standaloneRoutes ?? []);
}

export function extensionNav(): ExtensionNavItem[] {
  return registry.modules.flatMap((m) => m.nav ?? []);
}

export function extensionNavActions() {
  return registry.modules.flatMap((m) => m.navActions ?? []);
}

/**
 * Hands a caught render error to every registered reporter.
 *
 * Never throws: reporting a crash must not cause one. A default build has no
 * reporters and this is a no-op.
 */
export function reportClientError(report: ClientErrorReport) {
  for (const m of registry.modules) {
    try {
      m.errorReporter?.(report);
    } catch {
      // A reporter that fails is not allowed to make the original error worse.
    }
  }
}
