import { act, fireEvent, render, screen } from "@testing-library/react";
import { StrictMode } from "react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";

import type { OmenpathApi } from "../../api/client";
import { FeedbackProvider } from "../../components/feedback/FeedbackProvider";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { TutorialPanel } from "./TutorialPanel";

beforeEach(() => vi.stubGlobal("WebSocket", undefined));
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

function LocationProbe() {
  return <span data-testid="location">{useLocation().pathname}</span>;
}

function setup(
  step: number,
  overrides: Partial<ReturnType<typeof snapshotAt>["app"]> = {},
  path = "/events",
) {
  const snapshot = snapshotAt();
  snapshot.app.tutorial_step = step;
  Object.assign(snapshot.app, overrides);
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  store.setConnection("connected");
  const next = snapshotAt("2026-09-05T10:00:01Z");
  const api = {
    state: async () => snapshot,
    tutorialSignal: vi.fn().mockResolvedValue({
      ...next,
      app: {
        ...next.app,
        tutorial_step: 1,
        expected_action: "OPEN_PORTAL_DETAILS",
      },
    }),
    resetTutorial: vi.fn().mockResolvedValue(next),
    startLive: vi.fn().mockResolvedValue({
      ...next,
      app: { ...next.app, mode: "LIVE" },
    }),
    sendObserver: vi.fn(),
  } as unknown as OmenpathApi;
  render(
    <MemoryRouter initialEntries={[path]}>
      <StrictMode>
        <SnapshotProvider api={api} store={store}>
          <FeedbackProvider>
            <TutorialPanel />
            <LocationProbe />
          </FeedbackProvider>
        </SnapshotProvider>
      </StrictMode>
    </MemoryRouter>,
  );
  return { api, store };
}

it("reuses Forward to leave current Portal Details for Dashboard", async () => {
  setup(
    2,
    { expected_action: "WAIT_CORRIDOR", tutorial_portal_id: 42 },
    "/portals/42",
  );

  const forward = screen.getByRole("button", { name: "Forward" });
  expect(forward).toBeEnabled();
  await userEvent.setup().click(forward);
  expect(screen.getByTestId("location")).toHaveTextContent("/");
});

it("uses Forward for card history before leaving Portal Details", () => {
  const { store } = setup(
    1,
    { expected_action: "OPEN_PORTAL_DETAILS", tutorial_portal_id: 42 },
    "/portals/42",
  );
  const next = snapshotAt("2026-09-05T10:00:02Z");
  next.app.tutorial_step = 2;
  next.app.expected_action = "WAIT_CORRIDOR";
  next.app.tutorial_portal_id = 42;
  act(() => store.acceptSnapshot(next));
  fireEvent.click(screen.getByRole("button", { name: "Back" }));

  fireEvent.click(screen.getByRole("button", { name: "Forward" }));

  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
  expect(screen.getByTestId("location")).toHaveTextContent("/portals/42");
});

it("renders authoritative step, targets, instruction and Step 0 CTA", () => {
  setup(0);
  expect(
    screen.getByRole("complementary", { name: "Tutorial" }),
  ).toHaveTextContent(/Step 0.*Welcome to Omenpath Research Lab.*85\/85/i);
  expect(screen.getByRole("button", { name: "Begin Practice" })).toBeVisible();
});

it("Step 2 waits for authoritative progress and has no action CTA", () => {
  setup(2);
  expect(screen.getByText(/Creatures block SEND/i)).toBeVisible();
  expect(
    screen.queryByRole("button", { name: /begin|try|start live/i }),
  ).not.toBeInTheDocument();
});

it("BEGIN PRACTICE posts the explicit signal once", async () => {
  const { api } = setup(0);
  const button = screen.getByRole("button", { name: "Begin Practice" });
  const user = userEvent.setup();
  await Promise.all([user.click(button), user.click(button)]);
  expect(api.tutorialSignal).toHaveBeenCalledOnce();
  expect(api.tutorialSignal).toHaveBeenCalledWith({
    signal: "TUTORIAL_INTRO_COMPLETED",
  });
});

it("Step 5 TRY SEND uses the ordinary endpoint and treats expected critical rejection as learning", async () => {
  const { api } = setup(5, {
    expected_action: "ATTEMPT_CRITICAL_SEND",
    tutorial_portal_id: 42,
  });
  vi.mocked(api.sendObserver).mockRejectedValue(
    new (await import("../../api/errors")).ApiError(
      409,
      "PORTAL_CRITICAL_RISK",
      false,
      "Portal risk is CRITICAL",
    ),
  );
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Try Send" }));
  expect(api.sendObserver).toHaveBeenCalledOnce();
  expect(api.sendObserver).toHaveBeenCalledWith(42, false);
  expect(screen.queryByText("Portal risk is CRITICAL")).not.toBeInTheDocument();
});

