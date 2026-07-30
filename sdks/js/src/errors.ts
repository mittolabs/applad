/**
 * Error thrown when the API responds with a non-2xx status.
 *
 * Carries the HTTP `status` and, when the server sent a JSON error body,
 * its `message` and `type` (the machine-readable error code) so callers can
 * branch on the real reason rather than parsing a string.
 */
export class AppladError extends Error {
  readonly status: number;
  /** Machine-readable error code from the server body (`type` field). */
  readonly type?: string;

  constructor(status: number, message: string, type?: string) {
    super(message);
    this.name = 'AppladError';
    this.status = status;
    this.type = type;
  }
}

/**
 * Build an {@link AppladError} from a failed `fetch` Response, reading the
 * server's `{message, type}` JSON body when present and falling back to a
 * `METHOD path -> status` summary otherwise.
 */
export async function errorFromResponse(
  res: Response,
  method: string,
  path: string
): Promise<AppladError> {
  let body: any;
  try {
    body = await res.json();
  } catch {
    body = undefined;
  }
  const message =
    body && typeof body.message === 'string' && body.message
      ? body.message
      : `${method} ${path} -> ${res.status}`;
  const type = body && typeof body.type === 'string' ? body.type : undefined;
  return new AppladError(res.status, message, type);
}
