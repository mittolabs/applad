"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Messaging = void 0;
class Messaging {
    constructor(client) {
        this.client = client;
    }
    sendEmail(to, subject, opts) {
        return this.client.call('POST', '/messaging/email', {
            to,
            subject,
            ...(opts?.html && { html: opts.html }),
            ...(opts?.text && { text: opts.text }),
        });
    }
    sendSMS(to, body) {
        return this.client.call('POST', '/messaging/sms', { to, body });
    }
    sendPush(to, title, body, opts) {
        return this.client.call('POST', '/messaging/push', {
            to,
            title,
            body,
            ...(opts?.data && { data: opts.data }),
        });
    }
    // --- Templates ---
    createTemplate(name, type, subject, body, opts) {
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
    getTemplate(templateId) {
        return this.client.call('GET', `/messaging/templates/${templateId}`);
    }
    updateTemplate(templateId, opts) {
        return this.client.call('PUT', `/messaging/templates/${templateId}`, opts);
    }
    deleteTemplate(templateId) {
        return this.client.call('DELETE', `/messaging/templates/${templateId}`);
    }
    sendTemplate(templateId, to, variables) {
        return this.client.call('POST', `/messaging/templates/${templateId}/send`, {
            to,
            variables: variables ?? {},
        });
    }
}
exports.Messaging = Messaging;
