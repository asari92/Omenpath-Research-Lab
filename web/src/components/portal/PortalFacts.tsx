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
          <dd data-status={portal.status}>{portal.status}</dd>
        </div>
        <div>
          <dt>Energy</dt>
          <dd data-value-kind="energy">{formatEnergy(portal.energy)}</dd>
        </div>
        <div>
          <dt>Stability</dt>
          <dd data-stability={portal.stability}>{portal.stability}</dd>
        </div>
        <div>
          <dt>Time Remaining</dt>
          <dd data-value-kind="time">
            {formatRemaining(portal.time_remaining_seconds)}
          </dd>
        </div>
        <div>
          <dt>Observer Flow</dt>
          <dd>{portal.observer_flow}</dd>
        </div>
      </dl>
    </section>
  );
}
