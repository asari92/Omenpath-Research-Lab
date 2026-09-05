import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ConfirmDialog } from "./ConfirmDialog";

describe("ConfirmDialog", () => {
  it("identifies action and Portal and returns focus after cancel", async () => {
    const cancel = vi.fn();
    const opener = document.createElement("button");
    opener.textContent = "Close";
    document.body.append(opener);
    opener.focus();
    const view = render(
      <ConfirmDialog
        action="Close"
        message="This may lose an Observer"
        onCancel={cancel}
        onConfirm={vi.fn()}
        open
        opener={opener}
        portalName="Omenpath #0042"
      />,
    );
    expect(screen.getByRole("dialog")).toHaveTextContent(
      /Close.*Omenpath #0042/i,
    );
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Cancel" }));
    expect(cancel).toHaveBeenCalledOnce();
    view.rerender(
      <ConfirmDialog
        action="Close"
        message=""
        onCancel={cancel}
        onConfirm={vi.fn()}
        open={false}
        opener={opener}
        portalName="Omenpath #0042"
      />,
    );
    expect(opener).toHaveFocus();
    opener.remove();
  });
});
