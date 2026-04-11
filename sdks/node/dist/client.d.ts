import { Analytics } from './analytics';
import { Billing } from './billing';
import { Users } from './users';
import { Databases } from './databases';
import { Storage } from './storage';
import { Functions } from './functions';
import { Teams } from './teams';
import { Workflows } from './workflows';
import { Messaging } from './messaging';
import { Deploy } from './deploy';
import { Edge } from './edge';
import { Flags } from './flags';
import { Regions } from './regions';
import { Search } from './search';
import { Vectors } from './vectors';
import { Observe } from './observe';
export interface ApplAdServerConfig {
    endpoint: string;
    projectId: string;
    apiKey: string;
}
export declare class ApplAdServer {
    readonly endpoint: string;
    readonly projectId: string;
    private headers;
    readonly analytics: Analytics;
    readonly billing: Billing;
    readonly users: Users;
    readonly databases: Databases;
    readonly storage: Storage;
    readonly functions: Functions;
    readonly teams: Teams;
    readonly workflows: Workflows;
    readonly messaging: Messaging;
    readonly deploy: Deploy;
    readonly edge: Edge;
    readonly flags: Flags;
    readonly regions: Regions;
    readonly search: Search;
    readonly vectors: Vectors;
    readonly observe: Observe;
    constructor(config: ApplAdServerConfig);
    call<T = any>(method: string, path: string, body?: unknown): Promise<T>;
    upload<T = any>(path: string, formData: FormData): Promise<T>;
    download(path: string): Promise<ArrayBuffer>;
}
