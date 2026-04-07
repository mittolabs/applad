import type { ApplAdClient } from './client';

export class Storage {
  constructor(private client: ApplAdClient) {}

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

  /**
   * Upload a file. Accepts React Native compatible file descriptors.
   *
   * @param bucketId - Target bucket ID
   * @param fileUri - Local file URI (e.g. from image picker: "file:///path/to/image.jpg")
   * @param fileName - The file name to use
   * @param mimeType - MIME type (e.g. "image/jpeg")
   * @param opts - Optional file ID and permissions
   */
  uploadFile(
    bucketId: string,
    fileUri: string,
    fileName: string,
    mimeType: string,
    opts?: { fileId?: string; permissions?: string[] }
  ) {
    const formData = new FormData();
    formData.append('fileId', opts?.fileId ?? 'unique()');
    // React Native FormData accepts { uri, name, type } objects
    formData.append('file', {
      uri: fileUri,
      name: fileName,
      type: mimeType,
    } as any);
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

  deleteFile(bucketId: string, fileId: string) {
    return this.client.call('DELETE', `/storage/buckets/${bucketId}/files/${fileId}`);
  }

  downloadFile(bucketId: string, fileId: string) {
    return this.client.download(`/storage/buckets/${bucketId}/files/${fileId}/download`);
  }

  /**
   * Build a URL for file preview/thumbnail with optional transforms.
   */
  getFilePreview(
    bucketId: string,
    fileId: string,
    opts?: { width?: number; height?: number; quality?: number; format?: string }
  ): string {
    const params = new URLSearchParams();
    params.set('project', this.client.projectId);
    if (opts?.width) params.set('width', String(opts.width));
    if (opts?.height) params.set('height', String(opts.height));
    if (opts?.quality) params.set('quality', String(opts.quality));
    if (opts?.format) params.set('output', opts.format);
    return `${this.client.endpoint}/v1/storage/buckets/${bucketId}/files/${fileId}/preview?${params.toString()}`;
  }
}
