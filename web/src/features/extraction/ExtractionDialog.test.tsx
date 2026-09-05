import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { ApiError } from "../../api/errors";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { planeDTO, snapshotAt } from "../../test/builders";
import { ExtractionDialog } from "./ExtractionDialog";

function setup(result = snapshotAt("2026-09-05T10:00:01Z")) {
  const initial = snapshotAt();
  initial.planes = Array.from({ length: 85 }, (_, index) =>
    planeDTO(index + 1),
  );
  const store = createSnapshotStore();
  store.acceptSnapshot(initial);
  const openExtraction = vi.fn(async () => {
    if (result instanceof Error) throw result;
    return result;
  });
  const api = {
    state: async () => initial,
    portal: vi.fn(),
    events: async () => [],
    stabilize: vi.fn(),
    close: vi.fn(),
    sendObserver: vi.fn(),
    recallObserver: vi.fn(),
    openExtraction,
    startTutorial: vi.fn(),
    resetTutorial: vi.fn(),
    tutorialSignal: vi.fn(),
    startLive: vi.fn(),
  } as unknown as OmenpathApi;
  const onClose = vi.fn();
  render(
    <SnapshotProvider api={api} store={store}>
      <ExtractionDialog onClose={onClose} open />
    </SnapshotProvider>,
  );
  return { onClose, openExtraction, store };
}

describe("ExtractionDialog", () => {
  it("renders all 85 cards with local art and authoritative badges/counts", () => {
    setup();
    const cards = screen.getAllByTestId("plane-card");
    expect(cards).toHaveLength(85);
    for (const image of screen.getAllByRole("img")) {
      expect(image.getAttribute("src")).toMatch(/^\/planes\//);
    }
    expect(cards[0]).toHaveTextContent(/UNEXPLORED.*In Plane 0.*Waiting 0/i);
  });

  it("filters with one active mode and shows exact 30 Energy selection cost", async () => {
    setup();
    const user = userEvent.setup();
    await user.type(screen.getByRole("searchbox"), "Plane 3");
    expect(screen.getAllByTestId("plane-card")).toHaveLength(11);
    await user.click(screen.getByRole("radio", { name: "Observer Present" }));
    expect(screen.queryAllByTestId("plane-card")).toHaveLength(0);
    await user.click(screen.getByRole("radio", { name: "All" }));
    await user.click(screen.getByRole("button", { name: "Plane 3" }));
    expect(screen.getByTestId("selection-summary")).toHaveTextContent(
      /cost.*30 Lab Energy/i,
    );
  });

  it("submits one positive plane_id, accepts snapshot and closes", async () => {
    const { onClose, openExtraction, store } = setup();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Plane 1" }));
    await user.click(screen.getByRole("button", { name: "Open Extraction" }));
    expect(openExtraction).toHaveBeenCalledOnce();
    expect(openExtraction).toHaveBeenCalledWith(1);
    expect(store.getState().snapshot?.generated_at).toBe(
      "2026-09-05T10:00:01Z",
    );
    expect(onClose).toHaveBeenCalledOnce();
  });

  it.each([
    new ApiError(409, "INSUFFICIENT_LAB_ENERGY", false, "need more energy"),
    new ApiError(404, "PLANE_NOT_FOUND", false, "missing"),
    new Error("network unavailable"),
  ])("preserves selection and shows feedback after %s", async (error) => {
    setup(error as never);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Plane 2" }));
    await user.click(screen.getByRole("button", { name: "Open Extraction" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(error.message);
    expect(screen.getByRole("button", { name: "Plane 2" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("traps keyboard focus and Escape cancels without POST", async () => {
    const { onClose, openExtraction } = setup();
    const user = userEvent.setup();
    const dialog = screen.getByRole("dialog");
    const buttons = within(dialog).getAllByRole("button");
    expect(dialog).toContainElement(document.activeElement as HTMLElement);
    buttons.at(-1)?.focus();
    await user.keyboard("{Tab}");
    expect(dialog).toContainElement(document.activeElement as HTMLElement);
    await user.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledOnce();
    expect(openExtraction).not.toHaveBeenCalled();
  });
});
