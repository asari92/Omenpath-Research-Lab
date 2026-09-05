import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import type { EventDTO } from "../../api/types";
import { EventList } from "./EventList";

function event(id: number, message: string, payload: unknown = null): EventDTO {
  return {
    id,
    event_type: "PORTAL_OPENED",
    portal_id: 42,
    observer_id: null,
    plane_id: 1,
    message,
    payload_json: payload,
    created_at: `2026-09-05T10:00:0${id}Z`,
  };
}

describe("EventList", () => {
  it("keeps API-provided chronological order", () => {
    render(<EventList events={[event(2, "second"), event(1, "first")]} />);
    expect(
      screen.getAllByTestId("event-row").map((row) => row.textContent),
    ).toEqual([
      expect.stringContaining("second"),
      expect.stringContaining("first"),
    ]);
  });

  it("keeps payload collapsed and pretty-prints an object on demand", async () => {
    render(
      <EventList events={[event(1, "opened", { risk: "LOW", count: 2 })]} />,
    );
    const row = screen.getByTestId("event-row");
    const disclosure = within(row).getByText("Payload").closest("details");
    expect(disclosure).not.toHaveAttribute("open");
    expect(within(row).getByText(/"risk"/)).not.toBeVisible();
    await userEvent.setup().click(within(row).getByText("Payload"));
    expect(within(row).getByText(/"risk": "LOW"/)).toBeVisible();
  });
});
