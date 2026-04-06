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
}
exports.Messaging = Messaging;
