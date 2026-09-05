import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { ApiError } from "../../api/errors";
import { FeedbackProvider } from "../../components/feedback/FeedbackProvider";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { usePortalCommand } from "./usePortalCommand";

function Harness() {
  const command = usePortalCommand(42);
  return (
    <button
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
  });
});
