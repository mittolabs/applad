import { afterEach, describe, expect, it, vi } from 'vitest';
import { loadChunk } from './chunk';

const reload = vi.fn();

Object.defineProperty(window, 'location', {
  value: { reload },
  writable: true,
});

afterEach(() => {
  reload.mockClear();
  window.sessionStorage.clear();
});

describe('loadChunk', () => {
  it('returns the module when the chunk loads', async () => {
    await expect(loadChunk(async () => ({ ok: true }))).resolves.toEqual({ ok: true });
    expect(reload).not.toHaveBeenCalled();
  });

  it('reloads once when the chunk is gone, rather than resolving', async () => {
    let settled = false;
    void loadChunk(() => Promise.reject(new Error('Failed to fetch dynamically imported module')))
      .then(() => (settled = true))
      .catch(() => (settled = true));

    await Promise.resolve();
    await Promise.resolve();

    expect(reload).toHaveBeenCalledTimes(1);
    // Nothing renders in the moment before the page goes away.
    expect(settled).toBe(false);
  });

  it('gives up rather than reloading forever', async () => {
    window.sessionStorage.setItem('applad.chunk-reloaded', '1');
    await expect(loadChunk(() => Promise.reject(new Error('gone')))).rejects.toThrow('gone');
    expect(reload).not.toHaveBeenCalled();
  });

  it('forgets the reload once a chunk loads again', async () => {
    window.sessionStorage.setItem('applad.chunk-reloaded', '1');
    await loadChunk(async () => ({}));
    expect(window.sessionStorage.getItem('applad.chunk-reloaded')).toBeNull();
  });
});
