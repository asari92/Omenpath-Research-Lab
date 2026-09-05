import { act, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import type { OmenpathApi } from "../api/client";
import type { SlotPortalDTO, StateSnapshot } from "../api/types";
import { SnapshotProvider } from "../state/SnapshotProvider";
import { createSnapshotStore } from "../state/snapshot-store";
import { observerTransit, portalDetails, snapshotAt } from "../test/builders";
import { DashboardPage } from "./DashboardPage";

function portal(id: number, plane: string): SlotPortalDTO {
  return {
    id,
    name: `Omenpath #${String(id).padStart(4, "0")}`,
    destination_plane_id: id,
    destination_plane_name: plane,
    destination_explored: false,
    energy: 64.26,
    stability: "UNSTABLE",
    time_remaining_seconds: 125,
    creatures_inside: 2,
    status: "OPEN",
    quick_actions: {
      can_stabilize: true,
      stabilize_unavailable_reason: null,
      can_close: true,
      close_unavailable_reason: null,
      can_send_observer: false,
      send_observer_unavailable_reason: "PORTAL_CREATURES_PRESENT",
      can_recall_observer: false,
      recall_observer_unavailable_reason: "NO_WAITING_OBSERVER",
    },
  };
}

function renderDashboard(snapshot: StateSnapshot) {
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  const api: OmenpathApi = {
    state: async () => snapshot,
    portal: async (id) => portalDetails(id),
    events: async () => [],
    stabilize: async () => snapshot,
    close: async () => snapshot,
    sendObserver: async () => snapshot,
    recallObserver: async () => snapshot,
    openExtraction: async () => snapshot,
    startTutorial: async () => snapshot,
    resetTutorial: async () => snapshot,
    tutorialSignal: async () => snapshot,
    startLive: async () => snapshot,
  };
  const view = render(
    <MemoryRouter>
      <SnapshotProvider api={api} store={store}>
        <DashboardPage />
      </SnapshotProvider>
    </MemoryRouter>,
  );
  return { ...view, store };
}

describe("DashboardPage", () => {
  it("shows the scoped transit and whole-card instability while empty commands are disabled", () => {
    const snapshot = snapshotAt();
    snapshot.slots[0].portal = portal(1, "Alara");
    snapshot.observer_transits = [observerTransit(1, 17, "RETURNING")];
    renderDashboard(snapshot);
    const slots = screen.getAllByTestId("portal-slot");
    expect(slots[0]).toHaveAttribute("data-stability", "UNSTABLE");
    expect(slots[0]).toHaveTextContent("Observer #17");
    expect(slots[0]).toHaveTextContent("Returning · 00:05");
    expect(slots[0]).toHaveTextContent("UNEXPLORED");
    for (const button of within(slots[1]).getAllByRole("button")) expect(button).toBeDisabled();
  });
  it("renders every summary value from one authoritative snapshot", () => {
    const snapshot = snapshotAt();
    snapshot.lab.current_energy = 73;
    snapshot.exploration.explored = 11;
    snapshot.observers.available = 4;
    snapshot.observers.in_worlds = 3;
    snapshot.observers.in_transit = 2;
    snapshot.observers.lost = 1;
    snapshot.portals.active = 2;
    snapshot.portals.critical = 1;
    renderDashboard(snapshot);

    const summary = screen.getByLabelText("Laboratory summary");
    for (const text of [
      "73 / 100",
      "11 / 85",
      "Available 4",
      "In worlds 3",
      "In transit 2",
      "Lost 1",
      "Active 2 / 7",
      "Critical 1",
    ]) {
      expect(summary).toHaveTextContent(text);
    }
  });

  it("renders exactly Slot 1..7 by slot_index without leaking diagnostics", () => {
    const snapshot = snapshotAt();
    snapshot.slots[0].portal = portal(1, "Alara");
    snapshot.slots[2].portal = portal(3, "Amonkhet");
    snapshot.slots = [
      snapshot.slots[2],
      snapshot.slots[0],
      ...snapshot.slots.slice(3),
      snapshot.slots[1],
    ];
    renderDashboard(snapshot);

    const slots = screen.getAllByTestId("portal-slot");
    expect(slots).toHaveLength(7);
    expect(
      slots.map((slot) => within(slot).getByText(/Slot \d/).textContent),
    ).toEqual([
      "Slot 1",
      "Slot 2",
      "Slot 3",
      "Slot 4",
      "Slot 5",
      "Slot 6",
      "Slot 7",
    ]);
    expect(slots[0]).toHaveTextContent("64.3%");
    expect(slots[0]).toHaveTextContent("02:05");
    expect(screen.queryByText(/risk/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/recommendation/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/history/i)).not.toBeInTheDocument();
  });

  it("links Needs Attention without reordering the matching Portal", () => {
    const snapshot = snapshotAt();
    snapshot.slots[4].portal = portal(55, "Ravnica");
    snapshot.needs_attention_portal_id = 55;
    renderDashboard(snapshot);

    expect(
      screen.getByRole("link", { name: /needs attention.*omenpath #0055/i }),
    ).toHaveAttribute("href", "/portals/55");
    expect(screen.getAllByTestId("portal-slot")[4]).toHaveTextContent(
      "Omenpath #0055",
    );
  });

  it("shows Override deadline and the seven-slot waiting state", () => {
    const snapshot = snapshotAt();
    snapshot.lab.leyline_override_active = true;
    snapshot.lab.leyline_override_until = "2026-09-05T10:00:20Z";
    renderDashboard(snapshot);

    expect(screen.getByRole("status")).toHaveTextContent(
      /leyline override active/i,
    );
    expect(screen.getByRole("status")).toHaveTextContent(/10:00:20/i);
    expect(screen.getByText(/waiting for an omenpath/i)).toBeInTheDocument();
    expect(screen.getAllByTestId("portal-slot")).toHaveLength(7);
    expect(
      screen.getAllByRole("button", {
        name: /stabilize|close|send observer|recall observer/i,
      }),
    ).toHaveLength(28);
  });

  it("rejects malformed Slot identities instead of rendering a partial board", () => {
    const snapshot = snapshotAt();
    snapshot.slots[6] = { slot_index: 6, portal: null };
    renderDashboard(snapshot);

    expect(screen.getByRole("alert")).toHaveTextContent(
      "State snapshot must contain exactly Slot 1..7",
    );
    expect(screen.queryByTestId("portal-board")).not.toBeInTheDocument();
  });

  it("applies later availability and attention snapshots without moving Slots", () => {
    const initial = snapshotAt();
    initial.slots[0].portal = portal(1, "Alara");
    initial.slots[1].portal = portal(2, "Amonkhet");
    initial.portals.active = 2;
    initial.needs_attention_portal_id = 1;
    const { store } = renderDashboard(initial);
    const firstSlot = screen.getAllByTestId("portal-slot")[0];
    expect(within(firstSlot).getByTestId("portal-canvas")).toHaveAttribute(
      "data-density",
      "high",
    );
    expect(
      within(firstSlot).getByRole("button", { name: "Send Observer" }),
    ).toHaveAttribute("aria-disabled", "true");

    const update = snapshotAt("2026-09-05T10:00:01Z");
    update.slots[0].portal = {
      ...portal(1, "Alara"),
      quick_actions: {
        ...portal(1, "Alara").quick_actions,
        can_send_observer: true,
        send_observer_unavailable_reason: null,
      },
    };
    update.portals.active = 1;
    act(() => {
      store.acceptSnapshot(update);
    });

    expect(screen.getAllByTestId("portal-slot")).toHaveLength(7);
    expect(screen.getAllByTestId("portal-slot")[0]).toBe(firstSlot);
    expect(screen.getAllByRole("button")).toHaveLength(28);
    expect(
      within(firstSlot).getByRole("button", { name: "Send Observer" }),
    ).toHaveAttribute("aria-disabled", "false");
    expect(within(firstSlot).getByTestId("portal-canvas")).toHaveAttribute(
      "data-density",
      "low",
    );
    expect(screen.getAllByTestId("portal-slot")[1]).toHaveTextContent(
      "Awaiting Portal",
    );
  });
});
