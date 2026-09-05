import { describe, expect, it, vi } from "vitest";

import { createLiveResource } from "./live-resource";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((ok, fail) => {
    resolve = ok;
    reject = fail;
  });
  return { promise, resolve, reject };
}

async function flush() {
  await Promise.resolve();
  await Promise.resolve();
}

describe("createLiveResource", () => {
  it("coalesces three refresh edges during one GET into one trailing GET", async () => {
    const first = deferred<number>();
    const second = deferred<number>();
    const loader = vi
      .fn<(signal: AbortSignal) => Promise<number>>()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const resource = createLiveResource(loader);

    resource.refresh();
    resource.refresh();
    resource.refresh();
    expect(loader).toHaveBeenCalledOnce();
    first.resolve(1);
    await flush();
    expect(loader).toHaveBeenCalledTimes(2);
    second.resolve(2);
    await flush();
    expect(resource.getSnapshot()).toMatchObject({
      data: 2,
      loading: false,
      error: null,
    });
  });

  it("publishes loader errors and can retry", async () => {
    const loader = vi
      .fn<(signal: AbortSignal) => Promise<number>>()
      .mockRejectedValueOnce(new Error("network unavailable"))
      .mockResolvedValueOnce(7);
    const resource = createLiveResource(loader);
    resource.refresh();
    await flush();
    expect(resource.getSnapshot().error?.message).toBe("network unavailable");
    resource.refresh();
    await flush();
    expect(resource.getSnapshot()).toMatchObject({ data: 7, error: null });
  });

  it("dispose aborts the current GET and discards trailing work", async () => {
    const request = deferred<number>();
    let capturedSignal: AbortSignal | null = null;
    const loader = vi.fn((signal: AbortSignal) => {
      capturedSignal = signal;
      return request.promise;
    });
    const resource = createLiveResource(loader);
    resource.refresh();
    resource.refresh();
    resource.dispose();
    expect(capturedSignal?.aborted).toBe(true);
    request.resolve(1);
    await flush();
    expect(loader).toHaveBeenCalledOnce();
  });
});
