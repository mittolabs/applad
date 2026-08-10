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

describe('Databases service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('createDatabase sends POST /databases', async () => {
    const mock = mockFetch({ $id: 'db1', name: 'mydb' });
    const client = createClient();
    await client.databases.createDatabase('mydb');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"name":"mydb"'),
      })
    );
  });

  it('listDatabases sends GET /databases', async () => {
    mockFetch({ databases: [], total: 0 });
    const client = createClient();
    const res = await client.databases.listDatabases();
    expect(res.databases).toEqual([]);
  });

  it('getDatabase sends GET /databases/:id', async () => {
    const mock = mockFetch({ $id: 'db1' });
    const client = createClient();
    await client.databases.getDatabase('db1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('deleteDatabase sends DELETE /databases/:id', async () => {
    const mock = mockFetch({}, 204);
    const client = createClient();
    await client.databases.deleteDatabase('db1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1',
      expect.objectContaining({ method: 'DELETE' })
    );
  });

  it('createTable sends POST with correct path', async () => {
    const mock = mockFetch({ $id: 't1' });
    const client = createClient();
    await client.databases.createTable('db1', 'users');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1/tables',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"name":"users"'),
      })
    );
  });

  it('listTables sends GET to correct path', async () => {
    const mock = mockFetch({ tables: [] });
    const client = createClient();
    await client.databases.listTables('db1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1/tables',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('createStringColumn constructs correct body', async () => {
    const mock = mockFetch({ key: 'name' });
    const client = createClient();
    await client.databases.createStringColumn('db1', 't1', 'name', { required: true, size: 255 });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1/tables/t1/columns/string',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"required":true'),
      })
    );
  });

  it('createStringColumn omits encrypted by default', async () => {
    const mock = mockFetch({ key: 'name' });
    const client = createClient();
    await client.databases.createStringColumn('db1', 't1', 'name');
    const body = JSON.parse(mock.mock.calls[0][1].body);
    expect(body.encrypted).toBeUndefined();
  });

  it('createStringColumn sends encrypted:true when requested', async () => {
    const mock = mockFetch({ key: 'ssn' });
    const client = createClient();
    await client.databases.createStringColumn('db1', 't1', 'ssn', { encrypted: true });
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/databases/db1/tables/t1/columns/string',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"encrypted":true'),
      })
    );
  });

  it('createIntegerColumn sends encrypted:true when requested', async () => {
    const mock = mockFetch({ key: 'salary' });
    const client = createClient();
    await client.databases.createIntegerColumn('db1', 't1', 'salary', { encrypted: true });
    const body = JSON.parse(mock.mock.calls[0][1].body);
    expect(body.encrypted).toBe(true);
  });

  it('createBooleanColumn sends encrypted:true when requested', async () => {
    const mock = mockFetch({ key: 'flag' });
    const client = createClient();
    await client.databases.createBooleanColumn('db1', 't1', 'flag', { encrypted: true });
    const body = JSON.parse(mock.mock.calls[0][1].body);
    expect(body.encrypted).toBe(true);
  });
});
