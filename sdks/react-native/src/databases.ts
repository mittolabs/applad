import type { ApplAdClient } from './client';

export class Databases {
  constructor(private client: ApplAdClient) {}

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

  // --- Tables ---

  createTable(
    databaseId: string,
    name: string,
    opts?: { tableId?: string; permissions?: string[]; documentSecurity?: boolean }
  ) {
    return this.client.call('POST', `/databases/${databaseId}/tables`, {
      name,
      tableId: opts?.tableId ?? 'unique()',
      permissions: opts?.permissions ?? [],
      documentSecurity: opts?.documentSecurity ?? false,
    });
  }

  listTables(databaseId: string) {
    return this.client.call('GET', `/databases/${databaseId}/tables`);
  }

  getTable(databaseId: string, tableId: string) {
    return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}`);
  }

  updateTable(
    databaseId: string,
    tableId: string,
    name: string,
    opts?: { permissions?: string[]; enabled?: boolean }
  ) {
    return this.client.call('PUT', `/databases/${databaseId}/tables/${tableId}`, {
      name,
      ...(opts?.permissions && { permissions: opts.permissions }),
      ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
    });
  }

  deleteTable(databaseId: string, tableId: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}`);
  }

  // --- Columns ---

  createStringColumn(
    databaseId: string,
    tableId: string,
    key: string,
    opts?: { required?: boolean; size?: number; defaultValue?: string; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/columns/string`,
      {
        key,
        required: opts?.required ?? false,
        array: opts?.array ?? false,
        ...(opts?.size && { size: opts.size }),
        ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
      }
    );
  }

  createIntegerColumn(
    databaseId: string,
    tableId: string,
    key: string,
    opts?: { required?: boolean; min?: number; max?: number; defaultValue?: number; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/columns/integer`,
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

  createBooleanColumn(
    databaseId: string,
    tableId: string,
    key: string,
    opts?: { required?: boolean; defaultValue?: boolean; array?: boolean }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/columns/boolean`,
      {
        key,
        required: opts?.required ?? false,
        array: opts?.array ?? false,
        ...(opts?.defaultValue !== undefined && { default: opts.defaultValue }),
      }
    );
  }

  listColumns(databaseId: string, tableId: string) {
    return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/columns`);
  }

  deleteColumn(databaseId: string, tableId: string, key: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/columns/${key}`);
  }

  // --- Indexes ---

  createIndex(
    databaseId: string,
    tableId: string,
    key: string,
    type: string,
    columns: string[],
    orders?: string[]
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/indexes`,
      { key, type, columns, ...(orders && { orders }) }
    );
  }

  listIndexes(databaseId: string, tableId: string) {
    return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/indexes`);
  }

  deleteIndex(databaseId: string, tableId: string, key: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/indexes/${key}`);
  }

  // --- Rows (Documents) ---

  createDocument(
    databaseId: string,
    tableId: string,
    data: Record<string, unknown>,
    opts?: { rowId?: string; permissions?: string[] }
  ) {
    return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/rows`, {
      rowId: opts?.rowId ?? 'unique()',
      data,
      permissions: opts?.permissions ?? [],
    });
  }

  listDocuments(
    databaseId: string,
    tableId: string,
    opts?: { limit?: number; offset?: number; queries?: string[] }
  ) {
    const params = new URLSearchParams();
    if (opts?.limit) params.set('limit', String(opts.limit));
    if (opts?.offset) params.set('offset', String(opts.offset));
    if (opts?.queries) {
      opts.queries.forEach((q) => params.append('queries[]', q));
    }
    const qs = params.toString();
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/rows${qs ? `?${qs}` : ''}`
    );
  }

  getDocument(databaseId: string, tableId: string, rowId: string) {
    return this.client.call('GET', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`);
  }

  updateDocument(
    databaseId: string,
    tableId: string,
    rowId: string,
    opts?: { data?: Record<string, unknown>; permissions?: string[] }
  ) {
    return this.client.call(
      'PATCH',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`,
      {
        ...(opts?.data && { data: opts.data }),
        ...(opts?.permissions && { permissions: opts.permissions }),
      }
    );
  }

  deleteDocument(databaseId: string, tableId: string, rowId: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`);
  }

  /** Create or update a document by ID. */
  upsertDocument(
    databaseId: string,
    tableId: string,
    rowId: string,
    data: Record<string, unknown>,
    opts?: { permissions?: string[] }
  ) {
    return this.client.call('PUT', `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`, {
      data,
      permissions: opts?.permissions ?? [],
    });
  }
}
