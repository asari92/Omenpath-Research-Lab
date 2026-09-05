import { useState } from "react";

import type { EventType } from "../../api/types";
import { eventTypeLabel, eventTypes, type EventFilter } from "./event-filter";
import styles from "./EventFilters.module.css";

const initial: EventFilter = {
  eventTypes: new Set(),
  portalId: null,
  observerId: null,
  planeId: null,
};

function positiveID(value: string): number | null {
  if (value === "") return null;
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}

export function EventFilters({
  onChange,
}: {
  onChange(filter: EventFilter): void;
}) {
  const [filter, setFilter] = useState<EventFilter>(initial);
  const update = (next: EventFilter) => {
    setFilter(next);
    onChange(next);
  };
  const toggle = (type: EventType) => {
    const selected = new Set(filter.eventTypes);
    if (selected.has(type)) selected.delete(type);
    else selected.add(type);
    update({ ...filter, eventTypes: selected });
  };
  return (
    <section className={styles.filters} aria-label="Event filters">
      <details>
        <summary>Event types ({filter.eventTypes.size || "all"})</summary>
        <div className={styles.types}>
          {eventTypes.map((type) => (
            <label key={type}>
              <input
                checked={filter.eventTypes.has(type)}
                onChange={() => toggle(type)}
                type="checkbox"
              />
              {eventTypeLabel(type)}
            </label>
          ))}
        </div>
      </details>
      {(["portalId", "observerId", "planeId"] as const).map((key) => (
        <label key={key}>
          {key === "portalId"
            ? "Portal ID"
            : key === "observerId"
              ? "Observer ID"
              : "Plane ID"}
          <input
            min="1"
            onChange={(event) =>
              update({ ...filter, [key]: positiveID(event.target.value) })
            }
            type="number"
            value={filter[key] ?? ""}
          />
        </label>
      ))}
      <button onClick={() => update(initial)} type="button">
        Clear filters
      </button>
    </section>
  );
}
