import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { EventRow } from "./EventRow";

it("shows UTC time then title then details while preserving machine-readable ISO", () => {
  render(
    <EventRow
      event={{
        id: 1,
        event_type: "PORTAL_OPENED",
        created_at: "2026-09-06T15:04:05Z",
        message: "A way opens",
        payload_json: null,
        portal_id: 1,
        observer_id: null,
        plane_id: 1,
      }}
    />,
  );
  const time = screen.getByText("15:04:05-06-09-2026");
  expect(time).toHaveAttribute("datetime", "2026-09-06T15:04:05Z");
  const title = screen.getByText("Portal opened");
  expect(
    time.compareDocumentPosition(title) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(
    title.compareDocumentPosition(screen.getByText("A way opens")) &
      Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
});
