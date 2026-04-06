import type { ApplAdServer } from './client';
export declare class Functions {
    private client;
    constructor(client: ApplAdServer);
    create(name: string, runtime: string, opts?: {
        entrypoint?: string;
        timeout?: number;
        vars?: Record<string, string>;
        source?: string;
    }): Promise<any>;
    list(): Promise<any>;
    get(functionId: string): Promise<any>;
    update(functionId: string, data: Record<string, unknown>): Promise<any>;
    delete(functionId: string): Promise<any>;
    execute(functionId: string, data?: Record<string, unknown>): Promise<any>;
    listExecutions(functionId: string): Promise<any>;
    getExecution(functionId: string, executionId: string): Promise<any>;
}
