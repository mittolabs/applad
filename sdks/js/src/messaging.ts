import type { Applad } from './client';

export class Messaging {
  constructor(private client: Applad) {}

  sendEmail(to: string[], subject: string, html?: string) {
    return this.client.call('POST', '/messaging/email', {
      to,
      subject,
      ...(html && { html }),
    });
  }
}
