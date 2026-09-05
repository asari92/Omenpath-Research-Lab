import type { PortalViewDTO } from "../../api/types";
import { formatEnergy, formatRemaining } from "../../state/selectors";

export function PortalFacts({ portal }: { portal: PortalViewDTO }) {
  return (
    <section>
      <h2>Portal</h2>
      <dl>
        <div>
          <dt>Name</dt>
          <dd>{portal.name}</dd>
        </div>
        {portal.termination_reason && (
          <div>
            <dt>Termination</dt>
            <dd>{portal.termination_reason.replaceAll("_", " ")}</dd>
          </div>
        )}
        <div>
          <dt>Status</dt>
          <dd
            style={portal.status !== "OPEN" ? { color: "#b7b7b7" } : undefined}
          >
            {portal.status}
          </dd>
        </div>
        <div>
          <dt>Energy</dt>
          <dd>{formatEnergy(portal.energy)}</dd>
        </div>
        <div>
          <dt>Stability</dt>
          <dd>{portal.stability}</dd>
        </div>
        <div>
          <dt>Time Remaining</dt>
          <dd>{formatRemaining(portal.time_remaining_seconds)}</dd>
        </div>
        <div>
          <dt>Observer Flow</dt>
          <dd>{portal.observer_flow}</dd>
        </div>
      </dl>
    </section>
  );
}
