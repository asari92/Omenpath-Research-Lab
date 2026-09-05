import { StrictMode } from "react";
import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { createApiClient } from "../api/client";
import type { StateSnapshot } from "../api/types";
import { snapshotAt } from "../test/builders";
import { FakeWebSocket } from "../test/FakeWebSocket";
import { SnapshotProvider } from "./SnapshotProvider";
import { createSnapshotStore } from "./snapshot-store";

beforeEach(() => {
  FakeWebSocket.reset();
  vi.stubGlobal("WebSocket", FakeWebSocket);
});
afterEach(() => vi.unstubAllGlobals());

function deferred() {
  let resolve!: (s: StateSnapshot) => void;
  let reject!: (e: Error) => void;
  const promise = new Promise<StateSnapshot>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

it("accepts REST before constructing WebSocket and keeps it on reconnect", async () => {
  const pending = deferred();
  const store = createSnapshotStore();
  const api = { ...createApiClient(), state: vi.fn(() => pending.promise) };
  const view = render(
    <SnapshotProvider api={api} store={store}>
      test
    </SnapshotProvider>,
  );
  expect(FakeWebSocket.instances).toHaveLength(0);
  const snapshot = snapshotAt();
  await act(async () => pending.resolve(snapshot));
  expect(store.getState().snapshot).toEqual(snapshot);
  expect(FakeWebSocket.instances).toHaveLength(1);
  act(() => FakeWebSocket.instances[0].close());
  expect(store.getState().snapshot).toEqual(snapshot);
  expect(store.getState().connection).toBe("reconnecting");
  view.unmount();
});

it("never constructs WebSocket after failed bootstrap", async () => {
  const pending = deferred();
  const store = createSnapshotStore();
  const api = { ...createApiClient(), state: () => pending.promise };
  render(
    <SnapshotProvider api={api} store={store}>
      test
    </SnapshotProvider>,
  );
  await act(async () => pending.reject(new Error("offline")));
  expect(store.getState().bootstrap).toBe("failed");
  expect(FakeWebSocket.instances).toHaveLength(0);
});

it("ignores late bootstrap after unmount even if transport ignores abort", async () => {
  const pending = deferred();
  const store = createSnapshotStore();
  const api = { ...createApiClient(), state: () => pending.promise };
  const view = render(
    <SnapshotProvider api={api} store={store}>
      test
    </SnapshotProvider>,
  );
  view.unmount();
  await act(async () => pending.resolve(snapshotAt()));
  expect(store.getState().snapshot).toBeNull();
  expect(FakeWebSocket.instances).toHaveLength(0);
});

it("StrictMode aborts obsolete bootstrap and opens exactly one socket", async () => {
  const first = deferred(),
    second = deferred();
  const state = vi
    .fn()
    .mockReturnValueOnce(first.promise)
    .mockReturnValueOnce(second.promise);
  const api = { ...createApiClient(), state };
  const view = render(
    <StrictMode>
      <SnapshotProvider api={api}>test</SnapshotProvider>
    </StrictMode>,
  );
  expect(state).toHaveBeenCalledTimes(2);
  expect(state.mock.calls[0][0].aborted).toBe(true);
  await act(async () => first.resolve(snapshotAt()));
  expect(FakeWebSocket.instances).toHaveLength(0);
  await act(async () => second.resolve(snapshotAt()));
  expect(FakeWebSocket.instances).toHaveLength(1);
  view.unmount();
  expect(FakeWebSocket.instances[0].readyState).toBe(3);
});

it("does not open WebSocket when bootstrap snapshot is rejected", async () => {
  const api = {
    ...createApiClient(),
    state: async () => snapshotAt("invalid"),
  };
  const store = createSnapshotStore();
  await act(async () => {
    render(
      <SnapshotProvider api={api} store={store}>
        test
      </SnapshotProvider>,
    );
  });
  expect(FakeWebSocket.instances).toHaveLength(0);
  expect(store.getState().bootstrap).toBe("failed");
});
