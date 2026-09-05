/*
 * The console's version, ready to display.
 *
 * Vite resolves import.meta.env at build time; the release workflow passes the
 * release tag (`v0.2.7`) and a local build gets `dev`. Normalising here rather
 * than at each call site is the point: the footer and the login page each
 * prefixed a "v" of their own, so a tag that already had one rendered
 * "vv0.2.7", and the `dev` fallback would have rendered "vdev".
 *
 * A value that starts with a digit gets the "v"; anything else — a tag that
 * brought its own, or a word like "dev" — is shown as it is.
 */
const raw = import.meta.env.VITE_APP_VERSION?.trim();

export const APP_VERSION = raw ? (/^\d/.test(raw) ? `v${raw}` : raw) : 'dev';
