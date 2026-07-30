import { Analytics } from './analytics';
import { Auth } from './auth';
import { Avatars } from './avatars';
import { Databases } from './databases';
import { Deploy } from './deploy';
import { Edge } from './edge';
import { Flags } from './flags';
import { Functions } from './functions';
import { Locale } from './locale';
import { Messaging } from './messaging';
import { Realtime } from './realtime';
import { Regions } from './regions';
import { Search } from './search';
import { Storage } from './storage';
import { Vectors } from './vectors';
import { Workflows } from './workflows';
import { Observe } from './observe';
import { errorFromResponse } from './errors';

export interface ApplAdConfig {
  endpoint: string;
  projectId: string;
}

export class Applad {
  readonly endpoint: string;
  readonly projectId: string;
  private headers: Record<string, string>;

  readonly analytics: Analytics;
  readonly auth: Auth;
  readonly avatars: Avatars;
  readonly databases: Databases;
  readonly deploy: Deploy;
  readonly edge: Edge;
  readonly flags: Flags;
  readonly functions: Functions;
  readonly locale: Locale;
  readonly messaging: Messaging;
  readonly realtime: Realtime;
  readonly regions: Regions;
  readonly search: Search;
  readonly storage: Storage;
  readonly vectors: Vectors;
  readonly workflows: Workflows;
  readonly observe: Observe;

  constructor(config: ApplAdConfig) {
    this.endpoint = config.endpoint.replace(/\/$/, '');
    this.projectId = config.projectId;
    this.headers = {
      'x-applad-project': config.projectId,
      'Content-Type': 'application/json',
    };
    this.analytics = new Analytics(this);
    this.auth = new Auth(this);
    this.avatars = new Avatars(this);
    this.databases = new Databases(this);
    this.deploy = new Deploy(this);
    this.edge = new Edge(this);
    this.flags = new Flags(this);
    this.functions = new Functions(this);
    this.locale = new Locale(this);
    this.messaging = new Messaging(this);
    this.realtime = new Realtime({ endpoint: this.endpoint, projectId: this.projectId });
    this.regions = new Regions(this);
    this.search = new Search(this);
    this.storage = new Storage(this);
    this.vectors = new Vectors(this);
    this.workflows = new Workflows(this);
    this.observe = new Observe(this);
  }

  /**
   * Authenticate as a signed-in user with their session secret / JWT.
   *
   * The backend authenticates users via `Authorization: Bearer <secret>`, so
   * this is an alias for {@link setJWT}. The token is also forwarded to the
   * realtime client so subscriptions to data channels are authorized.
   */
  setSession(sessionId: string): void {
    this.setJWT(sessionId);
  }

  setJWT(jwt: string): void {
    this.headers['Authorization'] = `Bearer ${jwt}`;
    this.realtime.setToken(jwt);
  }

  setKey(key: string): void {
    this.headers['x-applad-key'] = key;
  }

  async call<T = any>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method,
      headers: this.headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw await errorFromResponse(res, method, path);
    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  async upload<T = any>(path: string, formData: FormData): Promise<T> {
    const headers = { ...this.headers };
    delete headers['Content-Type']; // let fetch set multipart boundary
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method: 'POST',
      headers,
      body: formData,
    });
    if (!res.ok) throw await errorFromResponse(res, 'POST', path);
    return res.json() as Promise<T>;
  }

  async download(path: string): Promise<ArrayBuffer> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      headers: this.headers,
    });
    if (!res.ok) throw await errorFromResponse(res, 'GET', path);
    return res.arrayBuffer();
  }
}
