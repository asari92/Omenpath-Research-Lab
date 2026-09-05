import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../api/client";
import type { EventDTO, EventType } from "../api/types";
import { SnapshotProvider } from "../state/SnapshotProvider";
import { createSnapshotStore } from "../state/snapshot-store";
import { snapshotAt } from "../test/builders";
import { EventLogPage } from "./EventLogPage";
import {
  recordNavigationIntent,
  resetNavigationIntentForTests,
} from "../features/tutorial/navigation-signal";

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

function setup(events: readonly EventDTO[], snapshot = snapshotAt()) {
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  const eventsRequest = vi.fn(async () => [...events]);
  const tutorialSignal = vi.fn(async () => snapshot);
  const api = {
    state: async () => snapshot,
    events: eventsRequest,
    tutorialSignal,
  } as unknown as OmenpathApi;
  const view = render(
    <MemoryRouter>
      <SnapshotProvider api={api} store={store}>
        <EventLogPage />
      </SnapshotProvider>
    </MemoryRouter>,
  );
  return { ...view, api, store, eventsRequest, tutorialSignal };
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

  it("coalesces realtime edges and aborts the route request on unmount", async () => {
    let resolveFirst!: (events: EventDTO[]) => void;
    const signals: AbortSignal[] = [];
    const eventsRequest = vi
      .fn()
      .mockImplementationOnce((signal: AbortSignal) => {
        signals.push(signal);
        return new Promise<EventDTO[]>((resolve) => {
          resolveFirst = resolve;
        });
      })
      .mockImplementationOnce((signal: AbortSignal) => {
        signals.push(signal);
        return new Promise<EventDTO[]>(() => undefined);
      });
    const snapshot = snapshotAt();
    const store = createSnapshotStore();
    store.acceptSnapshot(snapshot);
    const api = {
      state: async () => snapshot,
      events: eventsRequest,
    } as unknown as OmenpathApi;
    const view = render(
      <MemoryRouter>
        <SnapshotProvider api={api} store={store}>
          <EventLogPage />
        </SnapshotProvider>
      </MemoryRouter>,
    );
    await waitFor(() => expect(eventsRequest).toHaveBeenCalledOnce());
    act(() => {
      store.acceptSnapshot(snapshotAt("2026-09-05T10:00:01Z"));
      store.acceptSnapshot(snapshotAt("2026-09-05T10:00:02Z"));
      store.acceptSnapshot(snapshotAt("2026-09-05T10:00:03Z"));
    });
    resolveFirst([event(1, "PORTAL_OPENED")]);
    await waitFor(() => expect(eventsRequest).toHaveBeenCalledTimes(2));
    view.unmount();
    await Promise.resolve();
    expect(signals.at(-1)?.aborted).toBe(true);
  });

  it("emits one matching Event Log signal and direct loads emit none", async () => {
    resetNavigationIntentForTests();
    const snapshot = snapshotAt();
    snapshot.app.tutorial_step = 8;
    snapshot.app.expected_action = "OPEN_EVENT_LOG";
    recordNavigationIntent({ kind: "events" });
    const first = setup([], snapshot);
    await screen.findByText("No events recorded yet.");
    await waitFor(() => expect(first.tutorialSignal).toHaveBeenCalledOnce());
    first.unmount();
    const direct = setup([], snapshot);
    await screen.findByText("No events recorded yet.");
    expect(direct.tutorialSignal).not.toHaveBeenCalled();
    resetNavigationIntentForTests();
  });

  it("keeps active filters when a realtime edge reveals new events", async () => {
    const snapshot = snapshotAt();
    const first = event(1, "PORTAL_OPENED");
    const matching = { ...event(2, "OBSERVER_RETURNED"), portal_id: 1 };
    const excluded = event(3, "PORTAL_CLOSED");
    const eventsRequest = vi
      .fn()
      .mockResolvedValueOnce([first])
      .mockResolvedValueOnce([first, matching, excluded]);
    const store = createSnapshotStore();
    store.acceptSnapshot(snapshot);
    const api = {
      state: async () => snapshot,
      events: eventsRequest,
    } as unknown as OmenpathApi;
    render(
      <MemoryRouter>
        <SnapshotProvider api={api} store={store}>
          <EventLogPage />
        </SnapshotProvider>
      </MemoryRouter>,
    );
    await screen.findByText("message 1");
    await userEvent.setup().type(screen.getByLabelText("Portal ID"), "1");
    act(() => store.acceptSnapshot(snapshotAt("2026-09-05T10:00:01Z")));
    expect(await screen.findByText("message 2")).toBeVisible();
    expect(screen.queryByText("message 3")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Portal ID")).toHaveValue(1);
  });
});
