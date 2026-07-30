import { Applad } from '../client';

// Shared mock setup
function createClient() {
  return new Applad({ endpoint: 'http://localhost:8080', projectId: 'test-project' });
}

function mockFetch(status = 200, data: any = {}) {
  const fn = jest.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
  });
  global.fetch = fn;
  return fn;
}

describe('Applad client', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('creates with endpoint and projectId', () => {
    const client = createClient();
    expect(client.endpoint).toBe('http://localhost:8080');
    expect(client.projectId).toBe('test-project');
  });

  it('strips trailing slash from endpoint', () => {
    const client = new Applad({ endpoint: 'http://localhost:8080/', projectId: 'test' });
    expect(client.endpoint).toBe('http://localhost:8080');
  });

  it('exposes all service instances', () => {
    const client = createClient();
    expect(client.auth).toBeDefined();
    expect(client.avatars).toBeDefined();
    expect(client.databases).toBeDefined();
    expect(client.deploy).toBeDefined();
    expect(client.flags).toBeDefined();
    expect(client.functions).toBeDefined();
    expect(client.locale).toBeDefined();
    expect(client.messaging).toBeDefined();
    expect(client.realtime).toBeDefined();
    expect(client.storage).toBeDefined();
    expect(client.workflows).toBeDefined();
  });

  it('call method builds correct URL and headers', async () => {
    const client = createClient();
    const mock = mockFetch(200, { data: 'test' });

    await client.call('GET', '/health');

    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/health',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          'x-applad-project': 'test-project',
          'Content-Type': 'application/json',
        }),
      })
    );
  });

  it('call sends JSON body for POST', async () => {
    const client = createClient();
    const mock = mockFetch(200, { ok: true });

    await client.call('POST', '/test', { key: 'value' });

    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/test',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ key: 'value' }),
      })
    );
  });

  it('call returns undefined for 204 responses', async () => {
    const client = createClient();
    mockFetch(204);

    const result = await client.call('DELETE', '/test');
    expect(result).toBeUndefined();
  });

  it('call throws on non-ok response', async () => {
    const client = createClient();
    mockFetch(404);
    await expect(client.call('GET', '/missing')).rejects.toThrow('404');
  });

  it('setJWT sets Authorization header', async () => {
    const client = createClient();
    const mock = mockFetch(200, {});
    client.setJWT('my-token');
    await client.call('GET', '/test');

    expect(mock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer my-token',
        }),
      })
    );
  });

  it('setKey sets x-applad-key header', async () => {
    const client = createClient();
    const mock = mockFetch(200, {});
    client.setKey('api-key-123');
    await client.call('GET', '/test');

    expect(mock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          'x-applad-key': 'api-key-123',
        }),
      })
    );
  });

  it('setSession sets Authorization: Bearer header (alias of setJWT)', async () => {
    const client = createClient();
    const mock = mockFetch(200, {});
    client.setSession('sess-123');
    await client.call('GET', '/test');

    expect(mock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer sess-123',
        }),
      })
    );
  });

  it('call throws AppladError with server message and status', async () => {
    const client = createClient();
    mockFetch(404, { message: 'row not found', type: 'not_found' });
    await expect(client.call('GET', '/missing')).rejects.toMatchObject({
      name: 'AppladError',
      status: 404,
      message: 'row not found',
      type: 'not_found',
    });
  });

  it('download returns ArrayBuffer', async () => {
    const client = createClient();
    mockFetch(200);
    const result = await client.download('/files/123');
    expect(result).toBeInstanceOf(ArrayBuffer);
  });
});
