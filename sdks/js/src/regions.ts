import type { Applad } from './client';

export class Regions {
  constructor(private client: Applad) {}

  /** List available regions. */
  list() {
    return this.client.call('GET', '/regions');
  }

  /** Get a specific region. */
  get(regionId: string) {
    return this.client.call('GET', `/regions/${regionId}`);
  }

  /** Get the active region for the current project. */
  getActive() {
    return this.client.call('GET', '/regions/active');
  }

  /** Set the active region for the current project. */
  setActive(regionId: string) {
    return this.client.call('PUT', '/regions/active', { regionId });
  }

  /** Get region health / latency info. */
  getHealth(regionId: string) {
    return this.client.call('GET', `/regions/${regionId}/health`);
  }
}
