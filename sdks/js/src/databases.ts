import type { Applad } from './client';

export class Databases {
  constructor(private client: Applad) {}

  // --- Databases ---

  createDatabase(name: string, databaseId?: string) {
    return this.client.call('POST', '/databases', {
      name,
      databaseId: databaseId ?? 'unique()',
    });
  }

  listDatabases() {
    return this.client.call('GET', '/databases');
  }

  getDatabase(databaseId: string) {
    return this.client.call('GET', `/databases/${databaseId}`);
  }

  updateDatabase(databaseId: string, name: string) {
    return this.client.call('PUT', `/databases/${databaseId}`, { name });
  }

  deleteDatabase(databaseId: string) {
    return this.client.call('DELETE', `/databases/${databaseId}`);
  }

  // --- Collections ---

  createCollection(
    databaseId: string,
    name: string,
    opts?: { collectionId?: string; permissions?: string[]; documentSecurity?: boolean }
  ) {
    return this.client.call('POST', `/databases/${databaseId}/collections`, {
      name,
      collectionId: opts?.collectionId ?? 'unique()',
      permissions: opts?.permissions ?? [],
      documentSecurity: opts?.documentSecurity ?? false,
    });
  }

  listCollections(databaseId: string) {
    return this.client.call('GET', `/databases/${databaseId}/collections`);
  }

  getCollection(databaseId: string, collectionId: string) {
    return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}`);
  }

  updateCollection(
    databaseId: string,
    collectionId: string,
    name: string,
    opts?: { permissions?: string[]; enabled?: boolean }
  ) {
    return this.client.call('PUT', `/databases/${databaseId}/collections/${collectionId}`, {
      name,
      ...(opts?.permissions && { permissions: opts.permissions }),
      ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
    });
  }

  deleteCollection(databaseId: string, collectionId: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/collections/${collectionId}`);
  }

  // --- Attributes ---

  createStringAttribute(
    databaseId: string,
    collectionId: string,
    key: string,
    opts?: { required?: boolean; size?: number; defaultValue?: string; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/collections/${collectionId}/attributes/string`,
      {
        key,
        required: opts?.required ?? false,
        array: opts?.array ?? false,
        ...(opts?.size && { size: opts.size }),
        ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
      }
    );
  }

  createIntegerAttribute(
    databaseId: string,
    collectionId: string,
    key: string,
    opts?: { required?: boolean; min?: number; max?: number; defaultValue?: number; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/collections/${collectionId}/attributes/integer`,
      {
        key,
        required: opts?.required ?? false,
        array: opts?.array ?? false,
        ...(opts?.min !== undefined && { min: opts.min }),
        ...(opts?.max !== undefined && { max: opts.max }),
        ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
      }
    );
  }

  createBooleanAttribute(
    databaseId: string,
    collectionId: string,
    key: string,
    opts?: { required?: boolean; defaultValue?: boolean; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/collections/${collectionId}/attributes/boolean`,
      {
        key,
        required: opts?.required ?? false,
        array: opts?.array ?? false,
        ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
      }
    );
  }

  listAttributes(databaseId: string, collectionId: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/collections/${collectionId}/attributes`
    );
  }

  deleteAttribute(databaseId: string, collectionId: string, key: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/collections/${collectionId}/attributes/${key}`
    );
  }

  // --- Indexes ---

  createIndex(
    databaseId: string,
    collectionId: string,
    key: string,
    type: string,
    attributes: string[],
    orders?: string[]
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/collections/${collectionId}/indexes`,
      { key, type, attributes, ...(orders && { orders }) }
    );
  }

  listIndexes(databaseId: string, collectionId: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/collections/${collectionId}/indexes`
    );
  }

  deleteIndex(databaseId: string, collectionId: string, key: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/collections/${collectionId}/indexes/${key}`
    );
  }

  // --- Documents ---

  createDocument(
    databaseId: string,
    collectionId: string,
    data: Record<string, unknown>,
    opts?: { documentId?: string; permissions?: string[] }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/collections/${collectionId}/documents`,
      {
        documentId: opts?.documentId ?? 'unique()',
        data,
        permissions: opts?.permissions ?? [],
      }
    );
  }

  listDocuments(
    databaseId: string,
    collectionId: string,
    opts?: { limit?: number; offset?: number }
  ) {
    const params = new URLSearchParams();
    if (opts?.limit) params.set('limit', String(opts.limit));
    if (opts?.offset) params.set('offset', String(opts.offset));
    const qs = params.toString();
    return this.client.call(
      'GET',
      `/databases/${databaseId}/collections/${collectionId}/documents${qs ? `?${qs}` : ''}`
    );
  }

  getDocument(databaseId: string, collectionId: string, documentId: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`
    );
  }

  updateDocument(
    databaseId: string,
    collectionId: string,
    documentId: string,
    opts?: { data?: Record<string, unknown>; permissions?: string[] }
  ) {
    return this.client.call(
      'PATCH',
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`,
      {
        ...(opts?.data && { data: opts.data }),
        ...(opts?.permissions && { permissions: opts.permissions }),
      }
    );
  }

  deleteDocument(databaseId: string, collectionId: string, documentId: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`
    );
  }
}
