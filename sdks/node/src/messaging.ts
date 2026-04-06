import type { ApplAdServer } from './client';

export class Messaging {
  constructor(private client: ApplAdServer) {}

  sendEmail(to: string[], subject: string, opts?: { html?: string; text?: string }) {
    return this.client.call('POST', '/messaging/email', {
      to,
      subject,
      ...(opts?.html && { html: opts.html }),
      ...(opts?.text && { text: opts.text }),
    });
  }

  sendSMS(to: string[], body: string) {
    return this.client.call('POST', '/messaging/sms', { to, body });
  }

  sendPush(to: string[], title: string, body: string, opts?: { data?: Record<string, unknown> }) {
    return this.client.call('POST', '/messaging/push', {
      to,
      title,
      body,
      ...(opts?.data && { data: opts.data }),
    });
  }
}
