import type { EventDTO } from "../../api/types";
import { eventTypeLabel } from "../../features/event-log/event-filter";

export function EventRow({ event }: { event: EventDTO }) {
  const payload =
    event.payload_json === null || event.payload_json === undefined
      ? null
      : JSON.stringify(event.payload_json, null, 2);
  return (
    <article data-testid="event-row">
      <header>
        <time dateTime={event.created_at}>{event.created_at}</time>
        <strong>{eventTypeLabel(event.event_type)}</strong>
      </header>
      <p>{event.message}</p>
      {payload && (
        <details>
          <summary>Payload</summary>
          <pre>{payload}</pre>
        </details>
      )}
    </article>
  );
}
