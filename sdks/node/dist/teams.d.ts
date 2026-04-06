import type { ApplAdServer } from './client';
export declare class Teams {
    private client;
    constructor(client: ApplAdServer);
    create(name: string, opts?: {
        teamId?: string;
        roles?: string[];
    }): Promise<any>;
    list(): Promise<any>;
    get(teamId: string): Promise<any>;
    update(teamId: string, name: string): Promise<any>;
    delete(teamId: string): Promise<any>;
    createMembership(teamId: string, email: string, roles: string[], opts?: {
        userId?: string;
        name?: string;
    }): Promise<any>;
    listMemberships(teamId: string): Promise<any>;
    getMembership(teamId: string, membershipId: string): Promise<any>;
    updateMembership(teamId: string, membershipId: string, roles: string[]): Promise<any>;
    deleteMembership(teamId: string, membershipId: string): Promise<any>;
}
