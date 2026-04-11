import type { ApplAdServer } from './client';
import { QueryBuilder } from './query_builder';
export type { QueryResult } from './query_builder';
export declare class Databases {
    private client;
    constructor(client: ApplAdServer);
    createDatabase(name: string, databaseId?: string): Promise<any>;
    listDatabases(): Promise<any>;
    getDatabase(databaseId: string): Promise<any>;
    updateDatabase(databaseId: string, name: string): Promise<any>;
    deleteDatabase(databaseId: string): Promise<any>;
    createTable(databaseId: string, name: string, opts?: {
        tableId?: string;
        permissions?: string[];
        documentSecurity?: boolean;
    }): Promise<any>;
    listTables(databaseId: string): Promise<any>;
    getTable(databaseId: string, tableId: string): Promise<any>;
    updateTable(databaseId: string, tableId: string, name: string, opts?: {
        permissions?: string[];
        enabled?: boolean;
    }): Promise<any>;
    deleteTable(databaseId: string, tableId: string): Promise<any>;
    createStringColumn(databaseId: string, tableId: string, key: string, opts?: {
        required?: boolean;
        size?: number;
        defaultValue?: string;
        array?: boolean;
    }): Promise<any>;
    createIntegerColumn(databaseId: string, tableId: string, key: string, opts?: {
        required?: boolean;
        min?: number;
        max?: number;
        defaultValue?: number;
        array?: boolean;
    }): Promise<any>;
    createBooleanColumn(databaseId: string, tableId: string, key: string, opts?: {
        required?: boolean;
        defaultValue?: boolean;
        array?: boolean;
    }): Promise<any>;
    listColumns(databaseId: string, tableId: string): Promise<any>;
    deleteColumn(databaseId: string, tableId: string, key: string): Promise<any>;
    getColumnPermissions(databaseId: string, tableId: string, key: string): Promise<any>;
    setColumnPermissions(databaseId: string, tableId: string, key: string, permissions: ('read' | 'write')[]): Promise<any>;
    createIndex(databaseId: string, tableId: string, key: string, type: string, columns: string[], orders?: string[]): Promise<any>;
    listIndexes(databaseId: string, tableId: string): Promise<any>;
    deleteIndex(databaseId: string, tableId: string, key: string): Promise<any>;
    createRow(databaseId: string, tableId: string, data: Record<string, unknown>, opts?: {
        rowId?: string;
        permissions?: string[];
    }): Promise<any>;
    listRows(databaseId: string, tableId: string, opts?: {
        limit?: number;
        offset?: number;
    }): Promise<any>;
    getRow(databaseId: string, tableId: string, rowId: string): Promise<any>;
    updateRow(databaseId: string, tableId: string, rowId: string, opts?: {
        data?: Record<string, unknown>;
        permissions?: string[];
    }): Promise<any>;
    deleteRow(databaseId: string, tableId: string, rowId: string): Promise<any>;
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
    from(databaseId: string, tableId: string): QueryBuilder;
}
