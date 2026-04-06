"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Databases = void 0;
class Databases {
    constructor(client) {
        this.client = client;
    }
    // --- Databases ---
    createDatabase(name, databaseId) {
        return this.client.call('POST', '/databases', {
            name,
            databaseId: databaseId ?? 'unique()',
        });
    }
    listDatabases() {
        return this.client.call('GET', '/databases');
    }
    getDatabase(databaseId) {
        return this.client.call('GET', `/databases/${databaseId}`);
    }
    updateDatabase(databaseId, name) {
        return this.client.call('PUT', `/databases/${databaseId}`, { name });
    }
    deleteDatabase(databaseId) {
        return this.client.call('DELETE', `/databases/${databaseId}`);
    }
    // --- Collections ---
    createCollection(databaseId, name, opts) {
        return this.client.call('POST', `/databases/${databaseId}/collections`, {
            name,
            collectionId: opts?.collectionId ?? 'unique()',
            permissions: opts?.permissions ?? [],
            documentSecurity: opts?.documentSecurity ?? false,
        });
    }
    listCollections(databaseId) {
        return this.client.call('GET', `/databases/${databaseId}/collections`);
    }
    getCollection(databaseId, collectionId) {
        return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}`);
    }
    updateCollection(databaseId, collectionId, name, opts) {
        return this.client.call('PUT', `/databases/${databaseId}/collections/${collectionId}`, {
            name,
            ...(opts?.permissions && { permissions: opts.permissions }),
            ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
        });
    }
    deleteCollection(databaseId, collectionId) {
        return this.client.call('DELETE', `/databases/${databaseId}/collections/${collectionId}`);
    }
    // --- Attributes ---
    createStringAttribute(databaseId, collectionId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/collections/${collectionId}/attributes/string`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.size && { size: opts.size }),
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    createIntegerAttribute(databaseId, collectionId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/collections/${collectionId}/attributes/integer`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.min !== undefined && { min: opts.min }),
            ...(opts?.max !== undefined && { max: opts.max }),
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    createBooleanAttribute(databaseId, collectionId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/collections/${collectionId}/attributes/boolean`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    listAttributes(databaseId, collectionId) {
        return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}/attributes`);
    }
    deleteAttribute(databaseId, collectionId, key) {
        return this.client.call('DELETE', `/databases/${databaseId}/collections/${collectionId}/attributes/${key}`);
    }
    // --- Indexes ---
    createIndex(databaseId, collectionId, key, type, attributes, orders) {
        return this.client.call('POST', `/databases/${databaseId}/collections/${collectionId}/indexes`, { key, type, attributes, ...(orders && { orders }) });
    }
    listIndexes(databaseId, collectionId) {
        return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}/indexes`);
    }
    deleteIndex(databaseId, collectionId, key) {
        return this.client.call('DELETE', `/databases/${databaseId}/collections/${collectionId}/indexes/${key}`);
    }
    // --- Documents ---
    createDocument(databaseId, collectionId, data, opts) {
        return this.client.call('POST', `/databases/${databaseId}/collections/${collectionId}/documents`, {
            documentId: opts?.documentId ?? 'unique()',
            data,
            permissions: opts?.permissions ?? [],
        });
    }
    listDocuments(databaseId, collectionId, opts) {
        const params = new URLSearchParams();
        if (opts?.limit)
            params.set('limit', String(opts.limit));
        if (opts?.offset)
            params.set('offset', String(opts.offset));
        const qs = params.toString();
        return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}/documents${qs ? `?${qs}` : ''}`);
    }
    getDocument(databaseId, collectionId, documentId) {
        return this.client.call('GET', `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`);
    }
    updateDocument(databaseId, collectionId, documentId, opts) {
        return this.client.call('PATCH', `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`, {
            ...(opts?.data && { data: opts.data }),
            ...(opts?.permissions && { permissions: opts.permissions }),
        });
    }
    deleteDocument(databaseId, collectionId, documentId) {
        return this.client.call('DELETE', `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`);
    }
}
exports.Databases = Databases;
