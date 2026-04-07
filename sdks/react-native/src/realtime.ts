import type { ApplAdClient } from './client';

export interface RealtimeEvent {
  type: string;
  channel: string;
  timestamp: string;
  payload: unknown;
}

export type RealtimeCallback = (event: RealtimeEvent) => void;

/**
 * Realtime WebSocket client for React Native.
 * Uses the built-in WebSocket global available in React Native.
 * Supports auto-reconnect with exponential backoff.
 */
export class Realtime {
  private client: ApplAdClient;
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<RealtimeCallback>> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private shouldReconnect = false;

  constructor(client: ApplAdClient) {
    this.client = client;
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /** Open the WebSocket connection. Attaches the JWT if a session exists. */
  async connect(): Promise<void> {
    if (this.ws) return;
    this.shouldReconnect = true;

    const token = await this.client.getSession();
    const wsEndpoint = this.client.endpoint.replace(/^http/, 'ws');
    let url = `${wsEndpoint}/v1/realtime?project=${this.client.projectId}`;
    if (token) {
      url += `&token=${encodeURIComponent(token)}`;
    }

    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      // Re-subscribe to existing channels on reconnect
      if (this.listeners.size > 0) {
        const channels = Array.from(this.listeners.keys());
        this.ws?.send(JSON.stringify({ type: 'subscribe', channels }));
      }
    };

    this.ws.onmessage = (msg) => {
      try {
        const event: RealtimeEvent = JSON.parse(typeof msg.data === 'string' ? msg.data : '');
        const cbs = this.listeners.get(event.channel);
        if (cbs) cbs.forEach((cb) => cb(event));
      } catch {
        // ignore malformed messages
      }
    };

    this.ws.onerror = () => {
      // onclose will fire after onerror
    };

    this.ws.onclose = () => {
      this.ws = null;
      if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
        this.reconnectAttempts++;
        this.reconnectTimer = setTimeout(() => {
          this.connect();
        }, delay);
      }
    };
  }

  /**
   * Subscribe to a channel. Returns an unsubscribe function.
   *
   * @example
   * const unsub = realtime.subscribe('databases.mydb.tables.posts.rows', (event) => {
   *   console.log('Row changed:', event.payload);
   * });
   * // later:
   * unsub();
   */
  subscribe(channel: string, callback: RealtimeCallback): () => void {
    if (!this.listeners.has(channel)) {
      this.listeners.set(channel, new Set());
    }
    this.listeners.get(channel)!.add(callback);

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'subscribe', channels: [channel] }));
    }

    return () => {
      this.listeners.get(channel)?.delete(callback);
      if (this.listeners.get(channel)?.size === 0) {
        this.listeners.delete(channel);
      }
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'unsubscribe', channels: [channel] }));
      }
    };
  }

  /** Disconnect and stop auto-reconnect. */
  disconnect(): void {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
    this.listeners.clear();
    this.reconnectAttempts = 0;
  }
}
