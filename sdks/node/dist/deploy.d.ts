import type { ApplAdServer } from './client';
export declare class Deploy {
    private client;
    constructor(client: ApplAdServer);
    create(name: string, type: string, config?: Record<string, unknown>): Promise<any>;
    list(): Promise<any>;
    get(deploymentId: string): Promise<any>;
    update(deploymentId: string, data: Record<string, unknown>): Promise<any>;
    updateStatus(deploymentId: string, status: string): Promise<any>;
    delete(deploymentId: string): Promise<any>;
}
