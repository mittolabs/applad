import type { ApplAdServer } from './client';

export class Flags {
  constructor(private client: ApplAdServer) {}

  // --- CRUD ---

  /// List all flags.
  list() {
    return this.client.call('GET', '/flags');
  }

  /// Create a new flag.
  create(key: string, name: string, opts?: { description?: string; enabled?: boolean; variants?: Record<string, unknown>; rules?: Record<string, unknown>[] }) {
    return this.client.call('POST', '/flags', {
      key,
      name,
      ...(opts?.description && { description: opts.description }),
      ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
      ...(opts?.variants && { variants: opts.variants }),
      ...(opts?.rules && { rules: opts.rules }),
    });
  }

  /// Get a flag by key.
  get(key: string) {
    return this.client.call('GET', `/flags/${key}`);
  }

  /// Update a flag.
  update(key: string, opts: { name?: string; description?: string; variants?: Record<string, unknown>; rules?: Record<string, unknown>[] }) {
    return this.client.call('PUT', `/flags/${key}`, {
      ...(opts.name && { name: opts.name }),
      ...(opts.description && { description: opts.description }),
      ...(opts.variants && { variants: opts.variants }),
      ...(opts.rules && { rules: opts.rules }),
    });
  }

  /// Delete a flag.
  delete(key: string) {
    return this.client.call('DELETE', `/flags/${key}`);
  }

  /// Toggle a flag on or off.
  toggle(key: string, enabled: boolean) {
    return this.client.call('PATCH', `/flags/${key}/toggle`, { enabled });
  }

  // --- Evaluation ---

  /// Get a single flag evaluation by key.
  getFlag(key: string) {
    return this.client.call('GET', `/flags/evaluate/${key}`);
  }

  /// Get all flag evaluations with optional context.
  getAllFlags(context?: Record<string, unknown>) {
    return this.client.call('POST', '/flags/evaluate/all', {
      ...(context && { context }),
    });
  }

  /// Evaluate a flag with key and context.
  evaluateFlag(key: string, context?: Record<string, unknown>) {
    return this.client.call('POST', '/flags/evaluate', {
      key,
      ...(context && { context }),
    });
  }
}
