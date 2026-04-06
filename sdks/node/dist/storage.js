"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Storage = void 0;
class Storage {
    constructor(client) {
        this.client = client;
    }
    // --- Buckets ---
    createBucket(name, opts) {
        return this.client.call('POST', '/storage/buckets', {
            name,
            bucketId: opts?.bucketId ?? 'unique()',
            permissions: opts?.permissions ?? [],
            allowedFileExtensions: opts?.allowedFileExtensions ?? [],
            ...(opts?.maximumFileSize && { maximumFileSize: opts.maximumFileSize }),
            ...(opts?.compression && { compression: opts.compression }),
            encryption: opts?.encryption ?? false,
            antivirus: opts?.antivirus ?? false,
        });
    }
    listBuckets() {
        return this.client.call('GET', '/storage/buckets');
    }
    getBucket(bucketId) {
        return this.client.call('GET', `/storage/buckets/${bucketId}`);
    }
    updateBucket(bucketId, name, opts) {
        return this.client.call('PUT', `/storage/buckets/${bucketId}`, {
            name,
            ...(opts?.permissions && { permissions: opts.permissions }),
            ...(opts?.maximumFileSize && { maximumFileSize: opts.maximumFileSize }),
            ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
        });
    }
    deleteBucket(bucketId) {
        return this.client.call('DELETE', `/storage/buckets/${bucketId}`);
    }
    // --- Files ---
    createFile(bucketId, file, fileName, opts) {
        const formData = new FormData();
        formData.append('fileId', opts?.fileId ?? 'unique()');
        formData.append('file', file, fileName);
        if (opts?.permissions) {
            formData.append('permissions', JSON.stringify(opts.permissions));
        }
        return this.client.upload(`/storage/buckets/${bucketId}/files`, formData);
    }
    listFiles(bucketId, opts) {
        const params = new URLSearchParams();
        if (opts?.limit)
            params.set('limit', String(opts.limit));
        if (opts?.offset)
            params.set('offset', String(opts.offset));
        const qs = params.toString();
        return this.client.call('GET', `/storage/buckets/${bucketId}/files${qs ? `?${qs}` : ''}`);
    }
    getFile(bucketId, fileId) {
        return this.client.call('GET', `/storage/buckets/${bucketId}/files/${fileId}`);
    }
    downloadFile(bucketId, fileId) {
        return this.client.download(`/storage/buckets/${bucketId}/files/${fileId}/download`);
    }
    deleteFile(bucketId, fileId) {
        return this.client.call('DELETE', `/storage/buckets/${bucketId}/files/${fileId}`);
    }
}
exports.Storage = Storage;
