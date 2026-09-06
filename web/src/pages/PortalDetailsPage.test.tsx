import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../api/client";
import { ApiError } from "../api/errors";
import type { PortalDetails, StateSnapshot } from "../api/types";
import { SnapshotProvider } from "../state/SnapshotProvider";
import { createSnapshotStore } from "../state/snapshot-store";
import { observerTransit, portalDetails, snapshotAt } from "../test/builders";
import {
  recordNavigationIntent,
  resetNavigationIntentForTests,
} from "../features/tutorial/navigation-signal";
import { PortalDetailsPage } from "./PortalDetailsPage";

beforeEach(() => vi.stubGlobal("WebSocket", undefined));
afterEach(() => {
  resetNavigationIntentForTests();
  vi.unstubAllGlobals();
});

function LocationProbe() {
  return <span data-testid="location">{useLocation().pathname}</span>;
}

function apiFor(
  result: PortalDetails | Error,
  snapshot: StateSnapshot = snapshotAt(),
): OmenpathApi {
  const portal = vi.fn(async () => {
    if (result instanceof Error) throw result;
    return result;
  });
  return {
    state: async () => snapshot,
    portal,
    events: async () => [],
    stabilize: async () => snapshot,
    close: async () => snapshot,
    sendObserver: async () => snapshot,
    recallObserver: async () => snapshot,
    openExtraction: async () => snapshot,
    startTutorial: async () => snapshot,
    resetTutorial: async () => snapshot,
    tutorialSignal: vi.fn(async () => snapshot),
    startLive: async () => snapshot,
  };
}

function renderDetails(
  path: string,
  api: OmenpathApi,
  initialSnapshot: StateSnapshot = snapshotAt(),
) {
  const store = createSnapshotStore();
  store.acceptSnapshot(initialSnapshot);
  store.setConnection("connected");
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SnapshotProvider api={api} store={store}>
        <Routes>
          <Route path="/portals/:id" element={<PortalDetailsPage />} />
          <Route path="/" element={<p>Dashboard route</p>} />
        </Routes>
        <LocationProbe />
      </SnapshotProvider>
    </MemoryRouter>,
  );
}

