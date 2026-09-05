import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { FeedbackProvider } from "../../components/feedback/FeedbackProvider";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { TutorialPanel } from "./TutorialPanel";

function setup(step: number) {
  const snapshot = snapshotAt();
  snapshot.app.tutorial_step = step;
  const store = createSnapshotStore();
  store.acceptSnapshot(snapshot);
  const api = {
    state: async () => snapshot,
    tutorialSignal: vi.fn(),
    resetTutorial: vi.fn(),
    startLive: vi.fn(),
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
