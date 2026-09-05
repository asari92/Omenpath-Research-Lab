import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import type { OmenpathApi } from "../../api/client";
import { SnapshotProvider } from "../../state/SnapshotProvider";
import { createSnapshotStore } from "../../state/snapshot-store";
import { snapshotAt } from "../../test/builders";
import { ConnectionState } from "./ConnectionState";

it("announces connection changes and retries without dropping the last snapshot", async () => {
  const store = createSnapshotStore();
  const old = snapshotAt();
  store.acceptSnapshot(old);
  const next = snapshotAt("2026-09-05T10:00:01Z");
  const state = vi
    .fn()
    .mockRejectedValueOnce(new Error("offline"))
    .mockResolvedValueOnce(next);
  const api = { state } as unknown as OmenpathApi;
  render(
    <SnapshotProvider api={api} store={store}>
      <ConnectionState />
    </SnapshotProvider>,
  );
  await screen.findByRole("alert");
  expect(store.getState().snapshot).toBe(old);
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Retry connection" }));
  expect(store.getState().snapshot).toBe(next);
  act(() => store.setConnection("reconnecting"));
  expect(screen.getByRole("status")).toHaveTextContent("Planar paths unstable");
  act(() => store.setConnection("connected"));
  expect(screen.getByRole("status")).toHaveTextContent("Planar link stable");
  expect(screen.queryByText(/Connection:/)).not.toBeInTheDocument();
  act(() => store.setConnection("offline"));
  expect(screen.getByRole("status")).toHaveTextContent(
    "Disconnected from the planes",
  );
});
