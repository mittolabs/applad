/** Recovery for a lazily-loaded route whose chunk is no longer on the server. */

// A chunk that will not load is nearly always a chunk that is no longer there:
// the tab has been open across a release, and the index.html it was served
// names files this build replaced. There is nothing to retry — the fix is to
// go and fetch the new index.html, which is what a reload does.
//
// The session flag stops that becoming a loop. If the page comes back and the
// import fails again, the build really is broken and the error belongs on
// screen rather than behind an endless refresh.
const RELOADED_KEY = 'applad.chunk-reloaded';

function flag(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    // Storage can be denied outright; recovery is a nicety, not a requirement.
    return null;
  }
}

export async function loadChunk<M>(loader: () => Promise<M>): Promise<M> {
  try {
    const loaded = await loader();
    flag()?.removeItem(RELOADED_KEY);
    return loaded;
  } catch (error) {
    const store = flag();
    if (store && !store.getItem(RELOADED_KEY)) {
      store.setItem(RELOADED_KEY, '1');
      window.location.reload();
      // The page is on its way out; never resolve, so nothing renders first.
      return new Promise<M>(() => {});
    }
    throw error;
  }
}
