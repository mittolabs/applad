/*
 * Applad hostnames, derived from wherever this page is being served.
 *
 * Local development mirrors production exactly, with `.localhost` appended:
 * applad.io.localhost, console.applad.io.localhost, and so on. So rather than
 * hardcoding one and breaking the other, take the suffix from the current
 * host.
 */

const DEV_SUFFIX = '.localhost';

function suffix(): string {
  if (typeof window === 'undefined') return '';
  return window.location.hostname.endsWith(DEV_SUFFIX) ? DEV_SUFFIX : '';
}

function scheme(): string {
  if (typeof window === 'undefined') return 'https:';
  return window.location.protocol;
}

function host(sub: string): string {
  return `${scheme()}//${sub}applad.io${suffix()}`;
}

export const CONSOLE_URL = () => host('console.');
export const DOCS_URL = () => host('docs.');
export const STATUS_URL = () => host('status.');
export const API_URL = () => host('api.');
export const SITE_URL = () => host('');

/**
 * Whether someone is signed in to the console.
 *
 * The console sets a non-sensitive marker cookie on the parent domain when a
 * session starts. It holds no token — it exists purely so this site can offer
 * "Go to console" instead of "Get started". Reading the session itself is
 * impossible here by design, and a cross-origin check was not an option:
 * browsers partition storage, so an iframe probe returns nothing.
 */
export function isSignedIn(): boolean {
  if (typeof document === 'undefined') return false;
  return document.cookie.split('; ').some((c) => c.startsWith('applad_session='));
}
