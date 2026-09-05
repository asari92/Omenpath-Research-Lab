import { beforeEach, describe, expect, it, vi } from "vitest";

import { createSnapshotStore } from "../state/snapshot-store";
import { FakeWebSocket } from "../test/FakeWebSocket";
import { snapshotAt } from "../test/builders";
import { createRealtimeClient } from "./realtime";

describe("realtime client", () => {
  beforeEach(() => {
    FakeWebSocket.reset();
    vi.useFakeTimers();
  });

  it("one initial snapshot reaches the shared store", () => {
    const store = createSnapshotStore();
    const realtime = createRealtimeClient(store, FakeWebSocket);
    realtime.start();

    expect(FakeWebSocket.instances[0]?.url).toBe("ws://localhost:3000/ws/lab");
    FakeWebSocket.instances[0]?.open();
    FakeWebSocket.instances[0]?.message(snapshotAt());
    expect(store.getState().snapshot?.lab.current_energy).toBe(100);
    expect(store.getState().connection).toBe("connected");
  });

  it("disconnect exposes status and bounded 1/2/5/10 second backoff", () => {
    const store = createSnapshotStore();
    const realtime = createRealtimeClient(store, FakeWebSocket);
    realtime.start();
    FakeWebSocket.instances[0]?.close();

    expect(store.getState().connection).toBe("reconnecting");
    expect(realtime.nextDelay()).toBe(1000);
    vi.advanceTimersByTime(1000);
    FakeWebSocket.instances[1]?.close();
    expect(realtime.nextDelay()).toBe(2000);
    vi.advanceTimersByTime(2000);
    FakeWebSocket.instances[2]?.close();
    expect(realtime.nextDelay()).toBe(5000);
  });

  it("stop leaves no live socket or reconnect timer", () => {
    const store = createSnapshotStore();
    const realtime = createRealtimeClient(store, FakeWebSocket);
    realtime.start();
    realtime.stop();
    vi.runAllTimers();

    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(store.getState().connection).toBe("offline");
  });
});
