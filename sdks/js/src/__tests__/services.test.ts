import { Applad } from '../client';

function mockFetch(data: any = {}, status = 200) {
  const fn = jest.fn().mockResolvedValue({
    ok: status < 400, status,
    json: () => Promise.resolve(data),
  });
  global.fetch = fn;
  return fn;
}

function createClient() {
  return new Applad({ endpoint: 'http://localhost:8080', projectId: 'proj-1' });
}

describe('Functions service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('create sends POST /functions', async () => {
    const mock = mockFetch({ $id: 'fn1' });
    const client = createClient();
    await client.functions.create('myFunc', 'node-18');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/functions',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"runtime":"node-18"'),
      })
    );
  });

  it('list sends GET /functions', async () => {
    mockFetch({ functions: [], total: 0 });
    const client = createClient();
    const res = await client.functions.list();
    expect(res.functions).toEqual([]);
  });

  it('execute sends POST to correct path', async () => {
    const mock = mockFetch({ output: 'hello' });
    const client = createClient();
    await client.functions.execute('fn1', { input: 'world' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/functions/fn1/executions',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('listExecutions sends GET', async () => {
    const mock = mockFetch({ executions: [] });
    const client = createClient();
    await client.functions.listExecutions('fn1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/functions/fn1/executions',
      expect.objectContaining({ method: 'GET' })
    );
  });
});

describe('Deploy service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('create sends POST /deploy', async () => {
    const mock = mockFetch({ $id: 'd1' });
    const client = createClient();
    await client.deploy.create('staging', 'docker');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/deploy',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"name":"staging"'),
      })
    );
  });

  it('list sends GET /deploy', async () => {
    mockFetch({ deployments: [] });
    const client = createClient();
    await client.deploy.list();
  });

  it('delete sends DELETE /deploy/:id', async () => {
    const mock = mockFetch({}, 204);
    const client = createClient();
    await client.deploy.delete('d1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/deploy/d1',
      expect.objectContaining({ method: 'DELETE' })
    );
  });
});

describe('Messaging service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('sendEmail sends POST /messaging/email', async () => {
    const mock = mockFetch({ success: true });
    const client = createClient();
    await client.messaging.sendEmail(['a@b.com'], 'Hello', '<p>Hi</p>');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/messaging/email',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"subject":"Hello"'),
      })
    );
  });
});

describe('Workflows service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('create sends POST /workflows', async () => {
    const mock = mockFetch({ $id: 'wf1' });
    const client = createClient();
    await client.workflows.create('my-workflow');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/workflows',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"name":"my-workflow"'),
      })
    );
  });

  it('list sends GET /workflows', async () => {
    mockFetch({ workflows: [] });
    const client = createClient();
    await client.workflows.list();
  });
});

describe('Flags service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('getFlag sends GET /flags/evaluate/:key', async () => {
    const mock = mockFetch({ value: true });
    const client = createClient();
    await client.flags.getFlag('dark-mode');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/flags/evaluate/dark-mode',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('evaluateFlag sends POST /flags/evaluate', async () => {
    const mock = mockFetch({ value: 'variant-a' });
    const client = createClient();
    await client.flags.evaluateFlag('experiment', { userId: 'u1' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/flags/evaluate',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"key":"experiment"'),
      })
    );
  });
});

describe('Analytics service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('trackEvent sends POST /analytics/events', async () => {
    const mock = mockFetch({ success: true });
    const client = createClient();
    await client.analytics.trackEvent('signup', { source: 'web' });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/analytics/events',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"event":"signup"'),
      })
    );
  });
});

describe('Search service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('query sends POST /search/indexes/:id/search', async () => {
    const mock = mockFetch({ documents: [] });
    const client = createClient();
    await client.search.query('idx1', 'hello');
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
    const client = createClient();
    await client.vectors.query('vec1', [0.1, 0.2]);
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
    const client = createClient();
    await client.edge.invoke('edge1', { name: 'test' });
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
    const client = createClient();
    await client.regions.setActive('fra1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/regions/active',
      expect.objectContaining({ method: 'PUT' })
    );
  });
});
