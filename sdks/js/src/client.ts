import { Auth } from './auth';
import { Databases } from './databases';
import { Storage } from './storage';
import { Workflows } from './workflows';

export interface ApplAdConfig {
  endpoint: string;
  projectId: string;
}

export class Applad {
  readonly endpoint: string;
  readonly projectId: string;
  private headers: Record<string, string>;

  readonly auth: Auth;
  readonly databases: Databases;
  readonly storage: Storage;
  readonly workflows: Workflows;

  constructor(config: ApplAdConfig) {
    this.endpoint = config.endpoint.replace(/\/$/, '');
    this.projectId = config.projectId;
    this.headers = {
      'x-applad-project': config.projectId,
      'Content-Type': 'application/json',
    };
    this.auth = new Auth(this);
    this.databases = new Databases(this);
    this.storage = new Storage(this);
    this.workflows = new Workflows(this);
  }

  setSession(sessionId: string): void {
    this.headers['x-applad-session'] = sessionId;
  }

  setJWT(jwt: string): void {
    this.headers['Authorization'] = `Bearer ${jwt}`;
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
    if (!res.ok) throw new Error(`${method} ${path} → ${res.status}`);
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
    if (!res.ok) throw new Error(`POST ${path} → ${res.status}`);
    return res.json() as Promise<T>;
  }

  async download(path: string): Promise<ArrayBuffer> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      headers: this.headers,
    });
    if (!res.ok) throw new Error(`GET ${path} → ${res.status}`);
    return res.arrayBuffer();
  }
}
