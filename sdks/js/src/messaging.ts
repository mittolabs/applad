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
