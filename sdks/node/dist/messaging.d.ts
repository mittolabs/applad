import type { ApplAdServer } from './client';
export declare class Messaging {
    private client;
    constructor(client: ApplAdServer);
    sendEmail(to: string[], subject: string, opts?: {
        html?: string;
        text?: string;
    }): Promise<any>;
    sendSMS(to: string[], body: string): Promise<any>;
    sendPush(to: string[], title: string, body: string, opts?: {
        data?: Record<string, unknown>;
    }): Promise<any>;
    createTemplate(name: string, type: 'email' | 'sms' | 'push', subject: string, body: string, opts?: {
        templateId?: string;
        variables?: string[];
    }): Promise<any>;
    listTemplates(): Promise<any>;
    getTemplate(templateId: string): Promise<any>;
    updateTemplate(templateId: string, opts: {
        name?: string;
        type?: 'email' | 'sms' | 'push';
        subject?: string;
        body?: string;
        variables?: string[];
    }): Promise<any>;
    deleteTemplate(templateId: string): Promise<any>;
    sendTemplate(templateId: string, to: string[], variables?: Record<string, string>): Promise<any>;
}