it("Reset confirms once and accepts the authoritative Step 0 snapshot", async () => {
  const { api } = setup(6, {
    expected_action: "WAIT_RESEARCH",
    tutorial_portal_id: 42,
  });
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Reset Tutorial" }));
  expect(screen.getByRole("dialog")).toHaveTextContent(
    /Energy.*Observers.*exploration.*Event history/i,
  );
  await user.click(
    screen.getByRole("button", { name: "Confirm Reset Tutorial" }),
  );
  expect(api.resetTutorial).toHaveBeenCalledOnce();
  expect(
    await screen.findByRole("button", { name: "Begin Practice" }),
  ).toBeVisible();
});

it("START LIVE waits for authoritative LIVE and then returns to Dashboard", async () => {
  const { api } = setup(9, { expected_action: "START_LIVE" });
  const live = snapshotAt("2026-09-05T10:00:02Z");
  live.app.mode = "LIVE";
  let resolveLive: ((value: typeof live) => void) | null = null;
  vi.mocked(api.startLive).mockImplementation(
    () => new Promise((resolve) => (resolveLive = resolve)),
  );
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Start Live" }));
  expect(api.startLive).toHaveBeenCalledOnce();
  expect(screen.getByTestId("location")).toHaveTextContent("/events");
  expect(
    screen.getByRole("complementary", { name: "Tutorial" }),
  ).toBeInTheDocument();
  await act(async () => resolveLive?.(live));
  expect(
    screen.queryByRole("complementary", { name: "Tutorial" }),
  ).not.toBeInTheDocument();
  expect(screen.getByTestId("location")).toHaveTextContent("/");
});

it("shows all context, system action and completion without expandable copy", () => {
  setup(1);
  expect(screen.queryByText("More context")).not.toBeInTheDocument();
  expect(screen.getByText(/Portal Details explains/)).toBeVisible();
  expect(screen.getByText(/System:/)).toBeVisible();
  expect(screen.getByText(/Complete when:/)).toBeVisible();
});

it("replays a skipped Step 2 for exactly seven seconds without delaying the snapshot", () => {
  vi.useFakeTimers();
  const { store, api } = setup(1);
  const next = snapshotAt("2026-09-05T10:00:02Z");
  next.app.tutorial_step = 3;
  act(() => {
    store.acceptSnapshot(next);
  });
  expect(store.getState().snapshot?.app.tutorial_step).toBe(3);
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Completed");
  act(() => {
    vi.advanceTimersByTime(6999);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
  fireEvent.click(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
  fireEvent.click(screen.getByRole("button", { name: "Forward" }));
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
  expect(screen.getByRole("button", { name: "Forward" })).toBeDisabled();
  expect(api.tutorialSignal).not.toHaveBeenCalled();
});

it("normal sequential progress is immediate and reset cancels replay timers and history", () => {
  vi.useFakeTimers();
  const { store } = setup(2);
  const next = snapshotAt("2026-09-05T10:00:02Z");
  next.app.tutorial_step = 3;
  act(() => {
    store.acceptSnapshot(next);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
  const jump = snapshotAt("2026-09-05T10:00:03Z");
  jump.app.tutorial_step = 6;
  act(() => {
    store.acceptSnapshot(jump);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 4");
  act(() => {
    store.acceptSnapshot(snapshotAt("2026-09-05T10:00:04Z"));
  });
  act(() => {
    vi.advanceTimersByTime(30000);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 0");
  expect(screen.getByRole("button", { name: "Back" })).toBeDisabled();
});

it.each([0, 3000])(
  "pauses the required replay after %ims while Back browses history, then Forward resumes its remaining visible time",
  (elapsed) => {
    vi.useFakeTimers();
    const { store, api } = setup(1);
    const next = snapshotAt("2026-09-05T10:00:02Z");
    next.app.tutorial_step = 3;
    act(() => {
      store.acceptSnapshot(next);
    });
    act(() => {
      vi.advanceTimersByTime(elapsed);
    });
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 1");
    act(() => {
      vi.advanceTimersByTime(30000);
    });
    expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 1");
    fireEvent.click(screen.getByRole("button", { name: "Forward" }));
    expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
    act(() => {
      vi.advanceTimersByTime(6999 - elapsed);
    });
    expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
    expect(api.tutorialSignal).not.toHaveBeenCalled();
  },
);

it("does not restart a replay deadline on snapshots and gives each newly missed card its own seven seconds", () => {
  vi.useFakeTimers();
  const { store } = setup(1);
  const at = (second: number, step: number) => {
    const value = snapshotAt(`2026-09-05T10:00:0${second}Z`);
    value.app.tutorial_step = step;
    return value;
  };
  act(() => {
    store.acceptSnapshot(at(1, 3));
  });
  act(() => {
    vi.advanceTimersByTime(3000);
    store.acceptSnapshot(at(2, 4));
  });
  act(() => {
    vi.advanceTimersByTime(3999);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 2");
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Completed");
  act(() => {
    vi.advanceTimersByTime(6999);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 3");
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(screen.getByLabelText("Tutorial")).toHaveTextContent("Step 4");
});
