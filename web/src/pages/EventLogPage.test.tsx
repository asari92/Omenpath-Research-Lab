import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../api/client";
import type { EventDTO, EventType } from "../api/types";
import { SnapshotProvider } from "../state/SnapshotProvider";
import { createSnapshotStore } from "../state/snapshot-store";
import { snapshotAt } from "../test/builders";
import { EventLogPage } from "./EventLogPage";

function event(id: number, type: EventType): EventDTO {
  return {
    id,
    event_type: type,
    portal_id: id,
    observer_id: null,
    plane_id: 1,
    message: `message ${id}`,
    payload_json: { html: "<img onerror=alert(1)>" },
    created_at: `2026-09-05T10:00:0${id}Z`,
  };
}

function setup(events: readonly EventDTO[]) {
  const snapshot = snapshotAt();
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  const api = {
    state: async () => snapshot,
    events: vi.fn(async () => [...events]),
  } as unknown as OmenpathApi;
  const view = render(
    <MemoryRouter>
      <SnapshotProvider api={api} store={store}>
        <EventLogPage />
      </SnapshotProvider>
    </MemoryRouter>,
  );
  return { ...view, api, store };
}

describe("EventLogPage", () => {
  it("renders the complete API source in received order and payload as text", async () => {
    setup([event(2, "OBSERVER_RETURNED"), event(1, "PORTAL_OPENED")]);
    const rows = await screen.findAllByTestId("event-row");
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining("message 2"),
      expect.stringContaining("message 1"),
    ]);
    await userEvent.setup().click(screen.getAllByText("Payload")[0]);
    expect(within(rows[0]).getByText(/<img onerror=alert/)).toBeVisible();
    expect(document.querySelector("img[onerror]")).toBeNull();
  });

  it("distinguishes an empty source from an empty filter result", async () => {
    const emptyView = setup([]);
    expect(await screen.findByText("No events recorded yet.")).toBeVisible();
    emptyView.unmount();
    setup([event(1, "PORTAL_OPENED")]);
    await screen.findByTestId("event-row");
    await userEvent.setup().type(screen.getByLabelText("Portal ID"), "999");
    expect(
      screen.getByText("No events match the active filters."),
    ).toBeVisible();
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getByTestId("event-row")).toHaveTextContent("message 1");
  });
});
