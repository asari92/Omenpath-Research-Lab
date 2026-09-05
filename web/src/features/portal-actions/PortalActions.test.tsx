import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { QuickActionsDTO } from "../../api/types";
import { PortalActions } from "./PortalActions";

const actions: QuickActionsDTO = {
  can_stabilize: true,
  stabilize_unavailable_reason: null,
  can_close: true,
  close_unavailable_reason: null,
  can_send_observer: false,
  send_observer_unavailable_reason: "PORTAL_CRITICAL_RISK",
  can_recall_observer: false,
  recall_observer_unavailable_reason: "NO_WAITING_OBSERVER",
};

describe("PortalActions", () => {
  it("always renders exactly four gameplay commands in stable order", () => {
    render(<PortalActions portalId={42} quickActions={actions} />);
    expect(
      screen.getAllByRole("button").map((button) => button.textContent),
    ).toEqual(["Stabilize", "Close", "Send Observer", "Recall Observer"]);
  });

  it("available action invokes only its exact command", async () => {
    const user = userEvent.setup();
    const onCommand = vi.fn();
    render(
      <PortalActions
        onCommand={onCommand}
        portalId={42}
        quickActions={actions}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Stabilize" }));
    expect(onCommand).toHaveBeenCalledOnce();
    expect(onCommand).toHaveBeenCalledWith("STABILIZE");
  });

  it("unavailable action stays focusable, explains why and does not dispatch", async () => {
    const user = userEvent.setup();
    const onCommand = vi.fn();
    render(
      <PortalActions
        onCommand={onCommand}
        portalId={42}
        quickActions={actions}
      />,
    );
    const send = screen.getByRole("button", { name: "Send Observer" });
    expect(send).toHaveAttribute("aria-disabled", "true");
    send.focus();
    expect(send).toHaveFocus();
    await user.click(send);
    expect(screen.getByRole("status")).toHaveTextContent(/critical risk/i);
    expect(onCommand).not.toHaveBeenCalled();
  });

  it("native-disables only the selected in-flight action", () => {
    render(
      <PortalActions
        busyKey="42:CLOSE"
        portalId={42}
        quickActions={{
          ...actions,
          can_send_observer: true,
          send_observer_unavailable_reason: null,
        }}
      />,
    );
    expect(screen.getByRole("button", { name: "Close" })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Send Observer" }),
    ).not.toBeDisabled();
  });

  it("Empty Slot uses the same four controls with a local reason", async () => {
    const user = userEvent.setup();
    render(<PortalActions portalId={null} quickActions={null} />);
    expect(screen.getAllByRole("button")).toHaveLength(4);
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.getByRole("status")).toHaveTextContent(
      /no portal occupies this slot/i,
    );
  });

  it("allows only the explicit tutorial critical SEND override", async () => {
    const user = userEvent.setup();
    const onCommand = vi.fn();
    const { rerender } = render(
      <PortalActions
        expectedCriticalSend
        onCommand={onCommand}
        portalId={42}
        quickActions={actions}
      />,
    );
    const send = screen.getByRole("button", { name: "Send Observer" });
    expect(send).toHaveAttribute("aria-disabled", "false");
    await user.click(send);
    expect(onCommand).toHaveBeenCalledWith("SEND");

    onCommand.mockClear();
    rerender(
      <PortalActions
        expectedCriticalSend
        onCommand={onCommand}
        portalId={42}
        quickActions={{
          ...actions,
          send_observer_unavailable_reason: "PORTAL_BUSY",
        }}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Send Observer" }));
    expect(onCommand).not.toHaveBeenCalled();
  });

  it("marks only the authoritative target command", () => {
    render(
      <PortalActions
        highlightedCommand="STABILIZE"
        portalId={42}
        quickActions={actions}
      />,
    );
    expect(screen.getByRole("button", { name: "Stabilize" })).toHaveAttribute(
      "data-tutorial-command",
      "true",
    );
    expect(
      screen.getByRole("button", { name: "Send Observer" }),
    ).not.toHaveAttribute("data-tutorial-command");
  });
});
