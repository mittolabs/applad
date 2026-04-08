import { Applad } from '../client';

function mockFetch(data: any = {}, status = 200) {
  const fn = jest.fn().mockResolvedValue({
    ok: status < 400,
    status,
    json: () => Promise.resolve(data),
  });
  global.fetch = fn;
  return fn;
}

function createClient() {
  return new Applad({ endpoint: 'http://localhost:8080', projectId: 'proj-1' });
}

describe('Auth service', () => {
  afterEach(() => jest.restoreAllMocks());

  it('createAccount sends POST /account', async () => {
    const mock = mockFetch({ $id: 'u1' });
    const client = createClient();
    await client.auth.createAccount('a@b.com', 'pass123');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/account',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"email":"a@b.com"'),
      })
    );
  });

  it('getAccount sends GET /account', async () => {
    const mock = mockFetch({ $id: 'u1', email: 'a@b.com' });
    const client = createClient();
    const res = await client.auth.getAccount();
    expect(res.$id).toBe('u1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/account',
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('createEmailSession sends POST /account/sessions/email', async () => {
    const mock = mockFetch({ $id: 's1', userId: 'u1' });
    const client = createClient();
    await client.auth.createEmailSession('a@b.com', 'pass');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/account/sessions/email',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('listSessions sends GET /account/sessions', async () => {
    mockFetch({ sessions: [], total: 0 });
    const client = createClient();
    const res = await client.auth.listSessions();
    expect(res.sessions).toEqual([]);
  });

  it('deleteSession sends DELETE', async () => {
    const mock = mockFetch({}, 204);
    const client = createClient();
    await client.auth.deleteSession('s1');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/account/sessions/s1',
      expect.objectContaining({ method: 'DELETE' })
    );
  });

  it('updateName sends PATCH /account/name', async () => {
    const mock = mockFetch({ name: 'New' });
    const client = createClient();
    await client.auth.updateName('New');
    expect(mock).toHaveBeenCalledWith(
      'http://localhost:8080/v1/account/name',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ name: 'New' }),
      })
    );
  });

  it('getJWT returns token string', async () => {
    mockFetch({ jwt: 'eyJhbGciOiJIUzI1NiJ9.test' });
    const client = createClient();
    const jwt = await client.auth.getJWT();
    expect(jwt).toBe('eyJhbGciOiJIUzI1NiJ9.test');
  });
});
