export interface RealtimeEvent {
  type: string;
  channel: string;
  timestamp: string;
  payload: unknown;
}

export type RealtimeCallback = (event: RealtimeEvent) => void;

/**
 * Realtime WebSocket client for subscribing to live events.
 *
 * Robust against ordering and disconnects:
 *  - subscriptions issued before the socket opens are buffered and flushed
 *    on open, so `connect()` then `subscribe()` works without a race;
 *  - dropped connections are re-established with capped exponential backoff
 *    and every active channel is re-subscribed on reopen;
 *  - `disconnect()` stops reconnection.
 *
 * Data (row) channels require an authenticated connection. When the parent
 * client has a session set (via `setJWT`/`setSession`), the token is forwarded
 * as `?token=` on the WebSocket URL — browsers may instead rely on the
 * `a_session` cookie.
 */
export class Realtime {
  private endpoint: string;
  private projectId: string;
  private token: string | null = null;
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<RealtimeCallback>> = new Map();
  private closedByUser = false;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(opts: { endpoint: string; projectId: string }) {
    this.endpoint = opts.endpoint.replace(/^http/, 'ws');
    this.projectId = opts.projectId;
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Set (or clear) the session token forwarded on the realtime connection.
   * Called automatically by the parent client's `setJWT`/`setSession`. If a
   * connection is already open it is re-established so the token takes effect.
   */
  setToken(token: string | null): void {
    this.token = token;
    const ws = this.ws;
    if (ws) {
      // Reconnect cleanly to apply the new token; the old socket's onclose
      // is scoped and will no-op once `this.ws` no longer points at it.
      this.ws = null;
      ws.close();
      this.connect();
    }
  }

  connect(): void {
    if (this.ws) return;
    this.closedByUser = false;

    let url = `${this.endpoint}/v1/realtime?project=${this.projectId}`;
    if (this.token) url += `&token=${encodeURIComponent(this.token)}`;

    const ws = new WebSocket(url);
    this.ws = ws;

    ws.onopen = () => {
      this.reconnectAttempts = 0;
      // (Re-)subscribe every active channel: covers subscriptions buffered
      // before open and re-subscription after a reconnect.
      const channels = [...this.listeners.keys()];
      if (channels.length) {
        ws.send(JSON.stringify({ type: 'subscribe', channels }));
      }
    };

    ws.onmessage = (msg) => {
      try {
        const event: RealtimeEvent = JSON.parse(msg.data);
        const cbs = this.listeners.get(event.channel);
        if (cbs) cbs.forEach((cb) => cb(event));
      } catch {}
    };

    ws.onclose = () => {
      if (this.ws !== ws) return; // superseded by a newer socket
      this.ws = null;
      if (!this.closedByUser) this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;
    const delay = Math.min(30000, 1000 * 2 ** this.reconnectAttempts);
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (!this.closedByUser) this.connect();
    }, delay);
  }

  subscribe(channel: string, callback: RealtimeCallback): () => void {
    if (!this.listeners.has(channel)) {
      this.listeners.set(channel, new Set());
    }
    this.listeners.get(channel)!.add(callback);

    // Send now if open; otherwise it is flushed by onopen.
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'subscribe', channels: [channel] }));
    }

    return () => {
      const cbs = this.listeners.get(channel);
      cbs?.delete(callback);
      if (cbs && cbs.size === 0) {
        this.listeners.delete(channel);
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: 'unsubscribe', channels: [channel] }));
        }
      }
    };
  }

  disconnect(): void {
    this.closedByUser = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = 0;
    this.ws?.close();
    this.ws = null;
    this.listeners.clear();
  }
}
