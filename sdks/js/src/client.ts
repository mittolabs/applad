export interface ApplAdConfig {
  endpoint: string;
  projectId: string;
}

export class Applad {
  readonly endpoint: string;
  readonly projectId: string;
  private headers: Record<string, string>;

  constructor(config: ApplAdConfig) {
    this.endpoint = config.endpoint.replace(/\/$/, '');
    this.projectId = config.projectId;
    this.headers = {
      'x-applad-project': config.projectId,
      'Content-Type': 'application/json',
    };
  }

  setSession(sessionId: string): void {
    this.headers['x-applad-session'] = sessionId;
  }

  setJWT(jwt: string): void {
    this.headers['Authorization'] = `Bearer ${jwt}`;
  }

  async call<T>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method,
      headers: this.headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`${method} ${path} → ${res.status}`);
    return res.json() as Promise<T>;
  }
}
