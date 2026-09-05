import type { EventDTO } from "../../api/types";
import { EventRow } from "./EventRow";

export function EventList({
  events,
  emptyMessage = "No events recorded.",
}: {
  events: readonly EventDTO[];
  emptyMessage?: string;
}) {
  if (events.length === 0) return <p>{emptyMessage}</p>;
  return (
    <div>
      {events.map((event) => (
        <EventRow event={event} key={event.id} />
      ))}
    </div>
  );
}
