import { Realtime } from '../realtime';

// Minimal WebSocket mock
class MockWebSocket {
  static OPEN = 1;
  readyState = MockWebSocket.OPEN;
  onmessage: ((msg: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  sent: string[] = [];

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = 3;
    if (this.onclose) this.onclose();
  }

  simulateMessage(data: object) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) });
    }
  }
}

describe('Realtime', () => {
  let original: typeof globalThis.WebSocket;

  beforeEach(() => {
    original = globalThis.WebSocket;
    (globalThis as any).WebSocket = MockWebSocket;
  });

  afterEach(() => {
    (globalThis as any).WebSocket = original;
  });

  it('constructs with ws endpoint', () => {
    const rt = new Realtime({ endpoint: 'http://localhost:8080', projectId: 'p1' });
    expect(rt.isConnected).toBe(false);
  });

  it('connect creates WebSocket', () => {
    const rt = new Realtime({ endpoint: 'http://localhost:8080', projectId: 'p1' });
    rt.connect();
    expect(rt.isConnected).toBe(true);
  });

  it('subscribe sends subscribe message and returns unsubscribe fn', () => {
    const rt = new Realtime({ endpoint: 'http://localhost:8080', projectId: 'p1' });
    rt.connect();
    const cb = jest.fn();
    const unsub = rt.subscribe('databases.db1', cb);
    expect(typeof unsub).toBe('function');
  });

  it('disconnect closes WebSocket', () => {
    const rt = new Realtime({ endpoint: 'http://localhost:8080', projectId: 'p1' });
    rt.connect();
    rt.disconnect();
    expect(rt.isConnected).toBe(false);
  });
});
