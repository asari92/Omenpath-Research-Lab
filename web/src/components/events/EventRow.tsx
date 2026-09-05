import type { EventDTO } from "../../api/types";
import { eventTypeLabel } from "../../features/event-log/event-filter";
import styles from "./EventRow.module.css";

export function formatEventTime(iso: string): string {
  const date = new Date(iso);
  if (!Number.isFinite(date.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}-${pad(date.getUTCDate())}-${pad(date.getUTCMonth() + 1)}-${date.getUTCFullYear()}`;
}

export function EventRow({ event }: { event: EventDTO }) {
  const payload =
    event.payload_json === null || event.payload_json === undefined
      ? null
      : JSON.stringify(event.payload_json, null, 2);
  return (
    <article
      className={styles.event}
      data-testid="event-row"
      data-event-type={event.event_type}
    >
      <header>
        <time dateTime={event.created_at}>
          {formatEventTime(event.created_at)}
        </time>
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
