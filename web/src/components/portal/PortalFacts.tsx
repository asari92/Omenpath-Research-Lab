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
        <div>
          <dt>Status</dt>
          <dd>{portal.status}</dd>
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
          <dt>Creatures</dt>
          <dd>{portal.creatures_inside}</dd>
        </div>
        <div>
          <dt>Observer Flow</dt>
          <dd>{portal.observer_flow}</dd>
        </div>
      </dl>
    </section>
  );
}
