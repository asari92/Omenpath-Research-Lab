import type { DestinationDTO } from "../../api/types";

export function DestinationFacts({
  destination,
}: {
  destination: DestinationDTO;
}) {
  return (
    <section>
      <h2>Destination</h2>
      <dl>
        <div>
          <dt>Plane</dt>
          <dd>{destination.name}</dd>
        </div>
        <div>
          <dt>Exploration</dt>
          <dd>{destination.explored ? "EXPLORED" : "UNEXPLORED"}</dd>
        </div>
        <div>
          <dt>Observers Exploring</dt>
          <dd>{destination.observers_exploring}</dd>
        </div>
        <div>
          <dt>Observers Waiting Return</dt>
          <dd>{destination.observers_waiting_return}</dd>
        </div>
        <div>
          <dt>Previous Connections</dt>
          <dd>{destination.previous_connection_count}</dd>
        </div>
      </dl>
    </section>
  );
}
