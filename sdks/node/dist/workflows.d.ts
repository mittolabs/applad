import type { ApplAdServer } from './client';
export declare class Workflows {
    private client;
    constructor(client: ApplAdServer);
    create(name: string, opts?: {
        description?: string;
        triggerType?: string;
        triggerConfig?: Record<string, unknown>;
        nodes?: unknown[];
        edges?: unknown[];
    }): Promise<any>;
    list(): Promise<any>;
    get(workflowId: string): Promise<any>;
    update(workflowId: string, data: Record<string, unknown>): Promise<any>;
    delete(workflowId: string): Promise<any>;
    execute(workflowId: string, triggerData?: Record<string, unknown>): Promise<any>;
    listExecutions(workflowId: string): Promise<any>;
    getExecution(workflowId: string, executionId: string): Promise<any>;
}
