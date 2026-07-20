import type { ApplAdServer } from './client';
import { QueryBuilder } from './query_builder';
export type { QueryResult } from './query_builder';

export class Databases {
  constructor(private client: ApplAdServer) {}

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
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/columns`
    );
  }

  deleteColumn(databaseId: string, tableId: string, key: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/tables/${tableId}/columns/${key}`
    );
  }

  getColumnPermissions(databaseId: string, tableId: string, key: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/columns/${key}/permissions`
    );
  }

  setColumnPermissions(databaseId: string, tableId: string, key: string, permissions: ('read' | 'write')[]) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/columns/${key}/permissions`,
      { permissions }
    );
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
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/indexes`
    );
  }

  deleteIndex(databaseId: string, tableId: string, key: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/tables/${tableId}/indexes/${key}`
    );
  }

  // --- Rows ---

  createRow(
    databaseId: string,
    tableId: string,
    data: Record<string, unknown>,
    opts?: { rowId?: string; permissions?: string[] }
  ) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/rows`,
      {
        rowId: opts?.rowId ?? 'unique()',
        data,
        permissions: opts?.permissions ?? [],
      }
    );
  }

  listRows(
    databaseId: string,
    tableId: string,
    opts?: { limit?: number; offset?: number; status?: 'draft' | 'published'; locale?: string }
  ) {
    const params = new URLSearchParams();
    if (opts?.limit) params.set('limit', String(opts.limit));
    if (opts?.offset) params.set('offset', String(opts.offset));
    // Content mode: narrow to a publish state and/or a locale.
    if (opts?.status) params.set('status', opts.status);
    if (opts?.locale) params.set('locale', opts.locale);
    const qs = params.toString();
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/rows${qs ? `?${qs}` : ''}`
    );
  }

  getRow(databaseId: string, tableId: string, rowId: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`
    );
  }

  updateRow(
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

  deleteRow(databaseId: string, tableId: string, rowId: string) {
    return this.client.call(
      'DELETE',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}`
    );
  }

  // --- Content mode ---
  // A table can act as an editorial collection: rows gain a draft/published
  // workflow, a slug, a locale and version history. Same table, same rows API.

  /** Turn a table into an editorial collection. */
  enableContentMode(databaseId: string, tableId: string) {
    return this.client.call('POST', `/databases/${databaseId}/tables/${tableId}/content`);
  }

  /** Hide the editorial tools again. Nothing is deleted. */
  disableContentMode(databaseId: string, tableId: string) {
    return this.client.call('DELETE', `/databases/${databaseId}/tables/${tableId}/content`);
  }

  /** Publish an entry. */
  publishRow(databaseId: string, tableId: string, rowId: string) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}/publish`
    );
  }

  /** Return an entry to draft. */
  unpublishRow(databaseId: string, tableId: string, rowId: string) {
    return this.client.call(
      'POST',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}/unpublish`
    );
  }

  /** Version snapshots for an entry, newest first. */
  listRowVersions(databaseId: string, tableId: string, rowId: string) {
    return this.client.call(
      'GET',
      `/databases/${databaseId}/tables/${tableId}/rows/${rowId}/versions`
    );
  }

  // --- Query builder ---

  /**
   * Returns a fluent {@link QueryBuilder} for the given table.
   *
   * @example
   * ```ts
   * const result = await server.databases
   *   .from('myDb', 'orders')
   *   .equal('status', 'pending')
   *   .orderAsc('created_at')
   *   .limit(50)
   *   .get();
   * ```
   */
  from(databaseId: string, tableId: string): QueryBuilder {
    return new QueryBuilder(async (params) => {
      const qs = new URLSearchParams();
      for (const [k, v] of Object.entries(params)) {
        if (Array.isArray(v)) {
          v.forEach((item) => qs.append(k, String(item)));
        } else if (v !== undefined) {
          qs.set(k, String(v));
        }
      }
      const query = qs.toString();
      return this.client.call(
        'GET',
        `/databases/${databaseId}/tables/${tableId}/rows${query ? `?${query}` : ''}`
      );
    });
  }
}
