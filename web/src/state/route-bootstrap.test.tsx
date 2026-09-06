import { StrictMode } from "react";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createApiClient } from "../api/client";
import { ConnectionState } from "../components/feedback/ConnectionState";
import type { StateSnapshot } from "../api/types";
import { EventLogPage } from "../pages/EventLogPage";
import { PortalDetailsPage } from "../pages/PortalDetailsPage";
import { FakeWebSocket } from "../test/FakeWebSocket";
import { portalDetails, snapshotAt } from "../test/builders";
import { SnapshotProvider } from "./SnapshotProvider";
import { createSnapshotStore } from "./snapshot-store";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

beforeEach(() => {
  FakeWebSocket.reset();
  vi.stubGlobal("WebSocket", FakeWebSocket);
});
afterEach(() => vi.unstubAllGlobals());

describe.each(["/events", "/portals/42"])("fresh direct route %s", (path) => {
  function setup(strict = false, preloaded = false) {
    const first = deferred<StateSnapshot>();
    const second = deferred<StateSnapshot>();
    const store = createSnapshotStore();
    if (preloaded) store.acceptSnapshot(snapshotAt());
    const load = vi.fn(async () => {
      expect(store.getState().bootstrap).toBe("ready");
      return path === "/events" ? [] : portalDetails();
    });
    const state = vi
      .fn()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const api = {
      ...createApiClient(),
      state,
      events: vi.fn(async () => {
        await load();
        return [];
      }),
      portal: vi.fn(async () => {
        await load();
        return portalDetails();
      }),
    };
    const children = (
      <MemoryRouter initialEntries={[path]}>
        <SnapshotProvider api={api} store={store}>
          <ConnectionState />
          <Routes>
            <Route path="/events" element={<EventLogPage />} />
            <Route path="/portals/:id" element={<PortalDetailsPage />} />
          </Routes>
        </SnapshotProvider>
      </MemoryRouter>
    );
    const view = render(
      strict ? <StrictMode>{children}</StrictMode> : children,
    );
    return { ...view, first, second, state, load, store };
  }

  it.each([false, true])(
    "waits for accepted REST bootstrap before dispatch, preloaded=%s",
    async (preloaded) => {
      const view = setup(false, preloaded);
      expect(view.load).not.toHaveBeenCalled();
      expect(FakeWebSocket.instances).toHaveLength(0);
      await act(async () => view.first.resolve(snapshotAt()));
      await waitFor(() => expect(view.load).toHaveBeenCalledOnce());
      expect(FakeWebSocket.instances).toHaveLength(1);
    },
  );

  it.each(["failure", "invalid", "abort"])(
    "does not dispatch on bootstrap %s",
    async (outcome) => {
      const view = setup();
      if (outcome === "abort") view.unmount();
      await act(async () => {
        if (outcome === "failure") view.first.reject(new Error("offline"));
        else
          view.first.resolve(
            snapshotAt(outcome === "invalid" ? "invalid" : undefined),
          );
      });
      expect(view.load).not.toHaveBeenCalled();
      expect(FakeWebSocket.instances).toHaveLength(0);
    },
  );

  it("StrictMode ignores a reordered obsolete bootstrap and dispatches once after the current result", async () => {
    const view = setup(true);
    expect(view.state).toHaveBeenCalledTimes(2);
    expect(view.state.mock.calls[0][0].aborted).toBe(true);
    await act(async () => view.first.resolve(snapshotAt()));
    expect(view.load).not.toHaveBeenCalled();
    expect(FakeWebSocket.instances).toHaveLength(0);
    await act(async () => view.second.resolve(snapshotAt()));
    await waitFor(() => expect(view.load).toHaveBeenCalledOnce());
    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it("retries failed bootstrap through the same gate and socket lifecycle while keeping the last snapshot", async () => {
    const view = setup(false, true);
    const old = view.store.getState().snapshot;
    await act(async () => view.first.reject(new Error("offline")));
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry connection" }));
    expect(view.store.getState().snapshot).toBe(old);
    expect(view.load).not.toHaveBeenCalled();
    expect(FakeWebSocket.instances).toHaveLength(0);
    const next = snapshotAt("2026-09-05T10:00:01Z");
    await act(async () => view.second.resolve(next));
    await waitFor(() => expect(view.load).toHaveBeenCalledOnce());
    expect(view.store.getState().snapshot).toBe(next);
    expect(FakeWebSocket.instances).toHaveLength(1);
  });
});
