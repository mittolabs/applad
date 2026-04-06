import type { Applad } from './client';

export class Avatars {
  constructor(private client: Applad) {}

  /** Get initials avatar image URL. */
  getInitials(name: string, opts?: { width?: number; height?: number; background?: string }): string {
    const params = new URLSearchParams({ name });
    if (opts?.width) params.set('width', String(opts.width));
    if (opts?.height) params.set('height', String(opts.height));
    if (opts?.background) params.set('background', opts.background);
    return `${this.client.endpoint}/v1/avatars/initials?${params}`;
  }

  /** Get QR code image URL. */
  getQR(text: string, size?: number): string {
    const params = new URLSearchParams({ text });
    if (size) params.set('size', String(size));
    return `${this.client.endpoint}/v1/avatars/qr?${params}`;
  }

  /** Get favicon URL for a domain. */
  getFavicon(url: string): string {
    return `${this.client.endpoint}/v1/avatars/favicon?url=${encodeURIComponent(url)}`;
  }

  /** Get credit card brand icon URL. */
  getCreditCard(code: string): string {
    return `${this.client.endpoint}/v1/avatars/credit-cards/${code}`;
  }

  /** Get country flag URL. */
  getFlag(code: string): string {
    return `${this.client.endpoint}/v1/avatars/flags/${code}`;
  }
}
