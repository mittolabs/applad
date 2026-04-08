import type { Applad } from './client';

export class Search {
  constructor(private client: Applad) {}

  /** Create a search index. */
  createIndex(indexId: string, opts?: { name?: string; attributes?: string[] }) {
    return this.client.call('POST', '/search/indexes', { indexId, ...opts });
  }

  /** List all search indexes. */
  listIndexes() {
    return this.client.call('GET', '/search/indexes');
  }

  /** Get a search index by ID. */
  getIndex(indexId: string) {
    return this.client.call('GET', `/search/indexes/${indexId}`);
  }

  /** Delete a search index. */
  deleteIndex(indexId: string) {
    return this.client.call('DELETE', `/search/indexes/${indexId}`);
  }

  /** Index a document into a search index. */
  indexDocument(indexId: string, documentId: string, data: Record<string, unknown>) {
    return this.client.call('POST', `/search/indexes/${indexId}/documents`, { documentId, data });
  }

  /** Search documents in an index. */
  query(indexId: string, q: string, opts?: { limit?: number; offset?: number; filters?: Record<string, unknown> }) {
    return this.client.call('POST', `/search/indexes/${indexId}/search`, { query: q, ...opts });
  }

  /** Delete a document from a search index. */
  deleteDocument(indexId: string, documentId: string) {
    return this.client.call('DELETE', `/search/indexes/${indexId}/documents/${documentId}`);
  }
}
