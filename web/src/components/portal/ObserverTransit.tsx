import type { ObserverTransitDTO } from "../../api/types";
import { formatRemaining } from "../../state/selectors";

export function ObserverTransit({
  transit,
}: {
  transit: ObserverTransitDTO | null;
}) {
  if (!transit) return null;
  return (
    <p aria-label="Observer transit" data-direction={transit.direction}>
      Observer #{transit.observer_id} ·{" "}
      {transit.direction === "OUTBOUND" ? "Outbound" : "Returning"} ·{" "}
      {formatRemaining(transit.remaining_seconds)}
    </p>
  );
}
