import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { ReactNode } from "react";
import type { OmenpathApi } from "../api/client";
import { FeedbackProvider } from "../components/feedback/FeedbackProvider";
import { ExtractionDialog } from "../features/extraction/ExtractionDialog";
import { TutorialPanel } from "../features/tutorial/TutorialPanel";
import { PortalDetailsPage } from "../pages/PortalDetailsPage";
import { EventLogPage } from "../pages/EventLogPage";
import { AppShell } from "../app/AppShell";
import {
  recordNavigationIntent,
  resetNavigationIntentForTests,
} from "../features/tutorial/navigation-signal";
import { planeDTO, portalDetails, snapshotAt } from "../test/builders";
import { SnapshotProvider } from "./SnapshotProvider";
import { createSnapshotStore } from "./snapshot-store";

beforeEach(() => {
  vi.stubGlobal("WebSocket", undefined);
  resetNavigationIntentForTests();
});
afterEach(() => {
  vi.unstubAllGlobals();
  resetNavigationIntentForTests();
});

async function setup(snapshot = snapshotAt(), path = "/") {
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  const api = {
    state: async () => snapshot,
    portal: vi.fn(async () => portalDetails()),
    events: vi.fn(async () => []),
    tutorialSignal: vi.fn(async () => snapshot),
    openExtraction: vi.fn(async () => snapshot),
    sendObserver: vi.fn(async () => snapshot),
    close: vi.fn(async () => snapshot),
    resetTutorial: vi.fn(async () => snapshot),
    startLive: vi.fn(async () => snapshot),
  } as unknown as OmenpathApi;
  const wrap = (children: ReactNode) => (
    <MemoryRouter initialEntries={[path]}>
      <SnapshotProvider store={store} api={api}>
        <FeedbackProvider>{children}</FeedbackProvider>
      </SnapshotProvider>
    </MemoryRouter>
  );
  const view = render(wrap(null));
  await act(async () => {});
  act(() => store.setConnection("connected"));
  return {
    store,
    api,
    mount: (children: ReactNode) => view.rerender(wrap(children)),
  };
}

it.each(["reconnecting", "protocol", "bootstrap"])(
  "blocks an already-open Extraction dialog after %s failure",
  async (failure) => {
    const snapshot = snapshotAt();
    snapshot.planes = [planeDTO(1)];
    const { store, api, mount } = await setup(snapshot);
    mount(<ExtractionDialog open onClose={() => {}} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Plane 1" }));
    act(() => {
      if (failure === "protocol") store.setProtocolError("Invalid snapshot");
      else if (failure === "bootstrap") store.setBootstrap("failed");
      else store.setConnection("reconnecting");
    });
    const button = screen.getByRole("button", { name: "Open Extraction" });
    expect(button).toBeDisabled();
    await user.click(button);
    expect(api.openExtraction).not.toHaveBeenCalled();
  },
);

it.each([
  [0, "COMPLETE_INTRO", "Begin Practice"],
  [5, "ATTEMPT_CRITICAL_SEND", "Try Send"],
  [9, "START_LIVE", "Start Live"],
] as const)(
  "blocks tutorial step %s CTA and Reset after protocol failure",
  async (step, expected, label) => {
    const snapshot = snapshotAt();
    Object.assign(snapshot.app, {
      tutorial_step: step,
      expected_action: expected,
      tutorial_portal_id: 42,
    });
    const { store, api, mount } = await setup(snapshot);
    mount(<TutorialPanel />);
    act(() => store.setProtocolError("Invalid generated_at in state snapshot"));
    for (const name of [label, "Reset Tutorial"]) {
      const button = screen.getByRole("button", { name });
      expect(button).toBeDisabled();
      await userEvent.setup().click(button);
    }
    expect(api.tutorialSignal).not.toHaveBeenCalled();
    expect(api.sendObserver).not.toHaveBeenCalled();
    expect(api.startLive).not.toHaveBeenCalled();
    expect(api.resetTutorial).not.toHaveBeenCalled();
  },
);

it("rechecks health after Tutorial reset confirmation", async () => {
  const { store, api, mount } = await setup();
  mount(<TutorialPanel />);
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Reset Tutorial" }));
  act(() => store.setConnection("reconnecting"));
  await user.click(
    screen.getByRole("button", { name: "Confirm Reset Tutorial" }),
  );
  expect(api.resetTutorial).not.toHaveBeenCalled();
});

it.each(["portal", "events"] as const)(
  "does not emit %s navigation mutations while unhealthy, then consumes intent on recovery",
  async (kind) => {
    const snapshot = snapshotAt();
    Object.assign(snapshot.app, {
      tutorial_step: kind === "portal" ? 1 : 8,
      expected_action:
        kind === "portal" ? "OPEN_PORTAL_DETAILS" : "OPEN_EVENT_LOG",
      tutorial_portal_id: 42,
    });
    const { store, api, mount } = await setup(
      snapshot,
      kind === "portal" ? "/portals/42" : "/events",
    );
    act(() => store.setProtocolError("Invalid snapshot"));
    recordNavigationIntent(kind === "portal" ? { kind, id: 42 } : { kind });
    mount(
      <Routes>
        <Route path="/portals/:id" element={<PortalDetailsPage />} />
        <Route path="/events" element={<EventLogPage />} />
      </Routes>,
    );
    await act(async () => {});
    expect(api.tutorialSignal).not.toHaveBeenCalled();
    if (kind === "portal") {
      const button = screen.getByRole("button", { name: "Close" });
      expect(button).toHaveAttribute("aria-disabled", "true");
      await userEvent.setup().click(button);
      expect(api.close).not.toHaveBeenCalled();
    }
    act(() => store.setProtocolError(null));
    await act(async () => {});
    expect(api.tutorialSignal).toHaveBeenCalledOnce();
  },
);

it("red protocol-error indicator also blocks the shell extraction entry", async () => {
  const { store, mount } = await setup();
  mount(<AppShell />);
  act(() => store.acceptSnapshot(snapshotAt("invalid-timestamp")));
  expect(
    screen.getByRole("button", { name: "Open Extraction" }),
  ).toBeDisabled();
  expect(screen.getByLabelText("Connection state")).toHaveTextContent(
    "Disconnected from the planes",
  );
});
