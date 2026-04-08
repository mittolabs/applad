import type { ApplAdServer } from './client';

export class Search {
  constructor(private client: ApplAdServer) {}

  createIndex(indexId: string, opts?: { name?: string; attributes?: string[] }) {
    return this.client.call('POST', '/search/indexes', { indexId, ...opts });
  }

  listIndexes() {
    return this.client.call('GET', '/search/indexes');
  }

  getIndex(indexId: string) {
    return this.client.call('GET', `/search/indexes/${indexId}`);
  }

  deleteIndex(indexId: string) {
    return this.client.call('DELETE', `/search/indexes/${indexId}`);
  }

  indexDocument(indexId: string, documentId: string, data: Record<string, unknown>) {
    return this.client.call('POST', `/search/indexes/${indexId}/documents`, { documentId, data });
  }

  query(indexId: string, q: string, opts?: { limit?: number; offset?: number; filters?: Record<string, unknown> }) {
    return this.client.call('POST', `/search/indexes/${indexId}/search`, { query: q, ...opts });
  }

  deleteDocument(indexId: string, documentId: string) {
    return this.client.call('DELETE', `/search/indexes/${indexId}/documents/${documentId}`);
  }
}
