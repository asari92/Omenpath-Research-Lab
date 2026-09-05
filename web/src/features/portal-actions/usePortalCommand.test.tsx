import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { ApiError } from "../../api/errors";
import { FeedbackProvider } from "../../components/feedback/FeedbackProvider";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { usePortalCommand } from "./usePortalCommand";

beforeEach(() => vi.stubGlobal("WebSocket", undefined));
afterEach(() => vi.unstubAllGlobals());

function Harness() {
  const command = usePortalCommand(42);
  return (
    <button
      data-outcome={command.outcome}
      disabled={command.busyKey === "42:CLOSE"}
      onClick={() => void command.run("CLOSE")}
      type="button"
    >
      Close
    </button>
  );
}

function setup(close: ReturnType<typeof vi.fn>) {
  const snapshot = snapshotAt();
  snapshot.slots[0].portal = {
    id: 42,
    name: "Omenpath #0042",
    destination_plane_id: 1,
    destination_plane_name: "Agyrem",
    destination_explored: false,
    energy: 50,
    stability: "STABLE",
    time_remaining_seconds: 60,
    creatures_inside: 0,
    status: "OPEN",
    quick_actions: {
      can_stabilize: false,
      stabilize_unavailable_reason: "PORTAL_ALREADY_STABLE",
      can_close: true,
      close_unavailable_reason: null,
      can_send_observer: true,
      send_observer_unavailable_reason: null,
      can_recall_observer: false,
      recall_observer_unavailable_reason: "NO_WAITING_OBSERVER",
    },
  };
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  store.setConnection("connected");
  const api = { state: async () => snapshot, close } as unknown as OmenpathApi;
  render(
    <SnapshotProvider api={api} store={store}>
      <FeedbackProvider>
        <Harness />
      </FeedbackProvider>
    </SnapshotProvider>,
  );
  return { store };
}

describe("usePortalCommand confirmation", () => {
  it("does not retry confirmation after connection is lost", async () => {
    const close = vi
      .fn()
      .mockRejectedValue(
        new ApiError(
          409,
          "CONFIRMATION_REQUIRED",
          true,
          "confirmation required",
        ),
      );
    const { store } = setup(close);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Close" }));
    act(() => store.setConnection("reconnecting"));
    await user.click(screen.getByRole("button", { name: "Confirm Close" }));
    expect(close).toHaveBeenCalledTimes(1);
  });
  it.each(["reconnecting", "protocol", "bootstrap"])(
    "blocks unhealthy %s commands without POST or losing the accepted snapshot",
    async (failure) => {
      const close = vi.fn();
      const { store } = setup(close);
      const snapshot = store.getState().snapshot;
      await act(async () => {});
      act(() => {
        if (failure === "protocol") store.setProtocolError("Invalid snapshot");
        else if (failure === "bootstrap") store.setBootstrap("failed");
        else store.setConnection("reconnecting");
      });
      await userEvent
        .setup()
        .click(screen.getByRole("button", { name: "Close" }));
      expect(close).not.toHaveBeenCalled();
      expect(store.getState().snapshot).toBe(snapshot);
      expect(screen.getByRole("status")).toHaveTextContent(
        "Planar paths unstable",
      );
    },
  );

  it("shows pending, prevents a duplicate and acknowledges authoritative success", async () => {
    let finish!: (value: ReturnType<typeof snapshotAt>) => void;
    const close = vi.fn(
      () =>
        new Promise<ReturnType<typeof snapshotAt>>((resolve) => {
          finish = resolve;
        }),
    );
    const { store } = setup(close);
    const before = store.getState().snapshot;
    const button = screen.getByRole("button", { name: "Close" });
    const user = userEvent.setup();
    await user.click(button);
    expect(button).toBeDisabled();
    expect(store.getState().snapshot).toBe(before);
    await user.click(button);
    expect(close).toHaveBeenCalledOnce();
    const next = snapshotAt("2026-09-05T10:00:01Z");
    await act(async () => finish(next));
    expect(button).not.toBeDisabled();
    expect(button).toHaveAttribute("data-outcome", "success");
    expect(screen.getByRole("status")).toHaveTextContent("CLOSE completed");
    expect(store.getState().snapshot).toBe(next);
  });
  it("repeats the same method/id exactly once with confirm=true", async () => {
    const next = snapshotAt("2026-09-05T10:00:01Z");
    const close = vi
      .fn()
      .mockRejectedValueOnce(
        new ApiError(
          409,
          "CONFIRMATION_REQUIRED",
          true,
          "confirmation required",
        ),
      )
      .mockResolvedValueOnce(next);
    const { store } = setup(close);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("Omenpath #0042");
    await user.click(screen.getByRole("button", { name: "Confirm Close" }));
    expect(close.mock.calls).toEqual([
      [42, false],
      [42, true],
    ]);
    expect(store.getState().snapshot).toBe(next);
  });

  it("cancel sends no second request and returns focus", async () => {
    const close = vi
      .fn()
      .mockRejectedValue(
        new ApiError(
          409,
          "CONFIRMATION_REQUIRED",
          true,
          "confirmation required",
        ),
      );
    setup(close);
    const user = userEvent.setup();
    const button = screen.getByRole("button", { name: "Close" });
    await user.click(button);
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(close).toHaveBeenCalledTimes(1);
    expect(button).toHaveFocus();
  });

  it("clears busy state and shows safe text after non-confirmable failure", async () => {
    const close = vi
      .fn()
      .mockRejectedValue(
        new ApiError(409, "PORTAL_BUSY", false, "Portal is busy"),
      );
    setup(close);
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Close" }));
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Portal is busy",
    );
    expect(screen.getByRole("button", { name: "Close" })).not.toBeDisabled();
    expect(screen.getByRole("button", { name: "Close" })).toHaveAttribute(
      "data-outcome",
      "error",
    );
  });
});
