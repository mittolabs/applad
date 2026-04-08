import type { ApplAdServer } from './client';

export class Regions {
  constructor(private client: ApplAdServer) {}

  list() {
    return this.client.call('GET', '/regions');
  }

  get(regionId: string) {
    return this.client.call('GET', `/regions/${regionId}`);
  }

  getActive() {
    return this.client.call('GET', '/regions/active');
  }

  setActive(regionId: string) {
    return this.client.call('PUT', '/regions/active', { regionId });
  }

  getHealth(regionId: string) {
    return this.client.call('GET', `/regions/${regionId}/health`);
  }
}
