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

/**
 * A render error the console caught, offered to whoever wants to record it.
 *
 * Core catches the error and shows the user something human; it does not decide
 * where a report goes. A default build reports nowhere (a self-hosted console
 * has no vendor to tell), and the cloud build registers a reporter that posts it
 * so an operator can see it in the backoffice.
 */
export interface ClientErrorReport {
  message: string;
  stack?: string;
  componentStack?: string;
  path: string;
  userAgent: string;
}

export interface ExtensionModule {
  name: string;
  /** Pages the module owns, under the project shell (path relative to /project/:projectId). */
  routes?: ExtensionRoute[];
  /**
   * Pages the module owns that are NOT project-scoped: authed, but sitting beside
   * /account and /projects rather than inside a project. Billing is org-scoped
   * (a subscription belongs to an organization, not a project), so it lives here.
   * `path` is absolute from the app root, e.g. "/settings/billing".
   */
  standaloneRoutes?: ExtensionRoute[];
  /** Nav entries pointing at the module's own routes. */
  nav?: ExtensionNavItem[];
  /** Buttons contributed to the top nav. */
  navActions?: ComponentType[];
  /** Records a render error core caught. Must not throw. */
  errorReporter?: (report: ClientErrorReport) => void;
}

export interface ExtensionRegistry {
  modules: ExtensionModule[];
}
