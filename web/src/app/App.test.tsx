import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { App } from "./App";
import { AppShell } from "./AppShell";
import { MemoryRouter } from "react-router-dom";
import { SnapshotProvider } from "../state/SnapshotProvider";
import { createSnapshotStore } from "../state/snapshot-store";
import { snapshotAt } from "../test/builders";
import type { OmenpathApi } from "../api/client";

describe("application routes", () => {
  it("applies Override to the whole shell and keeps twenty roster pips", async () => {
    const snapshot = snapshotAt();
    snapshot.lab.leyline_override_active = true;
    snapshot.observers.in_lab = 17;
    const store = createSnapshotStore();
    store.acceptSnapshot(snapshot);
    const api = { state: async () => snapshot } as OmenpathApi;
    render(
      <MemoryRouter>
        <SnapshotProvider store={store} api={api}>
          <AppShell />
        </SnapshotProvider>
      </MemoryRouter>,
    );
    expect(screen.getByTestId("app-shell")).toHaveAttribute(
      "data-override",
      "true",
    );
    expect(screen.getByTestId("app-shell")).toHaveAttribute(
      "data-visual-theme",
      "arcane-laboratory",
    );
    expect(screen.getByTestId("lab-energy-gauge")).toHaveTextContent(
      `${snapshot.lab.current_energy} / ${snapshot.lab.maximum_energy}`,
    );
    expect(screen.getAllByTestId("observer-pip")).toHaveLength(20);
    expect(
      screen.getByRole("img", { name: "17 of 20 Observers in Lab" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Leyline Override")).toBeInTheDocument();
    expect(screen.getByText(/0 \/ 85/)).toBeInTheDocument();
    expect(screen.getByText(/Active 0 \/ 7/)).toBeInTheDocument();
    await act(async () => {});
    act(() => store.setConnection("reconnecting"));
    expect(
      screen.getByRole("button", { name: "Open Extraction" }),
    ).toBeDisabled();
  });
  it("renders the shared shell and dashboard at /", async () => {
    render(<App initialEntries={["/"]} />);

    expect(await screen.findByRole("banner")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /laboratory overview/i }),
    ).toBeInTheDocument();
  });

  it.each([
    ["/events", /event log/i],
    ["/ai-worklog", /ai worklog/i],
    ["/unknown", /not found/i],
  ])("maps %s to its page boundary", async (path, heading) => {
    render(<App initialEntries={[path]} />);

    expect(
      await screen.findByRole("heading", { name: heading, level: 1 }),
    ).toBeInTheDocument();
  });

  it("opens artwork credits from the shared shell", async () => {
    const user = userEvent.setup();
    render(<App initialEntries={["/"]} />);

    await user.click(screen.getByRole("button", { name: /artwork credits/i }));
    expect(
      screen.getByRole("dialog", { name: /plane artwork credits/i }),
    ).toBeInTheDocument();
  });

  it("exposes keyboard-reachable primary routes in stable order", async () => {
    const user = userEvent.setup();
    render(<App initialEntries={["/"]} />);

    const nav = screen.getByRole("navigation", { name: "Primary navigation" });
    expect(
      Array.from(nav.querySelectorAll("a, button")).map(
        (link) => link.textContent,
      ),
    ).toEqual([
      "Dashboard",
      "Event Log",
      "Help",
      "Open Extraction",
      "Artwork Credits",
      "AI Worklog",
    ]);
    screen.getByRole("link", { name: "Dashboard" }).focus();
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("link", { name: "Event Log" })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("link", { name: "Help" })).toHaveFocus();
  });
});
