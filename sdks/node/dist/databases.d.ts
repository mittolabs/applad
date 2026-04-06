import type { ApplAdServer } from './client';
export declare class Databases {
    private client;
    constructor(client: ApplAdServer);
    createDatabase(name: string, databaseId?: string): Promise<any>;
    listDatabases(): Promise<any>;
    getDatabase(databaseId: string): Promise<any>;
    updateDatabase(databaseId: string, name: string): Promise<any>;
    deleteDatabase(databaseId: string): Promise<any>;
    createCollection(databaseId: string, name: string, opts?: {
        collectionId?: string;
        permissions?: string[];
        documentSecurity?: boolean;
    }): Promise<any>;
    listCollections(databaseId: string): Promise<any>;
    getCollection(databaseId: string, collectionId: string): Promise<any>;
    updateCollection(databaseId: string, collectionId: string, name: string, opts?: {
        permissions?: string[];
        enabled?: boolean;
    }): Promise<any>;
    deleteCollection(databaseId: string, collectionId: string): Promise<any>;
    createStringAttribute(databaseId: string, collectionId: string, key: string, opts?: {
        required?: boolean;
        size?: number;
        defaultValue?: string;
        array?: boolean;
    }): Promise<any>;
    createIntegerAttribute(databaseId: string, collectionId: string, key: string, opts?: {
        required?: boolean;
        min?: number;
        max?: number;
        defaultValue?: number;
        array?: boolean;
    }): Promise<any>;
    createBooleanAttribute(databaseId: string, collectionId: string, key: string, opts?: {
        required?: boolean;
        defaultValue?: boolean;
        array?: boolean;
    }): Promise<any>;
    listAttributes(databaseId: string, collectionId: string): Promise<any>;
    deleteAttribute(databaseId: string, collectionId: string, key: string): Promise<any>;
    createIndex(databaseId: string, collectionId: string, key: string, type: string, attributes: string[], orders?: string[]): Promise<any>;
    listIndexes(databaseId: string, collectionId: string): Promise<any>;
    deleteIndex(databaseId: string, collectionId: string, key: string): Promise<any>;
    createDocument(databaseId: string, collectionId: string, data: Record<string, unknown>, opts?: {
        documentId?: string;
        permissions?: string[];
    }): Promise<any>;
    listDocuments(databaseId: string, collectionId: string, opts?: {
        limit?: number;
        offset?: number;
    }): Promise<any>;
    getDocument(databaseId: string, collectionId: string, documentId: string): Promise<any>;
    updateDocument(databaseId: string, collectionId: string, documentId: string, opts?: {
        data?: Record<string, unknown>;
        permissions?: string[];
    }): Promise<any>;
    deleteDocument(databaseId: string, collectionId: string, documentId: string): Promise<any>;
}
