export interface LiveResourceSnapshot<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

export interface LiveResource<T> {
  getSnapshot(): LiveResourceSnapshot<T>;
  subscribe(listener: () => void): () => void;
  refresh(): void;
  dispose(): void;
}

export function createLiveResource<T>(
  load: (signal: AbortSignal) => Promise<T>,
): LiveResource<T> {
  let state: LiveResourceSnapshot<T> = {
    data: null,
    loading: false,
    error: null,
  };
  let controller: AbortController | null = null;
  let trailing = false;
  let disposed = false;
  const listeners = new Set<() => void>();

  const publish = (next: LiveResourceSnapshot<T>) => {
    if (disposed) return;
    state = next;
    listeners.forEach((listener) => listener());
  };

  const start = () => {
    if (disposed) return;
    if (controller) {
      trailing = true;
      return;
    }
    controller = new AbortController();
    const active = controller;
    publish({ ...state, loading: true, error: null });
    void load(active.signal)
      .then((data) => {
        if (!active.signal.aborted)
          publish({ data, loading: false, error: null });
      })
      .catch((error: unknown) => {
        if (!active.signal.aborted) {
          publish({
            data: state.data,
            loading: false,
            error: error instanceof Error ? error : new Error("Request failed"),
          });
        }
      })
      .finally(() => {
        if (controller !== active) return;
        controller = null;
        if (trailing && !disposed) {
          trailing = false;
          start();
        }
      });
  };

  return {
    getSnapshot: () => state,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    refresh: start,
    dispose() {
      disposed = true;
      trailing = false;
      controller?.abort();
      controller = null;
      listeners.clear();
    },
  };
}
