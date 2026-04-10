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

  // --- Templates ---

  createTemplate(
    name: string,
    type: 'email' | 'sms' | 'push',
    subject: string,
    body: string,
    opts?: { templateId?: string; variables?: string[] }
  ) {
    return this.client.call('POST', '/messaging/templates', {
      templateId: opts?.templateId ?? 'unique()',
      name,
      type,
      subject,
      body,
      variables: opts?.variables ?? [],
    });
  }

  listTemplates() {
    return this.client.call('GET', '/messaging/templates');
  }

  getTemplate(templateId: string) {
    return this.client.call('GET', `/messaging/templates/${templateId}`);
  }

  updateTemplate(
    templateId: string,
    opts: {
      name?: string;
      type?: 'email' | 'sms' | 'push';
      subject?: string;
      body?: string;
      variables?: string[];
    }
  ) {
    return this.client.call('PUT', `/messaging/templates/${templateId}`, opts);
  }

  deleteTemplate(templateId: string) {
    return this.client.call('DELETE', `/messaging/templates/${templateId}`);
  }

  sendTemplate(
    templateId: string,
    to: string[],
    variables?: Record<string, string>
  ) {
    return this.client.call('POST', `/messaging/templates/${templateId}/send`, {
      to,
      variables: variables ?? {},
    });
  }
}
