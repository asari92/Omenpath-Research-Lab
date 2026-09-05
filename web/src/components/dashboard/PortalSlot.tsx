import { Link } from "react-router-dom";

import type { SlotPortalDTO } from "../../api/types";
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
      <Link to={`/portals/${portal.id}`}>Details</Link>
    </article>
  );
}
