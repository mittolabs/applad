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

export interface ApplAdServerConfig {
  endpoint: string;
  projectId: string;
  apiKey: string;
}

export class ApplAdServer {
  readonly endpoint: string;
  readonly projectId: string;
  private headers: Record<string, string>;

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

  constructor(config: ApplAdServerConfig) {
    this.endpoint = config.endpoint.replace(/\/$/, '');
    this.projectId = config.projectId;
    this.headers = {
      'X-Applad-Project': config.projectId,
      'X-Applad-Key': config.apiKey,
      'Content-Type': 'application/json',
    };
    this.analytics = new Analytics(this);
    this.billing = new Billing(this);
    this.users = new Users(this);
    this.databases = new Databases(this);
    this.storage = new Storage(this);
    this.functions = new Functions(this);
    this.teams = new Teams(this);
    this.workflows = new Workflows(this);
    this.messaging = new Messaging(this);
    this.deploy = new Deploy(this);
    this.edge = new Edge(this);
    this.flags = new Flags(this);
    this.regions = new Regions(this);
    this.search = new Search(this);
    this.vectors = new Vectors(this);
  }

  async call<T = any>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method,
      headers: this.headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}`);
    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  async upload<T = any>(path: string, formData: FormData): Promise<T> {
    const headers = { ...this.headers };
    delete headers['Content-Type'];
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method: 'POST',
      headers,
      body: formData,
    });
    if (!res.ok) throw new Error(`POST ${path} -> ${res.status}`);
    return res.json() as Promise<T>;
  }

  async download(path: string): Promise<ArrayBuffer> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      headers: this.headers,
    });
    if (!res.ok) throw new Error(`GET ${path} -> ${res.status}`);
    return res.arrayBuffer();
  }
}
