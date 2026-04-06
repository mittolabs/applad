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
    // --- Tables ---
    createTable(databaseId, name, opts) {
        return this.client.call('POST', `/databases/${databaseId}/tables`, {
            name,
            tableId: opts?.tableId ?? 'unique()',
            permissions: opts?.permissions ?? [],
            documentSecurity: opts?.documentSecurity ?? false,
        });
    }
    listTables(databaseId) {
        return this.client.call('GET', `/databases/${databaseId}/tables`);
    }
    getTable(databaseId, tableId) {
        return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}`);
    }
    updateTable(databaseId, tableId, name, opts) {
        return this.client.call('PUT', `/databases/${databaseId}/tables/${tableId}`, {
            name,
            ...(opts?.permissions && { permissions: opts.permissions }),
            ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
        });
    }
    deleteTable(databaseId, tableId) {
        return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}`);
    }
    // --- Columns ---
    createStringColumn(databaseId, tableId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/columns/string`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.size && { size: opts.size }),
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    createIntegerColumn(databaseId, tableId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/columns/integer`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.min !== undefined && { min: opts.min }),
            ...(opts?.max !== undefined && { max: opts.max }),
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    createBooleanColumn(databaseId, tableId, key, opts) {
        return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/columns/boolean`, {
            key,
            required: opts?.required ?? false,
            array: opts?.array ?? false,
            ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
        });
    }
    listColumns(databaseId, tableId) {
        return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/columns`);
    }
    deleteColumn(databaseId, tableId, key) {
        return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/columns/${key}`);
    }
    // --- Indexes ---
    createIndex(databaseId, tableId, key, type, columns, orders) {
        return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/indexes`, { key, type, columns, ...(orders && { orders }) });
    }
    listIndexes(databaseId, tableId) {
        return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/indexes`);
    }
    deleteIndex(databaseId, tableId, key) {
        return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/indexes/${key}`);
    }
    // --- Rows ---
    createRow(databaseId, tableId, data, opts) {
        return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/rows`, {
            rowId: opts?.rowId ?? 'unique()',
            data,
            permissions: opts?.permissions ?? [],
        });
    }
    listRows(databaseId, tableId, opts) {
        const params = new URLSearchParams();
        if (opts?.limit)
            params.set('limit', String(opts.limit));
        if (opts?.offset)
            params.set('offset', String(opts.offset));
        const qs = params.toString();
        return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/rows${qs ? `?${qs}` : ''}`);
    }
    getRow(databaseId, tableId, rowId) {
        return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`);
    }
    updateRow(databaseId, tableId, rowId, opts) {
        return this.client.call('PATCH', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`, {
            ...(opts?.data && { data: opts.data }),
            ...(opts?.permissions && { permissions: opts.permissions }),
        });
    }
    deleteRow(databaseId, tableId, rowId) {
        return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`);
    }
}
exports.Databases = Databases;
