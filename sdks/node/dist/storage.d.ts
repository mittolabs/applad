import type { ApplAdServer } from './client';
export declare class Storage {
    private client;
    constructor(client: ApplAdServer);
    createBucket(name: string, opts?: {
        bucketId?: string;
        permissions?: string[];
        maximumFileSize?: number;
        allowedFileExtensions?: string[];
        compression?: string;
        encryption?: boolean;
        antivirus?: boolean;
    }): Promise<any>;
    listBuckets(): Promise<any>;
    getBucket(bucketId: string): Promise<any>;
    updateBucket(bucketId: string, name: string, opts?: {
        permissions?: string[];
        maximumFileSize?: number;
        enabled?: boolean;
    }): Promise<any>;
    deleteBucket(bucketId: string): Promise<any>;
    createFile(bucketId: string, file: Blob, fileName: string, opts?: {
        fileId?: string;
        permissions?: string[];
    }): Promise<any>;
    listFiles(bucketId: string, opts?: {
        limit?: number;
        offset?: number;
    }): Promise<any>;
    getFile(bucketId: string, fileId: string): Promise<any>;
    downloadFile(bucketId: string, fileId: string): Promise<ArrayBuffer>;
    deleteFile(bucketId: string, fileId: string): Promise<any>;
}
