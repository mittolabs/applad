import { Applad } from '../client';

describe('Applad client', () => {
  it('creates with endpoint and projectId', () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080',
      projectId: 'test-project',
    });
    expect(client.endpoint).toBe('http://localhost:8080');
    expect(client.projectId).toBe('test-project');
  });

  it('strips trailing slash from endpoint', () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080/',
      projectId: 'test',
    });
    expect(client.endpoint).toBe('http://localhost:8080');
  });

  it('exposes service instances', () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080',
      projectId: 'test',
    });
    expect(client.auth).toBeDefined();
    expect(client.databases).toBeDefined();
    expect(client.storage).toBeDefined();
  });

  it('call method builds correct URL', async () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080',
      projectId: 'test',
    });

    // Mock fetch
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: 'test' }),
    });
    global.fetch = mockFetch;

    await client.call('GET', '/health');

    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/v1/health',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          'x-applad-project': 'test',
        }),
      })
    );
  });

  it('call throws on non-ok response', async () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080',
      projectId: 'test',
    });

    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 404,
    });

    await expect(client.call('GET', '/missing')).rejects.toThrow('404');
  });

  it('setJWT sets Authorization header', async () => {
    const client = new Applad({
      endpoint: 'http://localhost:8080',
      projectId: 'test',
    });

    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });

    client.setJWT('my-token');
    await client.call('GET', '/test');

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer my-token',
        }),
      })
    );
  });
});
