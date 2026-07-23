import type { ComponentType } from 'react';

/**
 * The console extension seam.
 *
 * This is a FIRST-PARTY, COMPILE-TIME seam: a build either includes modules or
 * it does not. There is no dynamic loading and no stable public plugin API.
 * A default build registers nothing and behaves as if this file did not exist.
 *
 * Note what is deliberately absent: there is no way to inject markup into a
 * core page. Banners are DATA (see `Notice`, served by /v1/entitlements) which
 * core renders in core styling. Components may only be contributed to surfaces a
 * module owns, which is what stops a composed build drifting away from the core
 * design and stops core refactors breaking whoever supplies them.
 */
export interface ExtensionRoute {
  /** Path under the project shell. */
  path: string;
  element: ComponentType;
}

export interface ExtensionNavItem {
  /** Route segment this item points at. Must match an ExtensionRoute path. */
  segment: string;
  label: string;
  /** Nav group to appear under, e.g. "settings". */
  group: string;
}

export interface ExtensionModule {
  name: string;
  /** Pages the module owns. Components are allowed here: it is the module's own surface. */
  routes?: ExtensionRoute[];
  /** Nav entries pointing at the module's own routes. */
  nav?: ExtensionNavItem[];
  /** Buttons contributed to the top nav. */
  navActions?: ComponentType[];
}

export interface ExtensionRegistry {
  modules: ExtensionModule[];
}
