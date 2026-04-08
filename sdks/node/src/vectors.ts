import type { ApplAdServer } from './client';

export class Vectors {
  constructor(private client: ApplAdServer) {}

  createIndex(indexId: string, dimensions: number, opts?: { metric?: string; name?: string }) {
    return this.client.call('POST', '/vectors/indexes', { indexId, dimensions, ...opts });
  }

  listIndexes() {
    return this.client.call('GET', '/vectors/indexes');
  }

  getIndex(indexId: string) {
    return this.client.call('GET', `/vectors/indexes/${indexId}`);
  }

  deleteIndex(indexId: string) {
    return this.client.call('DELETE', `/vectors/indexes/${indexId}`);
  }

  upsert(indexId: string, vectors: { id: string; values: number[]; metadata?: Record<string, unknown> }[]) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/vectors`, { vectors });
  }

  query(indexId: string, vector: number[], opts?: { topK?: number; filter?: Record<string, unknown> }) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/query`, { vector, ...opts });
  }

  deleteVectors(indexId: string, ids: string[]) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/delete`, { ids });
  }
}
