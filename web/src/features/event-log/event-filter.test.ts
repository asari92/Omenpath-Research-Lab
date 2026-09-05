import { describe, expect, it } from "vitest";

import type { EventDTO, EventType } from "../../api/types";
import { eventTypeLabel, filterEvents, type EventFilter } from "./event-filter";

function event(
  id: number,
  eventType: EventType,
  portalId: number | null,
  observerId: number | null,
  planeId: number | null,
): EventDTO {
  return {
    id,
    event_type: eventType,
    portal_id: portalId,
    observer_id: observerId,
    plane_id: planeId,
    message: `event ${id}`,
    payload_json: {},
    created_at: `2026-09-05T10:00:0${id}Z`,
  };
}

const source = [
  event(3, "OBSERVER_RETURNED", 2, 7, 4),
  event(2, "PORTAL_OPENED", 2, null, 4),
  event(1, "ACTION_REJECTED", 1, 7, 3),
];
const empty: EventFilter = {
  eventTypes: new Set(),
  portalId: null,
  observerId: null,
  planeId: null,
};

describe("filterEvents", () => {
  it("supports one or many event types and preserves received order", () => {
    expect(
      filterEvents(source, {
        ...empty,
        eventTypes: new Set(["PORTAL_OPENED"]),
      }).map((item) => item.id),
    ).toEqual([2]);
    expect(
      filterEvents(source, {
        ...empty,
        eventTypes: new Set(["OBSERVER_RETURNED", "ACTION_REJECTED"]),
      }).map((item) => item.id),
    ).toEqual([3, 1]);
  });

  it("combines exact positive entity IDs with logical AND", () => {
    expect(
      filterEvents(source, { ...empty, portalId: 2, planeId: 4 }).map(
        (item) => item.id,
      ),
    ).toEqual([3, 2]);
    expect(
      filterEvents(source, {
        ...empty,
        portalId: 2,
        observerId: 7,
        planeId: 4,
      }).map((item) => item.id),
    ).toEqual([3]);
    expect(filterEvents(source, empty)).toBe(source);
  });

  it("provides stable human labels for every canonical event type", () => {
    const types: EventType[] = [
      "PORTAL_OPENED",
      "PORTAL_STABILIZED",
      "PORTAL_CLOSED",
      "PORTAL_COLLAPSED",
      "RISK_LEVEL_CHANGED",
      "OBSERVER_DISPATCHED",
      "OBSERVER_ARRIVED",
      "RESEARCH_STARTED",
      "RESEARCH_COMPLETED",
      "OBSERVER_RETURN_STARTED",
      "OBSERVER_RETURNED",
      "OBSERVER_LOST",
      "PLANE_EXPLORED",
      "EXTRACTION_PORTAL_OPENED",
      "EXTRACTION_SYNCHRONIZED",
      "LEYLINE_OVERRIDE_STARTED",
      "LEYLINE_OVERRIDE_ENDED",
      "ACTION_REJECTED",
    ];
    expect(types.map(eventTypeLabel)).toEqual([
      "Portal opened",
      "Portal stabilized",
      "Portal closed",
      "Portal collapsed",
      "Risk level changed",
      "Observer dispatched",
      "Observer arrived",
      "Research started",
      "Research completed",
      "Observer return started",
      "Observer returned",
      "Observer lost",
      "Plane explored",
      "Extraction Portal opened",
      "Extraction synchronized",
      "Leyline Override started",
      "Leyline Override ended",
      "Action rejected",
    ]);
  });
});
