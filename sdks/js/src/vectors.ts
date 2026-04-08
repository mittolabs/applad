import type { Applad } from './client';

export class Vectors {
  constructor(private client: Applad) {}

  /** Create a vector index. */
  createIndex(indexId: string, dimensions: number, opts?: { metric?: string; name?: string }) {
    return this.client.call('POST', '/vectors/indexes', { indexId, dimensions, ...opts });
  }

  /** List all vector indexes. */
  listIndexes() {
    return this.client.call('GET', '/vectors/indexes');
  }

  /** Get a vector index by ID. */
  getIndex(indexId: string) {
    return this.client.call('GET', `/vectors/indexes/${indexId}`);
  }

  /** Delete a vector index. */
  deleteIndex(indexId: string) {
    return this.client.call('DELETE', `/vectors/indexes/${indexId}`);
  }

  /** Upsert vectors into an index. */
  upsert(indexId: string, vectors: { id: string; values: number[]; metadata?: Record<string, unknown> }[]) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/vectors`, { vectors });
  }

  /** Query vectors by similarity. */
  query(indexId: string, vector: number[], opts?: { topK?: number; filter?: Record<string, unknown> }) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/query`, { vector, ...opts });
  }

  /** Delete vectors by IDs. */
  deleteVectors(indexId: string, ids: string[]) {
    return this.client.call('POST', `/vectors/indexes/${indexId}/delete`, { ids });
  }
}
