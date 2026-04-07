import AsyncStorage from '@react-native-async-storage/async-storage';

const SESSION_KEY = '@applad_session';

export interface ApplAdConfig {
  endpoint: string;
  projectId: string;
}

export class ApplAdClient {
  readonly endpoint: string;
  readonly projectId: string;
  private jwt: string | null = null;

  constructor(config: ApplAdConfig) {
    this.endpoint = config.endpoint.replace(/\/$/, '');
    this.projectId = config.projectId;
  }

  // ---- Session persistence (AsyncStorage) ----

  async setSession(jwt: string): Promise<void> {
    this.jwt = jwt;
    await AsyncStorage.setItem(SESSION_KEY, jwt);
  }

  async getSession(): Promise<string | null> {
    if (this.jwt) return this.jwt;
    const stored = await AsyncStorage.getItem(SESSION_KEY);
    if (stored) this.jwt = stored;
    return this.jwt;
  }

  async clearSession(): Promise<void> {
    this.jwt = null;
    await AsyncStorage.removeItem(SESSION_KEY);
  }

  // ---- Internal helpers ----

  private async buildHeaders(contentType?: string): Promise<Record<string, string>> {
    const headers: Record<string, string> = {
      'X-Applad-Project': this.projectId,
    };
    if (contentType) {
      headers['Content-Type'] = contentType;
    }
    const token = await this.getSession();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  // ---- HTTP methods ----

  async call<T = any>(method: string, path: string, body?: unknown): Promise<T> {
    const headers = await this.buildHeaders('application/json');
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const text = await res.text().catch(() => '');
      throw new Error(`${method} ${path} -> ${res.status}: ${text}`);
    }
    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  async upload<T = any>(path: string, formData: FormData): Promise<T> {
    const headers = await this.buildHeaders(); // no Content-Type — let fetch set multipart boundary
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      method: 'POST',
      headers,
      body: formData,
    });
    if (!res.ok) {
      const text = await res.text().catch(() => '');
      throw new Error(`POST ${path} -> ${res.status}: ${text}`);
    }
    return res.json() as Promise<T>;
  }

  async download(path: string): Promise<ArrayBuffer> {
    const headers = await this.buildHeaders();
    const res = await fetch(`${this.endpoint}/v1${path}`, {
      headers,
    });
    if (!res.ok) throw new Error(`GET ${path} -> ${res.status}`);
    return res.arrayBuffer();
  }

  async get<T = any>(path: string): Promise<T> {
    return this.call<T>('GET', path);
  }

  async post<T = any>(path: string, body?: unknown): Promise<T> {
    return this.call<T>('POST', path, body);
  }

  async put<T = any>(path: string, body?: unknown): Promise<T> {
    return this.call<T>('PUT', path, body);
  }

  async patch<T = any>(path: string, body?: unknown): Promise<T> {
    return this.call<T>('PATCH', path, body);
  }

  async delete<T = any>(path: string, body?: unknown): Promise<T> {
    return this.call<T>('DELETE', path, body);
  }
}
