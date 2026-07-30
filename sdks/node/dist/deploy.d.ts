import type { ApplAdServer } from './client';
export declare class Deploy {
    private client;
    constructor(client: ApplAdServer);
    create(name: string, type: string, config?: Record<string, unknown>): Promise<any>;
    list(): Promise<any>;
    get(targetId: string): Promise<any>;
    update(targetId: string, data: Record<string, unknown>): Promise<any>;
    delete(targetId: string): Promise<any>;
    deploy(targetId: string, options?: {
        request?: string;
        trigger?: string;
    }): Promise<any>;
}
