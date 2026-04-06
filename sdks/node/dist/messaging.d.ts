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
}
