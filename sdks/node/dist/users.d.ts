import type { ApplAdServer } from './client';
export declare class Users {
    private client;
    constructor(client: ApplAdServer);
    createUser(email: string, password: string, opts?: {
        userId?: string;
        name?: string;
    }): Promise<any>;
    listUsers(opts?: {
        limit?: number;
        offset?: number;
        search?: string;
    }): Promise<any>;
    getUser(userId: string): Promise<any>;
    deleteUser(userId: string): Promise<any>;
    updateUserName(userId: string, name: string): Promise<any>;
    updateUserEmail(userId: string, email: string): Promise<any>;
    updateUserStatus(userId: string, status: boolean): Promise<any>;
}
