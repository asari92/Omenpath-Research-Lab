import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { FeedbackProvider } from "../../components/feedback/FeedbackProvider";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { TutorialPanel } from "./TutorialPanel";

beforeEach(() => vi.stubGlobal("WebSocket", undefined));
afterEach(() => vi.unstubAllGlobals());

function setup(
  step: number,
  overrides: Partial<ReturnType<typeof snapshotAt>["app"]> = {},
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
    <SnapshotProvider api={api} store={store}>
      <FeedbackProvider>
        <TutorialPanel />
      </FeedbackProvider>
    </SnapshotProvider>,
  );
  return { api, store };
}

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

it("START LIVE removes the panel only after an authoritative LIVE snapshot", async () => {
  const { api } = setup(9, { expected_action: "START_LIVE" });
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Start Live" }));
  expect(api.startLive).toHaveBeenCalledOnce();
  expect(
    screen.queryByRole("complementary", { name: "Tutorial" }),
  ).not.toBeInTheDocument();
});
