import { ApplAdServer } from '../client';
import { AppladError } from '../errors';

function mockFetch(data: any = {}, status = 200) {
  const fn = jest.fn().mockResolvedValue({
    ok: status < 400, status,
    json: () => Promise.resolve(data),
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
  });
  global.fetch = fn as any;
  return fn;
}

function createServer() {
  return new ApplAdServer({
    endpoint: 'http://localhost:8080',
    projectId: 'proj-1',
    apiKey: 'key-123',
  });
}

describe('ApplAdServer client', () => {
  afterEach(() => jest.restoreAllMocks());

  it('creates with endpoint, projectId and apiKey', () => {
    const srv = createServer();
    expect(srv.endpoint).toBe('http://localhost:8080');
    expect(srv.projectId).toBe('proj-1');
  });

  it('strips trailing slash', () => {
    const srv = new ApplAdServer({
      endpoint: 'http://localhost:8080/',
      projectId: 'p', apiKey: 'k',
    });
    expect(srv.endpoint).toBe('http://localhost:8080');
  });

  it('exposes all service instances', () => {
    const srv = createServer();
    expect(srv.analytics).toBeDefined();
    expect(srv.users).toBeDefined();
    expect(srv.databases).toBeDefined();
    expect(srv.storage).toBeDefined();
    expect(srv.edge).toBeDefined();
    expect(srv.functions).toBeDefined();
    expect(srv.teams).toBeDefined();
    expect(srv.workflows).toBeDefined();
    expect(srv.messaging).toBeDefined();
    expect(srv.deploy).toBeDefined();
    expect(srv.flags).toBeDefined();
    expect(srv.regions).toBeDefined();
    expect(srv.search).toBeDefined();
    expect(srv.vectors).toBeDefined();
  });

  it('call sends project and key headers', async () => {
    const mock = mockFetch({ ok: true });
    const srv = createServer();
    await srv.call('GET', '/health');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/health',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          'X-Applad-Project': 'proj-1',
          'X-Applad-Key': 'key-123',
        }),
      })
    );
  });

  it('call throws on error response', async () => {
    mockFetch({}, 500);
    const srv = createServer();
    await expect(srv.call('GET', '/fail')).rejects.toThrow('500');
  });

  it('call throws AppladError carrying server message, status and type', async () => {
    mockFetch({ message: 'invalid api key', type: 'unauthorized' }, 401);
    const srv = createServer();
    await expect(srv.call('GET', '/fail')).rejects.toMatchObject({
      name: 'AppladError',
      status: 401,
      message: 'invalid api key',
      type: 'unauthorized',
    });
    expect(AppladError).toBeDefined();
  });

  it('call returns undefined for 204', async () => {
    mockFetch({}, 204);
    const srv = createServer();
    const res = await srv.call('DELETE', '/resource');
    expect(res).toBeUndefined();
  });
});

describe('Users service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('listUsers sends GET /users', async () => {
    const mock = mockFetch({ users: [], total: 0 });
    const srv = createServer();
    await srv.users.listUsers();
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/users',
      expect.objectContaining({ method: 'GET' })
    );
  });
});

describe('Databases service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('listDatabases sends GET /databases', async () => {
    const mock = mockFetch({ databases: [] });
    const srv = createServer();
    await srv.databases.listDatabases();
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases',
      expect.objectContaining({ method: 'GET' })
    );
  });
});

describe('Flags service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('list sends GET /flags', async () => {
    const mock = mockFetch({ flags: [] });
    const srv = createServer();
    await srv.flags.list();
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/flags',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('create sends POST /flags', async () => {
    const mock = mockFetch({ key: 'dark-mode' });
    const srv = createServer();
    await srv.flags.create('dark-mode', 'Dark Mode');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/flags',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"key":"dark-mode"'),
      })
    );
  });

  it('toggle sends PATCH /flags/:key/toggle', async () => {
    const mock = mockFetch({ enabled: false });
    const srv = createServer();
    await srv.flags.toggle('dark-mode', false);
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/flags/dark-mode/toggle',
      expect.objectContaining({ method: 'PATCH' })
    );
  });
});

describe('Deploy service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('list sends GET /deploy/targets', async () => {
    const mock = mockFetch({ targets: [], total: 0 });
    const srv = createServer();
    await srv.deploy.list();
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/deploy/targets',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('deploy triggers POST /deploy/targets/:id/executions', async () => {
    const mock = mockFetch({ $id: 'exec1' });
    const srv = createServer();
    await srv.deploy.deploy('d1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/deploy/targets/d1/executions',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('Messaging service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('sendEmail sends POST /messaging/email', async () => {
    const mock = mockFetch({ success: true });
    const srv = createServer();
    await srv.messaging.sendEmail(['a@b.com'], 'Test', { html: '<p>Hi</p>' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/messaging/email',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"subject":"Test"'),
      })
    );
  });

  it('sendSMS posts an array "to" and a "body"', async () => {
    const mock = mockFetch({ status: 'sent' });
    const srv = createServer();
    await srv.messaging.sendSMS(['+15551112222', '+15553334444'], 'Hello');
    const body = JSON.parse((mock.mock.calls[0][1] as RequestInit).body as string);
    expect(mock.mock.calls[0][0]).toBe('http://localhost:8080/v1/messaging/sms');
    expect(body).toEqual({ to: ['+15551112222', '+15553334444'], body: 'Hello' });
  });

  it('sendPush posts an array "to", title/body and optional data', async () => {
    const mock = mockFetch({ status: 'sent' });
    const srv = createServer();
    await srv.messaging.sendPush(['dev1', 'dev2'], 'Hi', 'there', { data: { k: 'v' } });
    const body = JSON.parse((mock.mock.calls[0][1] as RequestInit).body as string);
    expect(mock.mock.calls[0][0]).toBe('http://localhost:8080/v1/messaging/push');
    expect(body).toEqual({ to: ['dev1', 'dev2'], title: 'Hi', body: 'there', data: { k: 'v' } });
  });
});

describe('Workflows service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('list sends GET /workflows', async () => {
    const mock = mockFetch({ workflows: [] });
    const srv = createServer();
    await srv.workflows.list();
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/workflows',
      expect.objectContaining({ method: 'GET' })
    );
  });
});

describe('Analytics service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('trackEvent sends POST /analytics/events', async () => {
    const mock = mockFetch({ success: true });
    const srv = createServer();
    await srv.analytics.trackEvent('signup', { source: 'web' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/analytics/events',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('Search service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('query sends POST /search/indexes/:id/search', async () => {
    const mock = mockFetch({ documents: [] });
    const srv = createServer();
    await srv.search.query('idx1', 'hello');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/search/indexes/idx1/search',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('Vectors service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('query sends POST /vectors/indexes/:id/query', async () => {
    const mock = mockFetch({ matches: [] });
    const srv = createServer();
    await srv.vectors.query('vec1', [0.1, 0.2]);
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/vectors/indexes/vec1/query',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('Edge service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('invoke sends POST /edge/functions/:id/invoke', async () => {
    const mock = mockFetch({ ok: true });
    const srv = createServer();
    await srv.edge.invoke('edge1', { name: 'test' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/edge/functions/edge1/invoke',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('Regions service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('setActive sends PUT /regions/active', async () => {
    const mock = mockFetch({ regionId: 'fra1' });
    const srv = createServer();
    await srv.regions.setActive('fra1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/regions/active',
      expect.objectContaining({ method: 'PUT' })
    );
  });
});
