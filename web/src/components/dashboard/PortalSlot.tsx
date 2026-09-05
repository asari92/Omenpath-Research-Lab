import { Link } from "react-router-dom";

import type { SlotPortalDTO } from "../../api/types";
import { PortalActions } from "../../features/portal-actions/PortalActions";
import { usePortalCommand } from "../../features/portal-actions/usePortalCommand";
import { recordNavigationIntent } from "../../features/tutorial/navigation-signal";
import { PortalEffect } from "../../portal-fx/PortalEffect";
import { formatEnergy, formatRemaining } from "../../state/selectors";
import styles from "./PortalSlot.module.css";

export function PortalSlot({
  slotIndex,
  portal,
  needsAttention,
}: {
  slotIndex: number;
  portal: SlotPortalDTO;
  needsAttention: boolean;
}) {
  const command = usePortalCommand(portal.id);
  return (
    <article className={styles.slot} data-testid="portal-slot">
      <header>
        <span>Slot {slotIndex}</span>
        <span>{portal.stability}</span>
      </header>
      <PortalEffect
        density={needsAttention ? "high" : "low"}
        planeId={portal.destination_plane_id}
        planeName={portal.destination_plane_name}
        portalId={portal.id}
      />
      <h2>{portal.name}</h2>
      <p>{portal.destination_plane_name}</p>
      <dl className={styles.metrics}>
        <div>
          <dt>Energy</dt>
          <dd>{formatEnergy(portal.energy)}</dd>
        </div>
        <div>
          <dt>Time</dt>
          <dd>{formatRemaining(portal.time_remaining_seconds)}</dd>
        </div>
        <div>
          <dt>Creatures</dt>
          <dd>{portal.creatures_inside}</dd>
        </div>
      </dl>
      <PortalActions
        busyKey={command.busyKey}
        onCommand={command.run}
        portalId={portal.id}
        quickActions={portal.quick_actions}
      />
      <Link
        onClick={() =>
          recordNavigationIntent({ kind: "portal", id: portal.id })
        }
        to={`/portals/${portal.id}`}
      >
        Details
      </Link>
    </article>
  );
}
