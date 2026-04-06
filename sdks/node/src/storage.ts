import type { ApplAdServer } from './client';

export class Storage {
  constructor(private client: ApplAdServer) {}

  // --- Buckets ---

  createBucket(
    name: string,
    opts?: {
      bucketId?: string;
      permissions?: string[];
      maximumFileSize?: number;
      allowedFileExtensions?: string[];
      compression?: string;
      encryption?: boolean;
      antivirus?: boolean;
    }
  ) {
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

  getBucket(bucketId: string) {
    return this.client.call('GET', `/storage/buckets/${bucketId}`);
  }

  updateBucket(
    bucketId: string,
    name: string,
    opts?: { permissions?: string[]; maximumFileSize?: number; enabled?: boolean }
  ) {
    return this.client.call('PUT', `/storage/buckets/${bucketId}`, {
      name,
      ...(opts?.permissions && { permissions: opts.permissions }),
      ...(opts?.maximumFileSize && { maximumFileSize: opts.maximumFileSize }),
      ...(opts?.enabled !== undefined && { enabled: opts.enabled }),
    });
  }

  deleteBucket(bucketId: string) {
    return this.client.call('DELETE', `/storage/buckets/${bucketId}`);
  }

  // --- Files ---

  createFile(
    bucketId: string,
    file: Blob,
    fileName: string,
    opts?: { fileId?: string; permissions?: string[] }
  ) {
    const formData = new FormData();
    formData.append('fileId', opts?.fileId ?? 'unique()');
    formData.append('file', file, fileName);
    if (opts?.permissions) {
      formData.append('permissions', JSON.stringify(opts.permissions));
    }
    return this.client.upload(`/storage/buckets/${bucketId}/files`, formData);
  }

  listFiles(bucketId: string, opts?: { limit?: number; offset?: number }) {
    const params = new URLSearchParams();
    if (opts?.limit) params.set('limit', String(opts.limit));
    if (opts?.offset) params.set('offset', String(opts.offset));
    const qs = params.toString();
    return this.client.call('GET', `/storage/buckets/${bucketId}/files${qs ? `?${qs}` : ''}`);
  }

  getFile(bucketId: string, fileId: string) {
    return this.client.call('GET', `/storage/buckets/${bucketId}/files/${fileId}`);
  }

  downloadFile(bucketId: string, fileId: string) {
    return this.client.download(`/storage/buckets/${bucketId}/files/${fileId}/download`);
  }

  deleteFile(bucketId: string, fileId: string) {
    return this.client.call('DELETE', `/storage/buckets/${bucketId}/files/${fileId}`);
  }
}
