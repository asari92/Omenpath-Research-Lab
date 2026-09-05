import type { EventDTO, EventType } from "../../api/types";

export interface EventFilter {
  eventTypes: ReadonlySet<EventType>;
  portalId: number | null;
  observerId: number | null;
  planeId: number | null;
}

export const eventTypes: readonly EventType[] = [
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

const labels: Record<EventType, string> = {
  PORTAL_OPENED: "Portal opened",
  PORTAL_STABILIZED: "Portal stabilized",
  PORTAL_CLOSED: "Portal closed",
  PORTAL_COLLAPSED: "Portal collapsed",
  RISK_LEVEL_CHANGED: "Risk level changed",
  OBSERVER_DISPATCHED: "Observer dispatched",
  OBSERVER_ARRIVED: "Observer arrived",
  RESEARCH_STARTED: "Research started",
  RESEARCH_COMPLETED: "Research completed",
  OBSERVER_RETURN_STARTED: "Observer return started",
  OBSERVER_RETURNED: "Observer returned",
  OBSERVER_LOST: "Observer lost",
  PLANE_EXPLORED: "Plane explored",
  EXTRACTION_PORTAL_OPENED: "Extraction Portal opened",
  EXTRACTION_SYNCHRONIZED: "Extraction synchronized",
  LEYLINE_OVERRIDE_STARTED: "Leyline Override started",
  LEYLINE_OVERRIDE_ENDED: "Leyline Override ended",
  ACTION_REJECTED: "Action rejected",
};

export function eventTypeLabel(type: EventType): string {
  return labels[type];
}

export function filterEvents(
  events: readonly EventDTO[],
  filter: EventFilter,
): readonly EventDTO[] {
  if (
    filter.eventTypes.size === 0 &&
    filter.portalId === null &&
    filter.observerId === null &&
    filter.planeId === null
  )
    return events;
  return events.filter(
    (event) =>
      (filter.eventTypes.size === 0 ||
        filter.eventTypes.has(event.event_type)) &&
      (filter.portalId === null || event.portal_id === filter.portalId) &&
      (filter.observerId === null || event.observer_id === filter.observerId) &&
      (filter.planeId === null || event.plane_id === filter.planeId),
  );
}
