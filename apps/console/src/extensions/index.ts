import type { ExtensionModule, ExtensionNavItem, ExtensionRegistry, ExtensionRoute } from './types';

export type { ExtensionModule, ExtensionNavItem, ExtensionRegistry, ExtensionRoute };

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

export function extensionNav(): ExtensionNavItem[] {
  return registry.modules.flatMap((m) => m.nav ?? []);
}

export function extensionNavActions() {
  return registry.modules.flatMap((m) => m.navActions ?? []);
}