describe("PortalDetailsPage", () => {
  it("shows authoritative transit and collapses History with its count", async () => {
    const details = portalDetails();
    details.observer_transit = observerTransit(42, 7, "RETURNING");
    renderDetails("/portals/42", apiFor(details));
    expect(await screen.findByLabelText("Observer transit")).toHaveTextContent(
      "7",
    );
    expect(screen.getByLabelText("Observer transit")).toHaveTextContent(
      /Returning.*00:05/i,
    );
    const history = screen.getByText("History (0)").closest("details");
    expect(history).not.toHaveAttribute("open");
    expect(screen.queryByText(/Refreshing/)).not.toBeInTheDocument();
  });
  it.each(["CLOSED", "COLLAPSED"] as const)(
    "passes terminal %s into the entire portal visual",
    async (status) => {
      const details = portalDetails();
      details.portal.status = status;
      renderDetails("/portals/42", apiFor(details));
      await screen.findByText(status);
      expect(
        screen.getByRole("img", { name: "Agyrem" }).parentElement,
      ).toHaveAttribute("data-status", status);
    },
  );
  it.each(["abc", "0", "-1", "1.5"])(
    "rejects non-positive-integer route id %s before an API call",
    async (id) => {
      const api = apiFor(portalDetails());
      renderDetails(`/portals/${id}`, api);
      expect(screen.getByRole("alert")).toHaveTextContent("Invalid Portal ID");
      expect(api.portal).not.toHaveBeenCalled();
    },
  );

  it("renders every required Portal, Destination and Diagnostics field", async () => {
    const details = portalDetails(42);
    details.portal.energy = 64.26;
    details.portal.time_remaining_seconds = 125;
    details.portal.creatures_inside = 3;
    details.portal.observer_flow = "OUTBOUND";
    details.destination = {
      ...details.destination,
      name: "Agyrem",
      explored: false,
      observers_exploring: 2,
      observers_waiting_return: 1,
      previous_connection_count: 4,
    };
    details.risk_level = "HIGH";
    details.recommendation = "STABILIZE";
    renderDetails("/portals/42", apiFor(details));

    expect(
      await screen.findByRole("heading", { name: "Portal" }),
    ).toBeVisible();
    const text = document.body.textContent ?? "";
    for (const value of [
      "Portal 42",
      "OPEN",
      "64.3%",
      "STABLE",
      "02:05",
      "3",
      "OUTBOUND",
      "Agyrem",
      "UNEXPLORED",
      "2",
      "1",
      "4",
      "HIGH",
      "STABILIZE",
    ]) {
      expect(text).toContain(value);
    }
    expect(screen.getByText("HIGH")).toHaveAttribute("data-risk", "HIGH");
    expect(screen.getByText("STABILIZE")).toHaveAttribute(
      "data-recommendation",
      "STABILIZE",
    );
    expect(screen.getByText("64.3%")).toHaveAttribute(
      "data-value-kind",
      "energy",
    );
    expect(screen.getByText("02:05")).toHaveAttribute(
      "data-value-kind",
      "time",
    );
    expect(screen.getByText("OPEN")).toHaveAttribute("data-status", "OPEN");
    expect(screen.getByText("STABLE")).toHaveAttribute(
      "data-stability",
      "STABLE",
    );
    const creatures = screen.getByText("Creatures");
    expect(creatures.closest("section")).toContainElement(
      screen.getByRole("heading", { name: "Destination" }),
    );
  });

  it("renders terminal diagnostics as Not applicable", async () => {
    const details = portalDetails();
    details.portal.status = "CLOSED";
    details.risk_level = null;
    details.recommendation = null;
    renderDetails("/portals/42", apiFor(details));
    await screen.findByText("CLOSED");
    expect(screen.getAllByText("Not applicable")).toHaveLength(2);
  });

  it("explains risk bands without leaking calculations", async () => {
    renderDetails("/portals/42", apiFor(portalDetails()));
    const disclosure = await screen.findByText("How Risk Works");
    await userEvent.setup().click(disclosure);
    const explanation = screen.getByTestId("risk-explanation");
    expect(explanation).toHaveTextContent(/LOW.*MEDIUM.*HIGH.*CRITICAL/i);
    expect(explanation).toHaveTextContent(/guidance, not a restriction/i);
    expect(explanation).not.toHaveTextContent(
      /risk_score|decay|lifetime|timestamp/i,
    );
  });

  it("renders 404 separately and retries transient errors", async () => {
    const missing = apiFor(
      new ApiError(404, "PORTAL_NOT_FOUND", false, "missing"),
    );
    const first = renderDetails("/portals/42", missing);
    expect(await screen.findByText("Portal Not Found")).toBeVisible();
    first.unmount();

    const api = apiFor(new Error("network unavailable"));
    renderDetails("/portals/42", api);
    expect(
      await screen.findByText("Unable to load Portal Details."),
    ).toBeVisible();
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Retry" }));
    expect(api.portal).toHaveBeenCalledTimes(2);
  });

  it("renders API-provided History in order and the same four actions", async () => {
    const details = portalDetails();
    details.history = [
      {
        id: 2,
        event_type: "PORTAL_STABILIZED",
        portal_id: 42,
        observer_id: null,
        plane_id: 1,
        message: "second",
        payload_json: {},
        created_at: "2026-09-05T10:00:02Z",
      },
      {
        id: 1,
        event_type: "PORTAL_OPENED",
        portal_id: 42,
        observer_id: null,
        plane_id: 1,
        message: "first",
        payload_json: {},
        created_at: "2026-09-05T10:00:01Z",
      },
    ];
    renderDetails("/portals/42", apiFor(details));
    expect(await screen.findByText("History (2)")).toBeVisible();
    await userEvent.setup().click(screen.getByText("History (2)"));
    expect(
      screen.getAllByTestId("event-row").map((row) => row.textContent),
    ).toEqual([
      expect.stringContaining("second"),
      expect.stringContaining("first"),
    ]);
    expect(
      screen.getAllByRole("button", {
        name: /stabilize|close|send observer|recall observer/i,
      }),
    ).toHaveLength(4);
  });

  it("emits one matching signal from explicit intent and none on direct load", async () => {
    const snapshot = snapshotAt();
    snapshot.app.expected_action = "OPEN_PORTAL_DETAILS";
    snapshot.app.tutorial_portal_id = 42;
    const api = apiFor(portalDetails(), snapshot);
    const next = snapshotAt("2026-09-05T10:00:01Z");
    next.app.tutorial_step = 2;
    next.app.expected_action = "WAIT_CORRIDOR";
    vi.mocked(api.tutorialSignal).mockResolvedValue(next);
    recordNavigationIntent({ kind: "portal", id: 42 });
    renderDetails("/portals/42", api, snapshot);
    expect(await screen.findByText("Dashboard route")).toBeVisible();
    expect(api.tutorialSignal).toHaveBeenCalledOnce();
    expect(api.tutorialSignal).toHaveBeenCalledWith({
      signal: "PORTAL_DETAILS_OPENED",
      portal_id: 42,
    });
    expect(screen.getByTestId("location")).toHaveTextContent("/");
  });

  it("stays on Details when the matching Tutorial signal fails", async () => {
    const snapshot = snapshotAt();
    snapshot.app.expected_action = "OPEN_PORTAL_DETAILS";
    snapshot.app.tutorial_portal_id = 42;
    const api = apiFor(portalDetails(), snapshot);
    vi.mocked(api.tutorialSignal).mockRejectedValue(new Error("signal failed"));
    recordNavigationIntent({ kind: "portal", id: 42 });

    renderDetails("/portals/42", api, snapshot);

    expect(await screen.findByText("signal failed")).toBeVisible();
    expect(screen.getByTestId("location")).toHaveTextContent("/portals/42");
  });
});
