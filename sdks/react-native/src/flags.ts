import type { ApplAdClient } from './client';

export class Flags {
  constructor(private client: ApplAdClient) {}

  /** Get a single flag evaluation by key. */
  getFlag(key: string) {
    return this.client.call('GET', `/flags/evaluate/${key}`);
  }

  /** Get all flag evaluations with optional context. */
  getAllFlags(context?: Record<string, unknown>) {
    return this.client.call('POST', '/flags/evaluate/all', {
      ...(context && { context }),
    });
  }

  /** Evaluate a flag with key and context. */
  evaluateFlag(key: string, context?: Record<string, unknown>) {
    return this.client.call('POST', '/flags/evaluate', {
      key,
      ...(context && { context }),
    });
  }
}
