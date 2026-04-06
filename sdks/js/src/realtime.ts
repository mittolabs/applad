export interface RealtimeEvent {
  type: string;
  channel: string;
  timestamp: string;
  payload: unknown;
}

export type RealtimeCallback = (event: RealtimeEvent) => void;

/**
 * Realtime WebSocket client for subscribing to live events.
 */
export class Realtime {
  private endpoint: string;
  private projectId: string;
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<RealtimeCallback>> = new Map();

  constructor(opts: { endpoint: string; projectId: string }) {
    this.endpoint = opts.endpoint.replace(/^http/, 'ws');
    this.projectId = opts.projectId;
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  connect(): void {
    if (this.ws) return;
    const url = `${this.endpoint}/v1/realtime?project=${this.projectId}`;
    this.ws = new WebSocket(url);

    this.ws.onmessage = (msg) => {
      try {
        const event: RealtimeEvent = JSON.parse(msg.data);
        const cbs = this.listeners.get(event.channel);
        if (cbs) cbs.forEach((cb) => cb(event));
      } catch {}
    };

    this.ws.onclose = () => {
      this.ws = null;
    };
  }

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
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'unsubscribe', channels: [channel] }));
      }
    };
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
    this.listeners.clear();
  }
}
