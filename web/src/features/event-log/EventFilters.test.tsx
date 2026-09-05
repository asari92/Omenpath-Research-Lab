import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import { EventFilters } from "./EventFilters";

it("builds multi-type and exact positive ID filters then clears them", async () => {
  const onChange = vi.fn();
  const user = userEvent.setup();
  render(<EventFilters onChange={onChange} />);
  await user.click(screen.getByRole("checkbox", { name: "Portal opened" }));
  await user.click(screen.getByRole("checkbox", { name: "Observer returned" }));
  await user.type(screen.getByLabelText("Portal ID"), "42");
  expect(onChange).toHaveBeenLastCalledWith(
    expect.objectContaining({
      portalId: 42,
      eventTypes: new Set(["PORTAL_OPENED", "OBSERVER_RETURNED"]),
    }),
  );
  await user.click(screen.getByRole("button", { name: "Clear filters" }));
  expect(onChange).toHaveBeenLastCalledWith({
    eventTypes: new Set(),
    portalId: null,
    observerId: null,
    planeId: null,
  });
});
