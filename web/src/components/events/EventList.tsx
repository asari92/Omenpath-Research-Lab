import type { EventDTO } from "../../api/types";
import { EventRow } from "./EventRow";

export function EventList({ events }: { events: readonly EventDTO[] }) {
  if (events.length === 0) return <p>No events recorded.</p>;
  return (
    <div>
      {events.map((event) => (
        <EventRow event={event} key={event.id} />
      ))}
    </div>
  );
}
